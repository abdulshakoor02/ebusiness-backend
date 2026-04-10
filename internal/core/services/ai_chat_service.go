package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/abdulshakoor02/goCrmBackend/pkg/ai"
	openai "github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AIChatService struct {
	chatClient      *ai.ChatClient
	leadRepo        ports.LeadRepository
	invoiceRepo     ports.InvoiceRepository
	receiptRepo     ports.ReceiptRepository
	appointmentRepo ports.LeadAppointmentRepository
	followUpRepo    ports.LeadFollowUpRepository
}

func NewAIChatService(
	chatClient *ai.ChatClient,
	leadRepo ports.LeadRepository,
	invoiceRepo ports.InvoiceRepository,
	receiptRepo ports.ReceiptRepository,
	appointmentRepo ports.LeadAppointmentRepository,
	followUpRepo ports.LeadFollowUpRepository,
) *AIChatService {
	return &AIChatService{
		chatClient:      chatClient,
		leadRepo:        leadRepo,
		invoiceRepo:     invoiceRepo,
		receiptRepo:     receiptRepo,
		appointmentRepo: appointmentRepo,
		followUpRepo:    followUpRepo,
	}
}

const systemPrompt = `You are a helpful CRM assistant. You can answer questions about the user's CRM data including sales, leads, appointments, invoices, and follow-ups.

When a user asks a question, use the available tools to fetch real data from their CRM database. Always call the appropriate tool(s) before answering — never make up numbers.

After receiving tool results, provide a clear, concise natural language answer. Include specific numbers and be helpful.

If a question is outside the scope of CRM data, politely say so.

Current date: `

func (s *AIChatService) Chat(ctx context.Context, tenantID primitive.ObjectID, req ports.ChatRequest) (*ports.ChatResponse, error) {
	// Build messages
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt + time.Now().Format("January 2, 2006"),
		},
	}

	// Add conversation history
	for _, msg := range req.History {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add current user message
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: req.Message,
	})

	tools := ai.GetChatTools()
	var toolResults []ports.ToolResult

	// LLM call loop: may need multiple rounds if LLM calls multiple tools
	maxRounds := 3
	for round := 0; round < maxRounds; round++ {
		result, err := s.chatClient.CreateChatCompletion(ctx, messages, tools)
		if err != nil {
			slog.Error("AI chat completion failed", "error", err)
			return nil, fmt.Errorf("AI chat failed: %w", err)
		}

		// If no tool calls, return the text answer
		if !result.HasToolCalls() {
			return &ports.ChatResponse{
				Answer:    result.Content,
				ToolCalls: toolResults,
			}, nil
		}

		// Add assistant message with tool calls to conversation
		messages = append(messages, openai.ChatCompletionMessage{
			Role:      openai.ChatMessageRoleAssistant,
			Content:   result.Content,
			ToolCalls: result.ToolCalls,
		})

		// Execute each tool call
		for _, tc := range result.ToolCalls {
			toolResult, err := s.executeTool(ctx, tenantID, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				slog.Error("Tool execution failed", "tool", tc.Function.Name, "error", err)
				toolResult = fmt.Sprintf(`{"error": "failed to execute %s: %s"}`, tc.Function.Name, err.Error())
			}

			toolResults = append(toolResults, ports.ToolResult{
				ToolName: tc.Function.Name,
				Data:     json.RawMessage(toolResult),
			})

			// Add tool result to conversation
			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    toolResult,
				ToolCallID: tc.ID,
			})
		}

		// After tool results, don't send tools again — let LLM generate final answer
		tools = nil
	}

	// If we exhausted rounds, make one final call without tools
	result, err := s.chatClient.CreateFollowUpCompletion(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("AI chat follow-up failed: %w", err)
	}

	return &ports.ChatResponse{
		Answer:    result.Content,
		ToolCalls: toolResults,
	}, nil
}

// executeTool dispatches a tool call to the appropriate Go function.
func (s *AIChatService) executeTool(ctx context.Context, tenantID primitive.ObjectID, name, arguments string) (string, error) {
	switch name {
	case "get_sales_summary":
		return s.executeGetSalesSummary(ctx, tenantID, arguments)
	case "get_appointments_summary":
		return s.executeGetAppointmentsSummary(ctx, tenantID, arguments)
	case "get_leads_summary":
		return s.executeGetLeadsSummary(ctx, tenantID, arguments)
	case "get_invoices_summary":
		return s.executeGetInvoicesSummary(ctx, tenantID, arguments)
	case "get_followups_summary":
		return s.executeGetFollowUpsSummary(ctx, tenantID, arguments)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

type toolArgs struct {
	Month  *int    `json:"month,omitempty"`
	Year   *int    `json:"year,omitempty"`
	Status *string `json:"status,omitempty"`
}

func parseToolArgs(arguments string) (toolArgs, error) {
	var args toolArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return args, fmt.Errorf("invalid arguments: %w", err)
	}
	return args, nil
}

func (s *AIChatService) getDateRange(args toolArgs) (int, int, time.Time, time.Time) {
	now := time.Now()
	month := now.Month()
	year := now.Year()

	if args.Month != nil && *args.Month >= 1 && *args.Month <= 12 {
		month = time.Month(*args.Month)
	}
	if args.Year != nil && *args.Year > 0 {
		year = *args.Year
	}

	startDate := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Second)

	return int(month), year, startDate, endDate
}

func (s *AIChatService) executeGetSalesSummary(ctx context.Context, tenantID primitive.ObjectID, arguments string) (string, error) {
	args, err := parseToolArgs(arguments)
	if err != nil {
		return "", err
	}

	month, year, startDate, endDate := s.getDateRange(args)
	totalPaid, receiptCount, err := s.receiptRepo.SumPaidByDateRange(ctx, tenantID, startDate, endDate)
	if err != nil {
		return "", err
	}

	result := ports.SalesSummary{
		TotalPaid:    totalPaid,
		ReceiptCount: receiptCount,
		Month:        month,
		Year:         year,
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

func (s *AIChatService) executeGetAppointmentsSummary(ctx context.Context, tenantID primitive.ObjectID, arguments string) (string, error) {
	args, err := parseToolArgs(arguments)
	if err != nil {
		return "", err
	}

	month, year, startDate, endDate := s.getDateRange(args)
	status := ""
	if args.Status != nil {
		status = *args.Status
	}

	count, err := s.appointmentRepo.CountByTenantAndDateRange(ctx, tenantID, startDate, endDate, status)
	if err != nil {
		return "", err
	}

	result := ports.AppointmentsSummary{
		TotalCount: count,
		Status:     status,
		Month:      month,
		Year:       year,
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

func (s *AIChatService) executeGetLeadsSummary(ctx context.Context, tenantID primitive.ObjectID, arguments string) (string, error) {
	args, err := parseToolArgs(arguments)
	if err != nil {
		return "", err
	}

	month, year, startDate, endDate := s.getDateRange(args)
	byStatus, err := s.leadRepo.CountByStatusAndDateRange(ctx, tenantID, startDate, endDate)
	if err != nil {
		return "", err
	}

	var totalCount int64
	for _, count := range byStatus {
		totalCount += count
	}

	result := ports.LeadsSummary{
		TotalCount: totalCount,
		ByStatus:   byStatus,
		Month:      month,
		Year:       year,
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

func (s *AIChatService) executeGetInvoicesSummary(ctx context.Context, tenantID primitive.ObjectID, arguments string) (string, error) {
	args, err := parseToolArgs(arguments)
	if err != nil {
		return "", err
	}

	month, year, startDate, endDate := s.getDateRange(args)
	countByStatus, amountByStatus, err := s.invoiceRepo.AggregateByDateRange(ctx, tenantID, startDate, endDate)
	if err != nil {
		return "", err
	}

	var totalCount int64
	var totalAmount float64
	for _, count := range countByStatus {
		totalCount += count
	}
	for _, amount := range amountByStatus {
		totalAmount += amount
	}

	result := ports.InvoicesSummary{
		TotalCount:     totalCount,
		TotalAmount:    totalAmount,
		ByStatus:       countByStatus,
		AmountByStatus: amountByStatus,
		Month:          month,
		Year:           year,
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

func (s *AIChatService) executeGetFollowUpsSummary(ctx context.Context, tenantID primitive.ObjectID, arguments string) (string, error) {
	args, err := parseToolArgs(arguments)
	if err != nil {
		return "", err
	}

	month, year, startDate, endDate := s.getDateRange(args)
	byStatus, err := s.followUpRepo.CountByDateRange(ctx, tenantID, startDate, endDate)
	if err != nil {
		return "", err
	}

	var totalCount int64
	for _, count := range byStatus {
		totalCount += count
	}

	result := ports.FollowUpsSummary{
		TotalCount: totalCount,
		ByStatus:   byStatus,
		Month:      month,
		Year:       year,
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

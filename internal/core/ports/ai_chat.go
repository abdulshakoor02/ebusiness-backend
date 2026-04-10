package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatRequest is the input for the AI chat endpoint.
type ChatRequest struct {
	Message string        `json:"message"`
	History []ChatMessage `json:"history,omitempty"`
}

// ChatMessage represents a single message in the conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse is the output from the AI chat endpoint.
type ChatResponse struct {
	Answer    string       `json:"answer"`
	ToolCalls []ToolResult `json:"tool_calls,omitempty"`
}

// ToolResult shows which tools were called and their data (for transparency).
type ToolResult struct {
	ToolName string      `json:"tool_name"`
	Data     interface{} `json:"data"`
}

// SalesSummary is returned by get_sales_summary tool.
type SalesSummary struct {
	TotalPaid    float64 `json:"total_paid"`
	ReceiptCount int64   `json:"receipt_count"`
	Month        int     `json:"month"`
	Year         int     `json:"year"`
}

// AppointmentsSummary is returned by get_appointments_summary tool.
type AppointmentsSummary struct {
	TotalCount int64  `json:"total_count"`
	Status     string `json:"status,omitempty"`
	Month      int    `json:"month"`
	Year       int    `json:"year"`
}

// LeadsSummary is returned by get_leads_summary tool.
type LeadsSummary struct {
	TotalCount int64            `json:"total_count"`
	ByStatus   map[string]int64 `json:"by_status"`
	Month      int              `json:"month"`
	Year       int              `json:"year"`
}

// InvoicesSummary is returned by get_invoices_summary tool.
type InvoicesSummary struct {
	TotalCount int64              `json:"total_count"`
	TotalAmount float64           `json:"total_amount"`
	ByStatus   map[string]int64   `json:"by_status"`
	AmountByStatus map[string]float64 `json:"amount_by_status"`
	Month      int                `json:"month"`
	Year       int                `json:"year"`
}

// FollowUpsSummary is returned by get_followups_summary tool.
type FollowUpsSummary struct {
	TotalCount int64            `json:"total_count"`
	ByStatus   map[string]int64 `json:"by_status"`
	Month      int              `json:"month"`
	Year       int              `json:"year"`
}

// AIChatService defines the interface for the AI chat feature.
type AIChatService interface {
	Chat(ctx context.Context, tenantID primitive.ObjectID, req ChatRequest) (*ChatResponse, error)
}

package ai

import openai "github.com/sashabaranov/go-openai"

// GetChatTools returns the tool definitions available to the AI chat feature.
// Each tool corresponds to a Go function that queries the database.
func GetChatTools() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_sales_summary",
				Description: "Get total sales/revenue collected for a given time period. Returns total amount paid and number of receipts.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"month": map[string]any{
							"type":        "integer",
							"description": "Month number (1-12). Defaults to current month if not provided.",
							"minimum":     1,
							"maximum":     12,
						},
						"year": map[string]any{
							"type":        "integer",
							"description": "Year (e.g. 2025). Defaults to current year if not provided.",
						},
					},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_appointments_summary",
				Description: "Get number of appointments booked in a given time period, optionally filtered by status.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"month": map[string]any{
							"type":        "integer",
							"description": "Month number (1-12). Defaults to current month if not provided.",
							"minimum":     1,
							"maximum":     12,
						},
						"year": map[string]any{
							"type":        "integer",
							"description": "Year (e.g. 2025). Defaults to current year if not provided.",
						},
						"status": map[string]any{
							"type":        "string",
							"description": "Filter by appointment status.",
							"enum":        []string{"scheduled", "completed", "rescheduled", "cancelled"},
						},
					},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_leads_summary",
				Description: "Get lead counts by status for a given time period. Returns total leads and breakdown by status (lead, active, inactive).",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"month": map[string]any{
							"type":        "integer",
							"description": "Month number (1-12). Defaults to current month if not provided.",
							"minimum":     1,
							"maximum":     12,
						},
						"year": map[string]any{
							"type":        "integer",
							"description": "Year (e.g. 2025). Defaults to current year if not provided.",
						},
					},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_invoices_summary",
				Description: "Get invoice counts and total amounts by status (pending, partial, paid) for a given time period.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"month": map[string]any{
							"type":        "integer",
							"description": "Month number (1-12). Defaults to current month if not provided.",
							"minimum":     1,
							"maximum":     12,
						},
						"year": map[string]any{
							"type":        "integer",
							"description": "Year (e.g. 2025). Defaults to current year if not provided.",
						},
					},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_followups_summary",
				Description: "Get follow-up counts by status (active, closed) for a given time period.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"month": map[string]any{
							"type":        "integer",
							"description": "Month number (1-12). Defaults to current month if not provided.",
							"minimum":     1,
							"maximum":     12,
						},
						"year": map[string]any{
							"type":        "integer",
							"description": "Year (e.g. 2025). Defaults to current year if not provided.",
						},
					},
				},
			},
		},
	}
}

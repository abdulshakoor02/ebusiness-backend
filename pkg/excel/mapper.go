package excel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abdulshakoor02/goCrmBackend/pkg/ai"
)

type ColumnMapping struct {
	ColumnIndex int     `json:"column_index"`
	HeaderName  string  `json:"header_name"`
	TargetField string  `json:"target_field"`
	Transform   string  `json:"transform"`
	Confidence  float64 `json:"confidence"`
	Notes       string  `json:"notes"`
}

type ColumnMappingResult struct {
	Mappings   []ColumnMapping `json:"mappings"`
	Confidence float64         `json:"confidence"`
}

type ReferenceOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ReferenceValueMapping struct {
	InputValue  string  `json:"input_value"`
	MatchedID   string  `json:"matched_id"`
	MatchedName string  `json:"matched_name"`
	Action      string  `json:"action"`
	Confidence  float64 `json:"confidence"`
}

type ReferenceMappingResponse struct {
	Mappings []ReferenceValueMapping `json:"mappings"`
}

var columnAliases = map[string]string{
	"first name":     "first_name",
	"firstname":      "first_name",
	"given name":     "first_name",
	"f name":         "first_name",
	"last name":      "last_name",
	"lastname":       "last_name",
	"surname":        "last_name",
	"family name":    "last_name",
	"l name":         "last_name",
	"full name":      "full_name",
	"name":           "full_name",
	"email":          "email",
	"email address":  "email",
	"e-mail":         "email",
	"mail":           "email",
	"phone":          "phone",
	"mobile":         "phone",
	"telephone":      "phone",
	"tel":            "phone",
	"contact":        "phone",
	"phone number":   "phone",
	"designation":    "designation",
	"title":          "designation",
	"job title":      "designation",
	"position":       "designation",
	"category":       "category_name",
	"lead category":  "category_name",
	"status":         "category_name",
	"lead status":    "category_name",
	"type":           "category_name",
	"lead type":      "category_name",
	"source":         "source_name",
	"lead source":    "source_name",
	"origin":         "source_name",
	"qualification":  "qualification_name",
	"qualifications": "qualification_name",
	"education":      "qualification_name",
	"country":        "country_name",
	"comments":       "comments",
	"notes":          "comments",
	"remarks":        "comments",
}

func MapColumns(ctx context.Context, client *ai.Client, headers []string, sampleRows [][]string) (*ColumnMappingResult, error) {
	systemPrompt := buildColumnSystemPrompt()
	userPrompt := buildColumnUserPrompt(headers, sampleRows)

	response, err := client.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return heuristicColumnMapping(headers), nil
	}

	jsonStr, err := ai.ParseJSONResponse(response)
	if err != nil {
		return heuristicColumnMapping(headers), nil
	}

	var result ColumnMappingResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return heuristicColumnMapping(headers), nil
	}

	if len(result.Mappings) == 0 {
		return heuristicColumnMapping(headers), nil
	}

	return &result, nil
}

func MapReferenceValues(ctx context.Context, client *ai.Client, fieldType string, inputValues []string, existingRefs []ReferenceOption) ([]ReferenceValueMapping, error) {
	systemPrompt := buildRefSystemPrompt()
	userPrompt := buildRefUserPrompt(fieldType, inputValues, existingRefs)

	response, err := client.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return heuristicRefMapping(inputValues, existingRefs), nil
	}

	jsonStr, err := ai.ParseJSONResponse(response)
	if err != nil {
		return heuristicRefMapping(inputValues, existingRefs), nil
	}

	var result ReferenceMappingResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return heuristicRefMapping(inputValues, existingRefs), nil
	}

	if len(result.Mappings) == 0 {
		return heuristicRefMapping(inputValues, existingRefs), nil
	}

	return result.Mappings, nil
}

func heuristicColumnMapping(headers []string) *ColumnMappingResult {
	mappings := make([]ColumnMapping, 0, len(headers))
	for i, h := range headers {
		normalized := normalizeHeader(h)
		if normalized == "full_name" {
			mappings = append(mappings, ColumnMapping{
				ColumnIndex: i,
				HeaderName:  h,
				TargetField: "first_name",
				Transform:   "split_name_first",
				Confidence:  0.7,
				Notes:       "Split full name - first part",
			})
			mappings = append(mappings, ColumnMapping{
				ColumnIndex: i,
				HeaderName:  h,
				TargetField: "last_name",
				Transform:   "split_name_last",
				Confidence:  0.7,
				Notes:       "Split full name - last part",
			})
		} else {
			mappings = append(mappings, ColumnMapping{
				ColumnIndex: i,
				HeaderName:  h,
				TargetField: normalized,
				Confidence:  0.8,
			})
		}
	}
	return &ColumnMappingResult{Mappings: mappings, Confidence: 0.7}
}

func normalizeHeader(header string) string {
	normalized := strings.ToLower(strings.TrimSpace(header))
	for alias, canonical := range columnAliases {
		if strings.Contains(normalized, alias) {
			return canonical
		}
	}
	return strings.ReplaceAll(normalized, " ", "_")
}

func heuristicRefMapping(inputValues []string, existingRefs []ReferenceOption) []ReferenceValueMapping {
	mappings := make([]ReferenceValueMapping, 0, len(inputValues))
	existingByName := make(map[string]ReferenceOption)
	for _, ref := range existingRefs {
		existingByName[strings.ToLower(ref.Name)] = ref
	}

	for _, val := range inputValues {
		lower := strings.ToLower(strings.TrimSpace(val))
		if ref, ok := existingByName[lower]; ok {
			mappings = append(mappings, ReferenceValueMapping{
				InputValue:  val,
				MatchedID:   ref.ID,
				MatchedName: ref.Name,
				Action:      "use_existing",
				Confidence:  1.0,
			})
		} else {
			mappings = append(mappings, ReferenceValueMapping{
				InputValue:  val,
				MatchedName: val,
				Action:      "create_new",
				Confidence:  0.5,
			})
		}
	}
	return mappings
}

func buildColumnSystemPrompt() string {
	return `You are a data mapping assistant for a CRM system. Map Excel columns to CRM fields by analyzing BOTH the column header names AND the sample data values.

TARGET FIELDS:
- first_name (string) - Person's first name
- last_name (string) - Person's last name
- full_name (string) - Full name (will be split into first/last)
- email (string) - Email address
- phone (string) - Phone number
- designation (string) - Job title/position
- category_name (reference) - Lead category/type (values like: Hot Lead, Warm Lead, Cold Lead, VIP, New, etc.)
- source_name (reference) - How they found us (values like: Website, Facebook, Referral, Google, etc.)
- qualification_name (reference) - Education level (values like: Bachelor, Master, PhD, etc.)
- country_name (reference) - Country name
- comments (string) - Free text notes

IMPORTANT DISAMBIGUATION RULES:
1. Look at BOTH the header name AND sample values to determine the correct mapping
2. "Lead Status" or "Status" columns with category-like values (Hot Lead, Warm Lead, Cold Lead, New, VIP, etc.) map to category_name
3. "Lead Source" or "Source" columns map to source_name
4. "Lead Type" or "Type" columns with category-like values map to category_name
5. "Qualifications" or "Education" columns map to qualification_name
6. "Preferred Country" or "Country" columns map to country_name
7. "Designation" or "Role" columns map to designation
8. Do NOT skip columns - if a column has values, it should be mapped
9. If unsure between category_name and source_name, look at the values: category = lead type/quality, source = where they came from

EXAMPLES:
- Header: "Lead Status", Values: ["Hot Lead", "Warm Lead", "Cold Lead"] → category_name
- Header: "Status", Values: ["Hot", "Warm", "Cold"] → category_name
- Header: "Lead Type", Values: ["VIP", "Regular", "New"] → category_name
- Header: "Lead Source", Values: ["Website", "Facebook", "Referral"] → source_name
- Header: "Source", Values: ["Google", "Friend"] → source_name
- Header: "Qualifications", Values: ["Bachelor", "Master"] → qualification_name
- Header: "Preferred Country", Values: ["UAE", "USA"] → country_name

RESPONSE FORMAT: Return ONLY valid JSON:
{"mappings": [{"column_index": 0, "header_name": "...", "target_field": "...", "transform": "none|split_name", "confidence": 0.9, "notes": "..."}], "confidence": 0.95}

Use column_index (0-based) to identify columns, NOT the header name.`
}

func buildColumnUserPrompt(headers []string, sampleRows [][]string) string {
	var b strings.Builder
	b.WriteString("EXCEL COLUMNS (with index):\n")
	for i, h := range headers {
		fmt.Fprintf(&b, "[%d] %s\n", i, h)
	}
	b.WriteString("\nSAMPLE ROWS:\n")
	for i, row := range sampleRows {
		fmt.Fprintf(&b, "Row %d: %s\n", i+1, strings.Join(row, " | "))
	}
	b.WriteString("\nAnalyze both the column headers AND the sample values to determine the correct mapping.")
	return b.String()
}

func buildRefSystemPrompt() string {
	return `You are a reference value mapping assistant. Map input values to existing database records.

RULES:
1. Match case-insensitively to existing records
2. If no match, mark for creation
3. Consider common abbreviations (e.g., "UAE" = "United Arab Emirates")

RESPONSE FORMAT: Return ONLY valid JSON:
{"mappings": [{"input_value": "...", "matched_id": "..." or "", "matched_name": "...", "action": "use_existing|create_new|skip", "confidence": 0.9}]}`
}

func buildRefUserPrompt(fieldType string, inputValues []string, existingRefs []ReferenceOption) string {
	var b strings.Builder
	fmt.Fprintf(&b, "FIELD TYPE: %s\n\nINPUT VALUES:\n", fieldType)
	for _, v := range inputValues {
		fmt.Fprintf(&b, "- %s\n", v)
	}
	b.WriteString("\nEXISTING RECORDS:\n")
	for _, ref := range existingRefs {
		fmt.Fprintf(&b, "- %s (ID: %s)\n", ref.Name, ref.ID)
	}
	return b.String()
}

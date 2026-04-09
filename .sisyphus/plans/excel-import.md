# File Import Utility - AI-Powered Lead Import (.xlsx + .csv)

## TL;DR

> **Quick Summary**: Add a file import feature that accepts .xlsx and .csv uploads, uses AI (Chutes provider) to smartly map varying column names and reference values, auto-creates missing categories/sources, maps country (defaults to UAE), handles assigned_to with role-based permissions, and bulk inserts leads into MongoDB with partial success reporting.
> 
> **Deliverables**:
> - File parser package (`pkg/excel/parser.go`) — supports both .xlsx and .csv
> - AI client package (`pkg/ai/client.go`) 
> - Column/reference mapper (`pkg/excel/mapper.go`)
> - New repository methods (FindByName for category/source/country, BulkInsert)
> - Import service method + handler endpoint with permission-based assignedTo logic
> 
> **Estimated Effort**: Medium
> **Parallel Execution**: YES - 3 waves
> **Critical Path**: Config → AI Client → Mapper → Service → Handler → Route

---

## Context

### Original Request
User wants to add a file import utility where users upload .xlsx or .csv files, the backend reads data using AI to intelligently map varying column formats, resolves reference fields (category, source, status) against database records or auto-creates new ones, then bulk inserts leads with error reporting.

### Interview Summary
**Key Discussions**:
- In-memory processing (no file storage on disk)
- AI provider: Chutes with custom LLM URL (OpenAI-compatible format)
- Simple HTTP client (not SDK) for maximum flexibility
- Two AI calls per import: (1) map headers, (2) map reference values
- Auto-create missing categories/sources; qualifications must exist
- Error tolerance: skip bad rows, insert valid ones, report all errors
- Supported formats: .xlsx (excelize/v2) and .csv (encoding/csv standard lib)
- .xls NOT supported (legacy format, poor Go library support)

**Country Mapping**:
- If country column exists, map by name to existing countries in DB (case-insensitive)
- If no country column or empty value, default to "United Arab Emirates"
- Country is global (no tenant_id), lookup via FindByName on CountryRepository

**AssignedTo + Permissions**:
- **Admin role**: Can select who to assign leads to via optional `assigned_to` parameter in API
- **User with import permission**: assigned_to always defaults to their own user_id
- **User without import permission**: Cannot import (403 Forbidden)
- Permission checked via existing RBAC system (path + method based)

**Research Findings**:
- excelize/v2 supports .xlsx only (NOT .xls - Excel 97-2003)
- Go standard lib `encoding/csv` handles CSV parsing natively
- Same AI mapping logic works for both formats (headers are headers)
- Chutes API is OpenAI-compatible (POST /v1/chat/completions)
- MongoDB bulk insert with ordered:false for performance
- Existing repos lack FindByName for country and BulkInsert for leads
- Country domain is global (no TenantID field)
- Permission system uses path+method with role-based access

### Metis Review
**Identified Gaps** (addressed):
- XLS format not supported by excelize → Reject with clear error
- CSV support added via standard lib encoding/csv
- Country mapping → FindByName + default to UAE
- AssignedTo mapping → Role-based: admin chooses, user defaults to self
- Permission check → Use existing RBAC system for import access
- AI response validation → Fallback to heuristic if AI fails
- File/row limits needed → 10MB max, 10,000 rows max
- Duplicate handling → Report as error, skip row

---

## Work Objectives

### Core Objective
Enable users to bulk import leads from Excel files with intelligent AI-powered column mapping that handles varying spreadsheet formats automatically.

### Concrete Deliverables
- `pkg/excel/parser.go` — File parsing with excelize (.xlsx) and encoding/csv (.csv)
- `pkg/excel/mapper.go` — AI-assigned column and reference value mapping
- `pkg/ai/client.go` — HTTP client for Chutes AI provider
- `internal/adapters/storage/mongo_lead_repo.go` — BulkInsert method
- `internal/adapters/storage/mongo_lead_category_repo.go` — FindByName method
- `internal/adapters/storage/mongo_lead_source_repo.go` — FindByName method
- `internal/adapters/storage/mongo_qualification_repo.go` — FindByName method
- `internal/adapters/storage/mongo_country_repo.go` — FindByName method
- `internal/core/services/lead_service.go` — ImportLeads method (with permission + assignedTo logic)
- `internal/adapters/handler/lead_handler.go` — ImportLeads endpoint (with optional assigned_to param)
- `internal/core/ports/repositories.go` — Updated interfaces
- `internal/core/ports/services.go` — Import types + updated interface
- `config/config.go` — AI configuration fields
- `cmd/api/main.go` — Route registration

### Definition of Done
- [ ] POST /api/v1/leads/import accepts multipart file upload + optional assigned_to form field
- [ ] .xlsx files are parsed in memory (no disk storage)
- [ ] .csv files are parsed in memory (no disk storage)
- [ ] .xls files are rejected with clear error message
- [ ] AI maps column headers to Lead fields correctly
- [ ] AI maps reference values to existing DB records
- [ ] Missing categories/sources are auto-created (tenant-scoped)
- [ ] Country mapped by name or defaults to "United Arab Emirates"
- [ ] Admin can specify assigned_to user, user's assigned_to defaults to self
- [ ] Permission check: users without import permission get 403
- [ ] Response includes inserted count, skipped count, errors array
- [ ] Files > 10MB are rejected
- [ ] Files > 10,000 rows are rejected

### Must Have
- In-memory processing (no temp files)
- AI-powered column mapping via Chutes
- Partial success (insert valid rows, report errors)
- Tenant isolation on all operations
- Auto-create missing categories/sources
- Country mapping with UAE default
- Role-based assignedTo (admin chooses, user defaults to self)
- Permission check for import access
- Detailed error reporting with row numbers

### Must NOT Have (Guardrails)
- File storage on disk
- Per-row AI calls (only structure-level mapping)
- Silent failures (every skip must be reported)
- Cross-tenant data leakage
- Auto-creation of qualifications (must exist in DB)
- .xls format support (only .xlsx and .csv)
- Auto-creation of countries (must exist in DB, default to UAE if missing)

---

## Verification Strategy (MANDATORY)

### Test Decision
- **Infrastructure exists**: YES (bun test / testify)
- **Automated tests**: Tests after implementation
- **Framework**: Go testify (existing)

### QA Policy
Every task includes agent-executed QA scenarios using Bash (curl) for API endpoints.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — all parallel, no dependencies):
├── Task 1: Config — Add AI fields to config.go
├── Task 2: AI Client — pkg/ai/client.go
├── Task 3: File Parser — pkg/excel/parser.go (.xlsx + .csv)
├── Task 4: Category Repo — Add FindByName to mongo_lead_category_repo.go
├── Task 5: Source Repo — Add FindByName to mongo_lead_source_repo.go
├── Task 6: Qualification Repo — Add FindByName to mongo_qualification_repo.go
├── Task 7: Country Repo — Add FindByName to mongo_country_repo.go
└── Task 8: Lead Repo — Add BulkInsert to mongo_lead_repo.go

Wave 2 (After Wave 1 — core logic):
├── Task 9: Mapper — pkg/excel/mapper.go (AI column + reference mapping)
├── Task 10: Port Types — Add ImportLeadsRequest, ImportResult to ports/services.go
├── Task 11: Repo Interfaces — Add FindByName + BulkInsert to ports/repositories.go
└── Task 12: Service — Add ImportLeads to lead_service.go (depends: 9, 10, 11)

Wave 3 (After Wave 2 — wiring):
├── Task 13: Handler — Add ImportLeads to lead_handler.go (with permission + assigned_to)
└── Task 14: Route — Register /leads/import in main.go

Critical Path: Task 1 → Task 2 → Task 9 → Task 12 → Task 13 → Task 14
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 8 (Wave 1)
```

### Dependency Matrix
- **1**: None → 2, 9
- **2**: None → 9
- **3**: None → 9
- **4**: None → 11
- **5**: None → 11
- **6**: None → 11
- **7**: None → 11
- **8**: None → 11, 12
- **9**: 1, 2, 3 → 12
- **10**: None → 12
- **11**: 4, 5, 6, 7, 8 → 12
- **12**: 9, 10, 11 → 13
- **13**: 12 → 14
- **14**: 13 → Done

### Agent Dispatch Summary
- **Wave 1**: 8 tasks — T1-T8 → `quick` or `unspecified-low`
- **Wave 2**: 4 tasks — T9 → `deep`, T10-T11 → `quick`, T12 → `unspecified-high`
- **Wave 3**: 2 tasks — T13 → `quick`, T14 → `quick`

---

## TODOs

- [ ] 1. Config — Add AI configuration fields

  **What to do**:
  - Add to `config/config.go` Config struct: `AIURL`, `AIAPIKey`, `AIModel` (string fields)
  - Add to `LoadConfig()`: read from env vars `AI_URL`, `AI_API_KEY`, `AI_MODEL` with defaults
  - Add `MaxImportFileSize` (int64, default 10MB) and `MaxImportRows` (int64, default 10000)

  **Must NOT do**:
  - Do not add per-tenant AI config (global only)
  - Do not modify existing config fields

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple config field additions
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2-7)
  - **Blocks**: Task 2 (AI client needs config)
  - **Blocked By**: None

  **References**:
  - `config/config.go:9-21` — Existing Config struct pattern
  - `config/config.go:23-45` — LoadConfig() pattern with getEnv fallback

  **Acceptance Criteria**:
  - [ ] Config struct has AIURL, AIAPIKey, AIModel, MaxImportFileSize, MaxImportRows fields
  - [ ] LoadConfig reads from env vars with sensible defaults
  - [ ] go build passes without errors

  **QA Scenarios**:
  ```
  Scenario: Config loads AI fields from env
    Tool: Bash
    Steps:
      1. Set env: AI_URL=https://test.chutes.ai/v1/chat/completions AI_API_KEY=test123 AI_MODEL=gpt-4
      2. Run: go run cmd/api/main.go (briefly to check no panic)
    Expected Result: Server starts without config-related errors
    Evidence: .sisyphus/evidence/task-1-config-loads.txt

  Scenario: Config uses defaults when env not set
    Tool: Bash
    Steps:
      1. Unset AI_URL, AI_API_KEY, AI_MODEL
      2. Run: go build ./...
    Expected Result: Build succeeds with default values
    Evidence: .sisyphus/evidence/task-1-config-defaults.txt
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `feat(config): add AI provider and import limit configuration`
  - Files: `config/config.go`

---

- [ ] 2. AI Client — HTTP client for Chutes provider

  **What to do**:
  - Create `pkg/ai/client.go` with struct `Client { baseURL, apiKey, model, httpClient }`
  - Constructor: `NewClient(baseURL, apiKey, model string) *Client`
  - Method: `Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error)`
  - HTTP POST to `{baseURL}` with OpenAI-compatible format: `{ model, messages: [{role, content}] }`
  - Auth header: `Authorization: Bearer {apiKey}`
  - Response parsing: extract `choices[0].message.content`
  - Error handling: timeout (30s), retry on 429 (exponential backoff), parse error messages
  - JSON response extraction helper: `ParseJSONResponse(raw string) (string, error)` to handle markdown code blocks

  **Must NOT do**:
  - Do not use go-openai SDK (simple HTTP for maximum flexibility)
  - Do not add streaming support
  - Do not add function calling support

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
    - Reason: HTTP client with JSON parsing, straightforward
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3-7)
  - **Blocks**: Task 8 (mapper needs AI client)
  - **Blocked By**: None (reads config values, doesn't depend on Task 1 structurally)

  **References**:
  - `pkg/utils/jwt.go` — Existing pkg utility pattern
  - Chutes API format: POST /v1/chat/completions, OpenAI-compatible messages array

  **Acceptance Criteria**:
  - [ ] Client struct with NewClient constructor
  - [ ] Chat method sends correct HTTP request format
  - [ ] JSON response extraction handles markdown code blocks
  - [ ] 30-second timeout configured
  - [ ] go build passes

  **QA Scenarios**:
  ```
  Scenario: AI client sends correct request format
    Tool: Bash (unit test)
    Steps:
      1. Write test that starts mock HTTP server
      2. Create client pointing to mock server
      3. Call Chat() with test prompts
      4. Assert mock received correct JSON body and headers
    Expected Result: Request body has model, messages array; Authorization header present
    Evidence: .sisyphus/evidence/task-2-ai-client-format.txt

  Scenario: AI client handles JSON response extraction
    Tool: Bash (unit test)
    Steps:
      1. Test ParseJSONResponse with raw JSON
      2. Test ParseJSONResponse with markdown-wrapped JSON
      3. Test ParseJSONResponse with invalid input
    Expected Result: Extracts JSON from all formats, returns error for invalid
    Evidence: .sisyphus/evidence/task-2-json-extraction.txt
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `feat(ai): add HTTP client for Chutes AI provider`
  - Files: `pkg/ai/client.go`

---

- [ ] 3. File Parser — Parse .xlsx and .csv files in memory

  **What to do**:
  - Create `pkg/excel/parser.go`
  - Function: `ParseFile(data []byte, ext string) (headers []string, rows [][]string, err error)`
  - **For .xlsx**: Use excelize to open from bytes: `excelize.OpenReader(bytes.NewReader(data))`
    - Get first sheet: `f.GetSheetList()[0]`
    - Stream rows using `f.Rows(sheetName)` iterator
    - First row = headers, remaining rows = data
    - Always call `defer rows.Close()` and `defer f.Close()`
  - **For .csv**: Use `encoding/csv` standard library
    - `csv.NewReader(bytes.NewReader(data))`
    - First Read() = headers
    - Remaining Read() = data rows
    - Handle comma, semicolon, and tab delimiters (auto-detect)
  - Add go.mod dependency: `github.com/xuri/excelize/v2`

  **Must NOT do**:
  - Do not support .xls format (only .xlsx and .csv)
  - Do not process multiple sheets (first sheet only for xlsx)
  - Do not write temp files

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single-purpose parser using well-documented library
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-2, 4-7)
  - **Blocks**: Task 8 (mapper needs parser output format)
  - **Blocked By**: None

  **References**:
  - excelize docs: `excelize.OpenReader(r io.Reader)` for in-memory parsing
  - excelize docs: `Rows()` returns streaming iterator for memory efficiency
  - Existing go.mod: dependencies use `require` block

  **Acceptance Criteria**:
  - [ ] ParseFile function accepts []byte and extension, returns headers + rows
  - [ ] Supports .xlsx via excelize streaming iterator
  - [ ] Supports .csv via encoding/csv standard lib
  - [ ] Auto-detects CSV delimiter (comma, semicolon, tab)
  - [ ] excelize/v2 added to go.mod
  - [ ] go build passes

  **QA Scenarios**:
  ```
  Scenario: Parse valid xlsx file
    Tool: Bash (unit test)
    Steps:
      1. Create test xlsx file with 3 columns, 5 rows
      2. Call ParseFile(fileBytes, ".xlsx")
      3. Assert headers = ["Name", "Email", "Phone"]
      4. Assert rows length = 4 (excluding header)
    Expected Result: Correct headers and data rows extracted
    Evidence: .sisyphus/evidence/task-3-parse-xlsx-valid.txt

  Scenario: Parse valid csv file
    Tool: Bash (unit test)
    Steps:
      1. Create test csv content: "Name,Email,Phone\nJohn,john@test.com,+1234"
      2. Call ParseFile(csvBytes, ".csv")
      3. Assert headers = ["Name", "Email", "Phone"]
      4. Assert rows length = 1
    Expected Result: Correct headers and data rows extracted
    Evidence: .sisyphus/evidence/task-3-parse-csv-valid.txt

  Scenario: Parse empty file
    Tool: Bash (unit test)
    Steps:
      1. Create empty xlsx file
      2. Call ParseFile(emptyBytes, ".xlsx")
    Expected Result: Returns empty headers and rows, no error
    Evidence: .sisyphus/evidence/task-3-parse-empty.txt
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `feat(excel): add in-memory xlsx and csv parser with streaming`
  - Files: `pkg/excel/parser.go`, `go.mod`, `go.sum`

---

- [ ] 4. Category Repo — Add FindByName method

  **What to do**:
  - Add to `mongo_lead_category_repo.go`: `FindByName(ctx context.Context, tenantID primitive.ObjectID, name string) (*domain.LeadCategory, error)`
  - Query: `{ tenant_id: tenantID, name: { $regex: "^name$", $options: "i" } }` (case-insensitive exact match)
  - Returns `mongo.ErrNoDocuments` wrapped as "category not found" if no match
  - Add to `ports/repositories.go` LeadCategoryRepository interface: `FindByName(ctx, tenantID, name) (*domain.LeadCategory, error)`

  **Must NOT do**:
  - Do not use scope filter middleware (tenantID passed directly for import context)
  - Do not modify existing methods

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single method addition following existing pattern
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-3, 5-7)
  - **Blocks**: Task 10 (repo interface needs this method)
  - **Blocked By**: None

  **References**:
  - `mongo_lead_category_repo.go:32-50` — GetByID pattern for filter + decode
  - `mongo_lead_category_repo.go:17-25` — Constructor pattern

  **Acceptance Criteria**:
  - [ ] FindByName method exists on MongoLeadCategoryRepository
  - [ ] Case-insensitive exact match using regex
  - [ ] Returns proper error when not found

  **QA Scenarios**:
  ```
  Scenario: FindByName returns existing category
    Tool: Bash (unit test)
    Steps:
      1. Insert test category "Hot Lead" for tenant
      2. Call FindByName(tenantID, "hot lead")
    Expected Result: Returns the category with correct ID
    Evidence: .sisyphus/evidence/task-4-findbyname-exists.txt
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `feat(storage): add FindByName to lead category repository`
  - Files: `internal/adapters/storage/mongo_lead_category_repo.go`

---

- [ ] 5. Source Repo — Add FindByName method

  **What to do**:
  - Add to `mongo_lead_source_repo.go`: `FindByName(ctx context.Context, tenantID primitive.ObjectID, name string) (*domain.LeadSource, error)`
  - Same pattern as Task 4: case-insensitive regex match scoped to tenant
  - Add to `ports/repositories.go` LeadSourceRepository interface

  **Must NOT do**:
  - Do not use scope filter middleware (tenantID passed directly)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Mirror of Task 4 for source repo
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-4, 6-7)
  - **Blocks**: Task 10
  - **Blocked By**: None

  **References**:
  - `mongo_lead_source_repo.go:31-49` — GetByID pattern
  - Task 4 (this is a mirror for sources)

  **Acceptance Criteria**:
  - [ ] FindByName method exists on MongoLeadSourceRepository
  - [ ] Case-insensitive exact match
  - [ ] Tenant-scoped query

  **QA Scenarios**:
  ```
  Scenario: FindByName returns existing source
    Tool: Bash (unit test)
    Steps:
      1. Insert test source "Website" for tenant
      2. Call FindByName(tenantID, "Website")
    Expected Result: Returns the source with correct ID
    Evidence: .sisyphus/evidence/task-5-findbyname-exists.txt
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `feat(storage): add FindByName to lead source repository`
  - Files: `internal/adapters/storage/mongo_lead_source_repo.go`

---

- [ ] 6. Qualification Repo — Add FindByName method

  **What to do**:
  - Add to `mongo_qualification_repo.go`: `FindByName(ctx context.Context, name string) (*domain.Qualification, error)`
  - Note: Qualification is NOT tenant-scoped (global), so no tenantID parameter
  - Same case-insensitive regex pattern
  - Add to `ports/repositories.go` QualificationRepository interface

  **Must NOT do**:
  - Do not add tenant_id filter (qualifications are global)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Mirror of Task 4/5 for qualification (simpler, no tenant filter)
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-5, 7)
  - **Blocks**: Task 10
  - **Blocked By**: None

  **References**:
  - `qualification.go` domain — No TenantID field (global entity)
  - Existing QualificationRepository interface in repositories.go

  **Acceptance Criteria**:
  - [ ] FindByName method exists on MongoQualificationRepository
  - [ ] No tenant_id filter (global lookup)
  - [ ] Case-insensitive match

  **QA Scenarios**:
  ```
  Scenario: FindByName returns existing qualification
    Tool: Bash (unit test)
    Steps:
      1. Insert test qualification "Bachelor"
      2. Call FindByName("bachelor")
    Expected Result: Returns the qualification with correct ID
    Evidence: .sisyphus/evidence/task-6-findbyname-exists.txt
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `feat(storage): add FindByName to qualification repository`
  - Files: `internal/adapters/storage/mongo_qualification_repo.go`

---

- [ ] 7. Country Repo — Add FindByName method

  **What to do**:
  - Add to `mongo_country_repo.go`: `FindByName(ctx context.Context, name string) (*domain.Country, error)`
  - Country is global (no TenantID), same pattern as qualification
  - Case-insensitive exact match using regex: `{ name: { $regex: "^name$", $options: "i" } }`
  - Also add: `GetDefaultCountry(ctx context.Context) (*domain.Country, error)` — finds UAE by name
  - Add to `ports/repositories.go` CountryRepository interface

  **Must NOT do**:
  - Do not add tenant_id filter (countries are global)
  - Do not auto-create countries

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple FindByName method, mirror of qualification pattern
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-8)
  - **Blocks**: Task 11
  - **Blocked By**: None

  **References**:
  - `internal/core/domain/country.go` — Country struct (no TenantID, global entity)
  - `internal/adapters/storage/mongo_country_repo.go` — Existing repo structure
  - `internal/core/ports/repositories.go:109-115` — CountryRepository interface

  **Acceptance Criteria**:
  - [ ] FindByName method exists on MongoCountryRepository
  - [ ] Case-insensitive exact match
  - [ ] No tenant_id filter (global)
  - [ ] go build passes

  **QA Scenarios**:
  ```
  Scenario: FindByName returns existing country
    Tool: Bash (unit test)
    Steps:
      1. Ensure "United Arab Emirates" exists in countries collection
      2. Call FindByName("united arab emirates")
    Expected Result: Returns country with correct ID
    Evidence: .sisyphus/evidence/task-7-findbyname-country.txt
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `feat(storage): add FindByName to country repository`
  - Files: `internal/adapters/storage/mongo_country_repo.go`

---

- [ ] 8. Lead Repo — Add BulkInsert method

  **What to do**:
  - Add to `mongo_lead_repo.go`: `BulkInsert(ctx context.Context, leads []*domain.Lead) (int, error)`
  - Use `mongo.InsertMany` with `options.InsertMany().SetOrdered(false)` for parallel inserts
  - Convert `[]*domain.Lead` to `[]interface{}` for InsertMany
  - Return count of successfully inserted documents
  - Add to `ports/repositories.go` LeadRepository interface: `BulkInsert(ctx, leads) (int, error)`

  **Must NOT do**:
  - Do not use ordered:true (slower, fails on first error)
  - Do not add deduplication logic (service layer handles that)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single method using existing mongo.Collection
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
- **Parallel Group**: Wave 1 (with Tasks 1-7)
- **Blocks**: Task 11, Task 12
- **Blocked By**: None

  **References**:
  - `mongo_lead_repo.go:29-31` — Existing Create method using InsertOne
  - `mongo_lead_repo.go:19-27` — Constructor and collection access
  - MongoDB InsertMany docs — SetOrdered(false) for parallel

  **Acceptance Criteria**:
  - [ ] BulkInsert method exists on MongoLeadRepository
  - [ ] Uses InsertMany with ordered:false
  - [ ] Returns count of inserted documents

  **QA Scenarios**:
  ```
  Scenario: BulkInsert inserts multiple leads
    Tool: Bash (unit test)
    Steps:
      1. Create 5 test Lead structs
      2. Call BulkInsert
      3. Count documents in collection
    Expected Result: 5 documents inserted, count returned = 5
    Evidence: .sisyphus/evidence/task-7-bulkinsert.txt
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `feat(storage): add BulkInsert to lead repository`
  - Files: `internal/adapters/storage/mongo_lead_repo.go`

---

- [ ] 9. Mapper — AI column + reference value mapping

  **What to do**:
  - Create `pkg/excel/mapper.go`
  - **Column Mapping**:
    - `MapColumns(ctx context.Context, ai *ai.Client, headers []string, sampleRow []string) (*ColumnMappingResult, error)`
    - Build system prompt with target fields list and descriptions
    - Build user prompt with headers + sample row
    - Parse AI response as JSON with mappings array
    - Each mapping: `{ excel_column, target_field, confidence, notes }`
    - Target fields: first_name, last_name, email, phone, designation, category_name, source_name, qualification_name, country_name, comments
    - Handle "split_name" transform for full name columns
  - **Reference Value Mapping**:
    - `MapReferenceValues(ctx context.Context, ai *ai.Client, fieldType string, inputValues []string, existingRefs []ReferenceOption) ([]ReferenceValueMapping, error)`
    - Build prompt with input values + existing DB records
    - Parse response: each value maps to existing ID or "create_new"
  - **Heuristic Fallback**:
    - If AI fails, fall back to simple string matching (lowercase, trim, alias map)
    - Basic aliases: "full name" → split to first/last, "email address" → email, etc.

  **Must NOT do**:
  - Do not call AI per-row (only per-import on structure)
  - Do not cache mappings (stateless per-request)

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Complex logic combining AI prompts, JSON parsing, heuristic fallback
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 9-10)
  - **Blocks**: Task 11 (service needs mapper)
  - **Blocked By**: Tasks 2, 3 (needs AI client and parser output format)

  **References**:
  - `pkg/ai/client.go` (Task 2) — AI client Chat method
  - `pkg/excel/parser.go` (Task 3) — Parser output format (headers, rows)
  - `internal/core/ports/services.go:176-188` — CreateLeadRequest fields for target field list
  - `internal/core/domain/lead.go:24-31` — Lead struct fields

  **Acceptance Criteria**:
  - [ ] MapColumns function exists and returns structured mapping result
  - [ ] MapReferenceValues function exists for category/source/qualification
  - [ ] Heuristic fallback works when AI is unavailable
  - [ ] JSON parsing handles markdown code blocks
  - [ ] go build passes

  **QA Scenarios**:
  ```
  Scenario: AI maps common column variants
    Tool: Bash (unit test with mock AI)
    Steps:
      1. Mock AI returning correct mapping JSON
      2. Call MapColumns with headers ["Full Name", "Email Address", "Phone Number"]
      3. Assert mappings split "Full Name" into first_name/last_name
    Expected Result: Correct field mappings with confidence scores
    Evidence: .sisyphus/evidence/task-8-column-mapping.txt

  Scenario: Fallback works when AI fails
    Tool: Bash (unit test)
    Steps:
      1. Mock AI returning error
      2. Call MapColumns with headers ["email", "first name", "last name"]
      3. Assert heuristic produces reasonable mappings
    Expected Result: Basic alias matching produces usable mappings
    Evidence: .sisyphus/evidence/task-8-fallback.txt
  ```

  **Commit**: YES (groups with Wave 2)
  - Message: `feat(excel): add AI-powered column and reference value mapping`
  - Files: `pkg/excel/mapper.go`

---

- [ ] 10. Port Types — Add import request/response types

  **What to do**:
  - Add to `internal/core/ports/services.go`:
    - `ImportResult` struct: `{ TotalRows int, Inserted int, Skipped int, CreatedCategories []string, CreatedSources []string, Errors []ImportError }`
    - `ImportError` struct: `{ Row int, Field string, Value string, Reason string }`
  - Add to `LeadService` interface: `ImportLeads(ctx context.Context, data []byte, ext string, assignedTo string) (*ImportResult, error)`

  **Must NOT do**:
  - Do not modify existing request/response types
  - Do not add ImportLeadsRequest (data passed directly as []byte)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple struct additions to existing file
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 9, 11)
  - **Blocks**: Task 12
  - **Blocked By**: None

  **References**:
  - `internal/core/ports/services.go:249-255` — Existing LeadService interface
  - `internal/core/ports/services.go:60-65` — Existing FilterRequest pattern
  - `internal/core/ports/services.go:176-188` — CreateLeadRequest fields

  **Acceptance Criteria**:
  - [ ] ImportResult struct with all fields
  - [ ] ImportError struct with row, field, value, reason
  - [ ] ImportLeads method added to LeadService interface
  - [ ] go build passes

  **QA Scenarios**:
  ```
  Scenario: Types compile correctly
    Tool: Bash
    Steps:
      1. Run: go build ./internal/core/ports/
    Expected Result: Build succeeds with new types
    Evidence: .sisyphus/evidence/task-9-types-compile.txt
  ```

  **Commit**: YES (groups with Wave 2)
  - Message: `feat(ports): add import result types and ImportLeads interface method`
  - Files: `internal/core/ports/services.go`

---

- [ ] 11. Repo Interfaces — Add new methods to repository interfaces

  **What to do**:
  - Add to `ports/repositories.go`:
    - `LeadCategoryRepository`: `FindByName(ctx context.Context, tenantID primitive.ObjectID, name string) (*domain.LeadCategory, error)`
    - `LeadSourceRepository`: `FindByName(ctx context.Context, tenantID primitive.ObjectID, name string) (*domain.LeadSource, error)`
    - `QualificationRepository`: `FindByName(ctx context.Context, name string) (*domain.Qualification, error)`
    - `CountryRepository`: `FindByName(ctx context.Context, name string) (*domain.Country, error)`
    - `LeadRepository`: `BulkInsert(ctx context.Context, leads []*domain.Lead) (int, error)`

  **Must NOT do**:
  - Do not modify existing interface methods
  - Do not add methods to other repositories

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Interface additions only
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 9-10)
  - **Blocks**: Task 12
  - **Blocked By**: Tasks 4, 5, 6, 7, 8 (methods must exist in implementations)

  **References**:
  - `internal/core/ports/repositories.go:33-39` — LeadCategoryRepository
  - `internal/core/ports/repositories.go:66-72` — LeadSourceRepository
  - `internal/core/ports/repositories.go:101-107` — QualificationRepository
  - `internal/core/ports/repositories.go:109-115` — CountryRepository
  - `internal/core/ports/repositories.go:25-31` — LeadRepository

  **Acceptance Criteria**:
  - [ ] All five interfaces updated with new method signatures
  - [ ] go build passes

  **QA Scenarios**:
  ```
  Scenario: Interfaces compile with implementations
    Tool: Bash
    Steps:
      1. Run: go build ./...
    Expected Result: Build succeeds (impls already have methods from Tasks 4-7)
    Evidence: .sisyphus/evidence/task-10-interfaces-compile.txt
  ```

  **Commit**: YES (groups with Wave 2)
  - Message: `feat(ports): add FindByName and BulkInsert to repository interfaces`
  - Files: `internal/core/ports/repositories.go`

---

- [ ] 12. Service — Add ImportLeads method

  **What to do**:
  - Add `ImportLeads(ctx context.Context, data []byte, ext string, assignedTo string) (*ports.ImportResult, error)` to `lead_service.go`
  - Constructor update: `NewLeadService(leadRepo, categoryRepo, sourceRepo, qualificationRepo, countryRepo, aiClient)` — add countryRepo + aiClient
  - Update `cmd/api/main.go` to pass new dependencies to NewLeadService
  - **Permission Logic** (in handler, not service — service receives resolved assignedTo):
    - Handler checks permission before calling service
    - If admin: uses assignedTo param from request (or empty = unassigned)
    - If user: assignedTo = their own user_id (always)
  - **Country Logic**:
    - If Excel has country column: `countryRepo.FindByName(ctx, countryName)` (case-insensitive)
    - If country not found or column missing: default to UAE via `countryRepo.FindByName(ctx, "United Arab Emirates")`
    - If UAE not found in DB: skip country assignment (leave empty)
  - Flow:
    1. Parse file using `excel.ParseFile(data, ext)` — handles both .xlsx and .csv
    2. Validate: file not empty, row count < max
    3. Map columns using `excel.MapColumns(ctx, aiClient, headers, rows[0])`
    4. Extract unique reference values per field (category, source, qualification, country)
    5. Fetch existing categories/sources/qualifications/countries from DB
    6. Map reference values using `excel.MapReferenceValues()`
    7. Create missing categories/sources (NOT qualifications or countries)
    8. For country: lookup by name, fallback to UAE, skip if not found
    9. Transform each row to `domain.Lead` with resolved assignedTo and countryID
    10. Deduplicate by email within batch
    11. Call `leadRepo.BulkInsert(ctx, leads)`
    12. Return ImportResult with counts, errors, and created items

  **Must NOT do**:
  - Do not auto-create qualifications (report as error)
  - Do not auto-create countries (default to UAE or skip)
  - Do not update existing leads (skip duplicates)
  - Do not store the file
  - Do not check permissions in service (handler does that)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Complex orchestration with multiple dependencies, error handling, and business logic
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 9-11)
  - **Blocks**: Task 13
  - **Blocked By**: Tasks 9, 10, 11

  **References**:
  - `internal/core/services/lead_service.go:13-21` — Constructor pattern
  - `internal/core/services/lead_service.go:23-88` — CreateLead method (reference for field mapping)
  - `internal/core/services/lead_service.go:24` — getTenantIDFromContext usage
  - `internal/core/domain/lead.go:44-58` — NewLead constructor
  - `internal/core/domain/lead_category.go:18-26` — NewLeadCategory constructor
  - `internal/core/domain/lead_source.go:18-26` — NewLeadSource constructor
  - `internal/core/domain/country.go` — Country struct (global, no TenantID)
  - `pkg/excel/parser.go` (Task 3) — ParseFile function
  - `pkg/excel/mapper.go` (Task 9) — MapColumns, MapReferenceValues functions
  - `pkg/ai/client.go` (Task 2) — AI client

  **Acceptance Criteria**:
  - [ ] ImportLeads method compiles with correct signature (includes assignedTo param)
  - [ ] NewLeadService accepts all required dependencies (including countryRepo, aiClient)
  - [ ] Country mapping: finds by name or defaults to UAE
  - [ ] main.go updated to pass new dependencies
  - [ ] go build passes

  **QA Scenarios**:
  ```
  Scenario: ImportLeads processes valid Excel data with country and assignedTo
    Tool: Bash (integration test with curl)
    Preconditions: Server running, valid JWT token (admin), test xlsx with country column
    Steps:
      1. POST /api/v1/leads/import with file + assigned_to form field
      2. Assert response has inserted=3, skipped=0
      3. Query MongoDB — leads have correct country_id and assigned_to
    Expected Result: Leads in database with resolved country and assignedTo
    Evidence: .sisyphus/evidence/task-12-import-with-country.txt

  Scenario: ImportLeads defaults to UAE when country missing
    Tool: Bash (integration test)
    Preconditions: xlsx file without country column
    Steps:
      1. POST /api/v1/leads/import
      2. Query leads — country_id should be UAE's ObjectID
    Expected Result: All leads assigned to UAE by default
    Evidence: .sisyphus/evidence/task-12-import-default-uae.txt

  Scenario: ImportLeads reports errors for invalid rows
    Tool: Bash (integration test)
    Preconditions: xlsx file with 2 valid rows and 1 row missing email/phone
    Steps:
      1. POST /api/v1/leads/import
      2. Assert inserted=2, skipped=1
      3. Assert errors array has entry for invalid row
    Expected Result: Partial success with error detail
    Evidence: .sisyphus/evidence/task-12-import-partial.txt
  ```

  **Commit**: YES (groups with Wave 2)
  - Message: `feat(service): add ImportLeads with AI mapping, country resolution, and assignedTo logic`
  - Files: `internal/core/services/lead_service.go`, `cmd/api/main.go`

---

- [ ] 13. Handler — Add ImportLeads endpoint

  **What to do**:
  - Add `ImportLeads(c *fiber.Ctx) error` method to `lead_handler.go`
  - Constructor needs `permissionService` for permission checks
  - Flow:
    1. **Permission Check**:
       - Get user role from `c.Locals("role")` and user_id from `c.Locals("user_id")`
       - Check if user has permission: `permissionService.CheckPermission(ctx, role, "/api/v1/leads/import", "POST")`
       - If no permission → return 403 Forbidden
    2. **File Validation**:
       - `file, err := c.FormFile("file")`
       - Get extension: `ext := strings.ToLower(filepath.Ext(file.Filename))`
       - Validate extension is .xlsx or .csv (reject .xls and others with 400)
    3. **AssignedTo Resolution**:
       - If role is admin: `assignedTo = c.FormValue("assigned_to")` (optional param)
       - If role is user: `assignedTo = c.Locals("user_id").(string)` (always self)
    4. **Process**:
       - `src, err := file.Open()` + `defer src.Close()`
       - `buf := new(bytes.Buffer)` + `io.Copy(buf, src)`
       - Call `h.service.ImportLeads(c.Context(), buf.Bytes(), ext, assignedTo)`
    5. Return JSON response with ImportResult
  - Add Swagger doc comments

  **Must NOT do**:
  - Do not save file to disk
  - Do not add progress streaming
  - Do not let non-admin users choose assigned_to

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standard handler following existing pattern with permission check
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3 (with Task 14)
  - **Blocks**: Task 14
  - **Blocked By**: Task 12

  **References**:
  - `internal/adapters/handler/lead_handler.go:32-45` — CreateLead handler pattern
  - `internal/adapters/handler/lead_handler.go:11-17` — Constructor pattern
  - `internal/adapters/handler/lead_handler.go:19-31` — Swagger doc pattern
  - `internal/core/services/permission_service.go:93` — CheckPermission method
  - `pkg/middleware/auth_middleware.go:28-31` — How role and user_id are set on context

  **Acceptance Criteria**:
  - [ ] ImportLeads handler method exists with permission check
  - [ ] Accepts .xlsx and .csv file extensions
  - [ ] Rejects .xls and other extensions with 400
  - [ ] Rejects users without import permission (403)
  - [ ] Admin can specify assigned_to via form field
  - [ ] Non-admin's assigned_to defaults to their user_id
  - [ ] Reads file into bytes.Buffer in memory
  - [ ] Returns ImportResult as JSON
  - [ ] Swagger docs present

  **QA Scenarios**:
  ```
  Scenario: Admin uploads xlsx with assigned_to
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:3000/api/v1/leads/import \
           -H "Authorization: Bearer $ADMIN_TOKEN" \
           -F "file=@test.xlsx" \
           -F "assigned_to=$USER_ID"
    Expected Result: 200 OK, leads assigned to specified user
    Evidence: .sisyphus/evidence/task-13-admin-upload.xlsx

  Scenario: User uploads xlsx (assigned_to = self)
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:3000/api/v1/leads/import \
           -H "Authorization: Bearer $USER_TOKEN" \
           -F "file=@test.xlsx"
    Expected Result: 200 OK, leads assigned to the authenticated user
    Evidence: .sisyphus/evidence/task-13-user-upload.xlsx

  Scenario: User without permission is rejected
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:3000/api/v1/leads/import \
           -H "Authorization: Bearer $NO_PERM_TOKEN" \
           -F "file=@test.xlsx"
    Expected Result: 403 Forbidden
    Evidence: .sisyphus/evidence/task-13-no-permission.txt

  Scenario: Upload xls file (rejected)
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:3000/api/v1/leads/import \
           -H "Authorization: Bearer $TOKEN" \
           -F "file=@test.xls"
    Expected Result: 400 Bad Request: "Unsupported format. Please upload .xlsx or .csv files."
    Evidence: .sisyphus/evidence/task-13-reject-xls.txt
  ```

  **Commit**: YES (groups with Wave 3)
  - Message: `feat(handler): add ImportLeads with permission check and assignedTo resolution`
  - Files: `internal/adapters/handler/lead_handler.go`

---

- [ ] 14. Route — Register /leads/import endpoint

  **What to do**:
  - In `cmd/api/main.go`, add route: `protected.Post("/leads/import", authz, leadHandler.ImportLeads)`
  - Add Fiber body limit middleware for this route group (50MB max)
  - Place after existing lead routes

  **Must NOT do**:
  - Do not change existing routes
  - Do not add route without authz middleware

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single line route registration
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3 (final)
  - **Blocks**: None
  - **Blocked By**: Task 13

  **References**:
  - `cmd/api/main.go:168-173` — Existing lead routes pattern
  - `cmd/api/main.go:135` — protected group with authz middleware

  **Acceptance Criteria**:
  - [ ] Route registered: POST /api/v1/leads/import
  - [ ] Route requires authentication (protected group)
  - [ ] Route requires authorization (authz middleware)
  - [ ] go build passes
  - [ ] Server starts without errors

  **QA Scenarios**:
  ```
  Scenario: Route is accessible with valid token
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:3000/api/v1/leads/import \
           -H "Authorization: Bearer $TOKEN" \
           -F "file=@test.xlsx"
    Expected Result: Request reaches handler (200 or error response, not 404)
    Evidence: .sisyphus/evidence/task-13-route-accessible.txt

  Scenario: Route rejects unauthenticated request
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:3000/api/v1/leads/import \
           -F "file=@test.xlsx"
    Expected Result: 401 Unauthorized
    Evidence: .sisyphus/evidence/task-13-route-unauth.txt
  ```

  **Commit**: YES (final commit)
  - Message: `feat(api): register leads import route with auth middleware`
  - Files: `cmd/api/main.go`

---

## Final Verification Wave

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Verify all 14 tasks completed. Check: Config has AI fields, AI client exists, Excel parser works for xlsx+csv, all repos have FindByName/BulkInsert, country repo has FindByName, service has ImportLeads with assignedTo, handler has permission check, route registered. Search for forbidden patterns (file storage, per-row AI calls, cross-tenant leaks).

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go build ./...` + `go vet ./...`. Check all new files for: proper error handling, defer Close() calls, no hardcoded values, consistent naming.

- [ ] F3. **Integration QA** — `unspecified-high`
  Test full flow: upload valid xlsx → verify leads in DB with correct country and assignedTo. Test permission: admin can choose assignedTo, user defaults to self. Test error cases: .xls rejection, oversized file, empty file, invalid data rows.

- [ ] F4. **Scope Fidelity Check** — `deep`
  Verify: no file storage on disk, no per-row AI calls, qualifications/countries NOT auto-created, only .xlsx/.csv supported, tenant isolation on all operations, permission check present, country defaults to UAE.

---

## Commit Strategy

Wave 1: `feat: add Excel import foundation (config, AI client, parser, repo methods)`
- config/config.go, pkg/ai/client.go, pkg/excel/parser.go
- mongo_lead_category_repo.go, mongo_lead_source_repo.go, mongo_qualification_repo.go, mongo_lead_repo.go

Wave 2: `feat: add import service with AI mapping and bulk insert`
- pkg/excel/mapper.go, internal/core/ports/services.go, internal/core/ports/repositories.go
- internal/core/services/lead_service.go, cmd/api/main.go

Wave 3: `feat: add import endpoint and route`
- internal/adapters/handler/lead_handler.go, cmd/api/main.go

---

## Success Criteria

### Verification Commands
```bash
go build ./...                          # Expected: no errors
go vet ./...                            # Expected: no warnings
curl -X POST http://localhost:3000/api/v1/leads/import \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test.xlsx"                  # Expected: 200 with ImportResult JSON
curl -X POST http://localhost:3000/api/v1/leads/import \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test.csv"                   # Expected: 200 with ImportResult JSON
curl -X POST http://localhost:3000/api/v1/leads/import \
  -F "file=@test.xlsx"                  # Expected: 401 Unauthorized
curl -X POST http://localhost:3000/api/v1/leads/import \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test.xls"                   # Expected: 400 Bad Request
```

### Final Checklist
- [ ] Config has AI_URL, AI_API_KEY, AI_MODEL fields
- [ ] pkg/ai/client.go exists with Chat method
- [ ] pkg/excel/parser.go exists with ParseFile function (supports .xlsx + .csv)
- [ ] pkg/excel/mapper.go exists with MapColumns and MapReferenceValues
- [ ] All repo interfaces updated with FindByName/BulkInsert
- [ ] All repo implementations have new methods (including country FindByName)
- [ ] LeadService has ImportLeads method with country resolution and assignedTo
- [ ] LeadHandler has ImportLeads method with permission check
- [ ] Route POST /api/v1/leads/import registered with auth
- [ ] go build passes
- [ ] .xlsx files processed correctly
- [ ] .csv files processed correctly
- [ ] .xls files rejected with clear error
- [ ] In-memory processing (no temp files)
- [ ] Country defaults to UAE when not provided
- [ ] Admin can choose assigned_to, user defaults to self
- [ ] Permission check rejects unauthorized users
- [ ] Partial success reporting works


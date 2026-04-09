# Follow Ups API Implementation

## TL;DR

> **Quick Summary**: Create a new "lead follow-ups" API following the existing lead appointments pattern, with key differences: renamed OrganizerID→CreatorID, status values active/closed (final), and authorization allowing ALL users with permission to update/delete (not just creator).

> **Deliverables**:
> - LeadFollowUp domain model with CreatorID field
> - FollowUpStatus enum (active/closed)
> - Complete CRUD API: Create, Get, Update, Delete, List
> - MongoDB repository with tenant isolation
> - Permission rules for lead-follow-ups resource
> - Service tests (TDD approach)
> - Swagger documentation

> **Estimated Effort**: Medium
> **Parallel Execution**: YES - 3 waves
> **Critical Path**: Domain → Service → Handler → Routes → Tests

---

## Context

### Original Request
User wants to add a follow-ups API similar to the existing lead appointments collection, using the same model structure but with different status values (active/closed) and authorization rules (any authorized user can modify, not just the organizer).

### Interview Summary

**Key Discussions**:
- **Field Structure**: Confirmed same fields as appointments (ID, TenantID, LeadID, CreatorID, Title, Description, StartTime, EndTime, Status, CreatedAt, UpdatedAt)
- **Status Values**: Only "active" and "closed" (different from appointments)
- **Status Logic**: Closed is final - cannot be reopened (validation needed)
- **Field Naming**: Renamed OrganizerID → CreatorID for semantic clarity
- **Authorization**: ALL authorized users can update/delete (KEY DIFFERENCE from appointments)
- **Permissions**: Both "list" and "list_own" permissions (same as appointments)
- **Test Strategy**: TDD with testify framework

**Research Findings**:
- Found complete lead appointments implementation to replicate
- Domain model: `internal/core/domain/lead_appointment.go:18-30` - Fields, status enum, constructor
- Service: `internal/core/services/lead_appointment_service.go:24-141` - Business logic with organizer checks
- Handler: `internal/adapters/handler/lead_appointment_handler.go:32-188` -5 HTTP endpoints
- Repository: `internal/adapters/storage/mongo_lead_appointment_repo.go:17-157` - MongoDB implementation
- Routes: `cmd/api/main.go:195-199` - Route registration
- Permissions: `pkg/database/migrations.go:218-225` - Permission rules

**Metis Review Identified**:
- Need to explicitly remove organizer-only authorization check from service layer
- Must validate status transitions (closed cannot be reopened)
- Must rename OrganizerID → CreatorID throughout
- Must add validation for closed status immutability

### Key Differences from Lead Appointments

| Aspect | Appointments | Follow-Ups |
|--------|-------------|-----------|
| ID Field | OrganizerID | CreatorID |
| Status values | scheduled, completed, rescheduled, cancelled | active, closed |
| Status logic | Can transition freely | Closed is final |
| Update/Delete auth | Only organizer | Any user with permission |
| Permission resource | lead-appointments | lead-follow-ups |

---

## Work Objectives

### Core Objective
Create a complete CRUD API for lead follow-ups that matches the appointments pattern with specified differences in authorization and status handling.

### Concrete Deliverables
- `internal/core/domain/lead_follow_up.go` - Domain model + status enum
- `internal/core/ports/repositories.go` - Add FollowUpRepository interface
- `internal/core/ports/services.go` - Add FollowUpService interface + request structs
- `internal/core/services/lead_follow_up_service.go` - Service implementation
- `internal/core/services/lead_follow_up_service_test.go` - TDD tests
- `internal/adapters/handler/lead_follow_up_handler.go` - HTTP handlers
- `internal/adapters/storage/mongo_lead_follow_up_repo.go` - MongoDB repository
- `cmd/api/main.go` - Route registration (updated)
- `pkg/database/migrations.go` - Permission rules (updated)

### Definition of Done
- [ ] All CRUD operations work via HTTP endpoints
- [ ] Authorization allows any user with permission to update/delete
- [ ] Status validation prevents updates to closed follow-ups
- [ ] Tenant isolation enforced
- [ ] Permissions registered in database
- [ ] Tests pass: `go test ./internal/core/services/...`
- [ ] Swagger docs generated: `swag init`

### Must Have
- Complete CRUD API for lead follow-ups
- Authorization check in permissions layer (NOT organizer-only in service)
- Status validation: cannot update closed follow-ups
- Tenant isolation via context
- Permission resource: lead-follow-ups
- MongoDB collection: lead_follow_ups

### Must NOT Have (Guardrails from Metis)
- NO status transitions from closed → active
- NO notification/reminder system
- NO soft delete
- NO additional fields beyond appointments
- NO reassignment of CreatorID
- NO bulk operations
- NO AI-generated over-abstraction

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: YES (testify framework found)
- **Automated tests**: YES (TDD approach)
- **Framework**: testify (stretchr/testify)
- **Approach**: Write tests BEFORE implementation

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Backend/API**: Use Bash (curl) — Send requests, assert status + response fields
- **Service Tests**: Use Bash (go test) — Run unit tests, verify pass/fail

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation):
├── Task 1: Domain model + status enum [quick]
├── Task 2: Repository interface [quick]
├── Task 3: Service interface + request structs [quick]
└── Task 4: Permission rules [quick]

Wave 2 (Core Implementation - depends on Wave 1):
├── Task 5: MongoDB repository implementation [quick]
├── Task 6: Service implementation (with tests via TDD) [unspecified-high]
├── Task 7: HTTP handler implementation [quick]
└── Task 8: Route registration [quick]

Wave FINAL (Verification):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Integration QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
```

### Dependency Matrix
- **1-4**: No dependencies (can run in parallel)
- **5**: Depends on 2
- **6**: Depends on 1, 3, 5
- **7**: Depends on 3, 6
- **8**: Depends on 7
- **FINAL**: Depends on all previous tasks

---

## TODOs

- [ ] 1. Create LeadFollowUp domain model with FollowUpStatus enum

  **What to do**:
  - Create `internal/core/domain/lead_follow_up.go`
  - Define `FollowUpStatus` type with constants: `StatusActive`, `StatusClosed`
  - Define `LeadFollowUp` struct with fields: ID, TenantID, LeadID, CreatorID (NOT OrganizerID), Title, Description, StartTime, EndTime, Status, CreatedAt, UpdatedAt
  - Add `NewLeadFollowUp()` constructor function
  - Add validation helper: `IsValidFollowUpStatus(status FollowUpStatus) bool`

  **Must NOT do**:
  - DO NOT name field OrganizerID (must be CreatorID)
  - DO NOT add status values beyond active/closed
  - DO NOT add soft delete fields
  - DO NOT add notification fields

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple domain model creation following exact pattern
  - **Skills**: []
    - No special skills needed - straightforward Go struct

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 2-4)
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4)
  - **Blocks**: Task 6 (service implementation needs domain model)
  - **Blocked By**: None (can start immediately)

  **References** (CRITICAL - Be Exhaustive):
  
  **Pattern References** (existing code to follow):
  - `internal/core/domain/lead_appointment.go:9-30` - Status enum pattern and struct fields (REPLICATE THIS EXACTLY, just rename OrganizerID→CreatorID)
  - `internal/core/domain/lead_appointment.go:32-46` - Constructor pattern (NewLeadAppointment function)
  
  **Why Each Reference Matters**:
  - `lead_appointment.go:9-30`: Shows exact status type pattern - use this as template, change values to active/closed
  - `lead_appointment.go:32-46`: Shows constructor pattern - copy this, change names toLeadFollowUp/CreatorID

  **Acceptance Criteria**:
  - [ ] File created: `internal/core/domain/lead_follow_up.go`
  - [ ] `FollowUpStatus` type defined with `StatusActive` and `StatusClosed` constants
  - [ ] `LeadFollowUp` struct has ALL fields: ID, TenantID, LeadID, CreatorID, Title, Description, StartTime, EndTime, Status, CreatedAt, UpdatedAt
  - [ ] Field named `CreatorID` (NOT OrganizerID)
  - [ ] `NewLeadFollowUp()` constructor exists and initializes CreatedAt/UpdatedAt
  - [ ] `go build ./internal/core/domain` succeeds

  **QA Scenarios (MANDATORY - agent-executed)**:
  ```
  Scenario: Validate status enum values
    Tool: Bash (go test)
    Preconditions: Domain file created
    Steps:
      1. Create test file: domain/lead_follow_up_test.go
      2. Add test: TestFollowUpStatus_Values
      3. Assert StatusActive == "active"
      4. Assert StatusClosed == "closed"
      5. Assert Len(FollowUpStatus constants) == 2
    Expected Result: All assertions pass
    Failure Indicators: Status values not matching, wrong count
    Evidence: .sisyphus/evidence/task-1-status-enum.test.go

  Scenario: Validate struct field names
    Tool: Bash (go test)
    Preconditions: Domain file created
    Steps:
      1. Add test: TestLeadFollowUp_FieldNames
      2. Use reflect to check struct has field "CreatorID"
      3. Assert NO field named "OrganizerID"
      4. Assert ALL fields present: ID, TenantID, LeadID, CreatorID, Title, Description, StartTime, EndTime, Status, CreatedAt, UpdatedAt
    Expected Result: All field assertions pass
    Failure Indicators: Missing fields, wrong field names
    Evidence: .sisyphus/evidence/task-1-field-names.test.go
  ```

  **Evidence to Capture**:
  - [ ] Test file: domain/lead_follow_up_test.go with passing tests
  - [ ] Build output showing success

  **Commit**: YES
  - Message: `feat(follow-ups): add LeadFollowUp domain model and status enum`
  - Files: `internal/core/domain/lead_follow_up.go, internal/core/domain/lead_follow_up_test.go`
  - Pre-commit: `go test ./internal/core/domain/...`

- [ ] 2. Add FollowUpRepository interface to ports

  **What to do**:
  - Edit `internal/core/ports/repositories.go`
  - Add `LeadFollowUpRepository` interface with methods:
    - `Create(ctx context.Context, followUp *domain.LeadFollowUp) error`
    - `GetByID(ctx context.Context, id primitive.ObjectID) (*domain.LeadFollowUp, error)`
    - `Update(ctx context.Context, followUp *domain.LeadFollowUp) error`
    - `Delete(ctx context.Context, id primitive.ObjectID) error`
    - `ListByLeadID(ctx context.Context, leadID primitive.ObjectID, filter interface{}, offset, limit int64) ([]*ports.FollowUpListItem, int64, error)`

  **Must NOT do**:
  - DO NOT use `OrganizerID` in any struct/interface definitions
  - DO NOT add methods beyond CRUD + ListByLeadID
  - DO NOT modify existing repository interfaces

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple interface definition following exact pattern
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 1, 3, 4)
  - **Parallel Group**: Wave 1
  - **Blocks**: Task 5 (MongoDB repository implementation)
  - **Blocked By**: None

  **References**:
  
  **Pattern References**:
  - `internal/core/ports/repositories.go:48-52` - LeadAppointmentRepository interface pattern (COPY THIS STRUCTURE)
  - `internal/core/ports/services.go:336-348` - FollowUpListItem response struct pattern
  
  **Why Each Reference Matters**:
  - `repositories.go:48-52`: Shows exact method signatures - replicate for LeadFollowUpRepository
  - `services.go:336-348`: Shows AppointmentListItem pattern - create similar FollowUpListItem

  **Acceptance Criteria**:
  - [ ] `LeadFollowUpRepository` interface added to `internal/core/ports/repositories.go`
  - [ ] Interface has all 5 methods: Create, GetByID, Update, Delete, ListByLeadID
  - [ ] Method signatures match pattern: context.Context, ObjectID params, domain pointer returns
  - [ ] `go build ./internal/core/ports` succeeds

  **QA Scenarios**:
  ```
  Scenario: Verify interface methods exist
    Tool: Bash (grep)
    Preconditions: File edited
    Steps:
      1. grep -n "LeadFollowUpRepository interface" internal/core/ports/repositories.go
      2. grep -n "Create(ctx context.Context" internal/core/ports/repositories.go | grep LeadFollowUp
      3. grep -n "GetByID(ctx context.Context" internal/core/ports/repositories.go | grep LeadFollowUp
      4. grep -n "Update(ctx context.Context" internal/core/ports/repositories.go | grep LeadFollowUp
      5. grep -n "Delete(ctx context.Context" internal.core/ports/repositories.go | grep LeadFollowUp
      6. grep -n "ListByLeadID(ctx context.Context" internal/core/ports/repositories.go | grep LeadFollowUp
    Expected Result: All5 methods found with correct signatures
    Failure Indicators: Missing methods, wrong signatures
    Evidence: .sisyphus/evidence/task-2-interface.txt
  ```

  **Commit**: NO (groups with Task 3)

- [ ] 3. Add FollowUpService interface and request structs to ports

  **What to do**:
  - Edit `internal/core/ports/services.go`
  - Add request structs:
    - `CreateLeadFollowUpRequest` with fields: Title, Description, StartTime, EndTime, Status
    - `UpdateLeadFollowUpRequest` with fields: Title, Description, StartTime, EndTime, Status (all optional except status validation)
    - `FollowUpListItem` with fields: ID, TenantID, LeadID, Creator, Title, Description, StartTime, EndTime, Status, CreatedAt, UpdatedAt
  - Add `LeadFollowUpService` interface with methods:
    - `CreateLeadFollowUp(ctx context.Context, leadID primitive.ObjectID, req CreateLeadFollowUpRequest) (*domain.LeadFollowUp, error)`
    - `GetLeadFollowUp(ctx context.Context, id primitive.ObjectID) (*domain.LeadFollowUp, error)`
    - `UpdateLeadFollowUp(ctx context.Context, id primitive.ObjectID, req UpdateLeadFollowUpRequest) (*domain.LeadFollowUp, error)`
    - `DeleteLeadFollowUp(ctx context.Context, id primitive.ObjectID) error`
    - `ListLeadFollowUps(ctx context.Context, leadID primitive.ObjectID, req FilterRequest) ([]*FollowUpListItem, int64, error)`

  **Must NOT do**:
  - DO NOT use `OrganizerID` anywhere - use `CreatorID`
  - DO NOT add fields beyond appointments pattern
  - DO NOT modify existing service interfaces

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple struct and interface definitions
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 1, 2, 4)
  - **Parallel Group**: Wave 1
  - **Blocks**: Task 6 (service implementation)
  - **Blocked By**: None

  **References**:
  
  **Pattern References**:
  - `internal/core/ports/services.go:319-333` - CreateLeadAppointmentRequest/UpdateLeadAppointmentRequest structs (COPY THIS)
  - `internal/core/ports/services.go:336-348` - AppointmentListItem struct (COPY THIS, change Organizer→Creator)
  - `internal/core/ports/services.go:350-356` - LeadAppointmentService interface (COPY THIS)

  **Acceptance Criteria**:
  - [ ] `CreateLeadFollowUpRequest` struct defined with Title, Description, StartTime, EndTime, Status fields
  - [ ] `UpdateLeadFollowUpRequest` struct defined with all optional fields
  - [ ] `FollowUpListItem` struct defined with Creator field (not Organizer)
  - [ ] `LeadFollowUpService` interface defined with all5 methods
  - [ ] `go build ./internal/core/ports` succeeds

  **Commit**: YES (with Task 2)
  - Message: `feat(follow-ups): add repository and service interfaces to ports`
  - Files: `internal/core/ports/repositories.go, internal/core/ports/services.go`
  - Pre-commit: `go build ./internal/core/ports/...`

- [ ] 4. Add permission rules for lead-follow-ups resource

  **What to do**:
  - Edit `pkg/database/migrations.go`
  - Add permission rules in the `DefaultPermissions` slice:
    - Resource: "lead-follow-ups"
    - Actions: create, view, update, delete, list, list_own
    - Admin: all actions granted
    - User: create, view, update, delete, list_own
  - Follow exact pattern from lead-appointments permissions

  **Must NOT do**:
  - DO NOT use resource name "lead-appointments" - use "lead-follow-ups"
  - DO NOT skip any actions (create, view, update, delete, list, list_own)
  - DO NOT modify existing permission rules

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple permission rule additions
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 1-3)
  - **Parallel Group**: Wave 1
  - **Blocks**: None (independent change)
  - **Blocked By**: None

  **References**:
  
  **Pattern References**:
  - `pkg/database/migrations.go:218-225` - Lead appointments permission rules (COPY THIS STRUCTURE)
  
  **Why This Reference Matters**:
  - Shows exact format and fields for permission rules
  - Demonstrates how to add both admin and user permissions
  - Shows URL pattern, HTTP method, and scope pattern

  **Acceptance Criteria**:
  - [ ] Permission rules added for resource "lead-follow-ups"
  - [ ] All6 actions present: create, view, update, delete, list, list_own
  - [ ] Admin role has all actions
  - [ ] User role has: create, view, update, delete, list_own
  - [ ] URLs use pattern: /api/v1/leads/:lead_id/follow-ups
  - [ ] `go build ./pkg/database` succeeds

  **QA Scenarios**:
  ```
  Scenario: Verify permission rules exist
    Tool: Bash (grep)
    Steps:
      1. grep -n "lead-follow-ups" pkg/database/migrations.go
      2. grep -n "Resource.*lead-follow-ups" pkg/database/migrations.go | wc -l
      3. Assert count equals 6 (for all actions)
    Expected Result: 6 permission rules found for lead-follow-ups
    Failure Indicators: Missing rules, wrong count
    Evidence: .sisyphus/evidence/task-4-permissions.txt
  ```

  **Commit**: YES
  - Message: `feat(follow-ups): add permission rules for lead-follow-ups resource`
  - Files: `pkg/database/migrations.go`
  - Pre-commit: `go build ./pkg/database/...`

- [ ] 5. Implement MongoDB repository for follow-ups

  **What to do**:
  - Create `internal/adapters/storage/mongo_lead_follow_up_repo.go`
  - Implement `MongoLeadFollowUpRepository` struct
  - Implement all 5 methods from `LeadFollowUpRepository` interface
  - Use collection name: "lead_follow_ups"
  - Add tenant isolation via `getTenantIDFromContext`
  - Add scope filter support via `middleware.GetScopeFilter`
  - Implement aggregation pipeline for ListByLeadID (join with users collection to resolve Creator)
  - Add BSON tags for all fields
  - Handle `mongo.ErrNoDocuments` properly

  **Must NOT do**:
  - DO NOT use collection name "lead_appointments"
  - DO NOT forget tenant isolation in GetByID, Update, Delete
  - DO NOT forget scope filter for list_own
  - DO NOT use Organizer field - use Creator

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Straightforward repository implementation following exact pattern
  - **Skills**: []
    - No special skills needed - copy from appointments repo

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 6-8)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 6 (service needs repository interface)
  - **Blocked By**: Task 2 (needs repository interface)

  **References**:
  
  **Pattern References**:
  - `internal/adapters/storage/mongo_lead_appointment_repo.go:17-30` - Repository struct and constructor (COPY THIS)
  - `internal/adapters/storage/mongo_lead_appointment_repo.go:27-51` - Create and GetByID methods (COPY PATTERN)
  - `internal/adapters/storage/mongo_lead_appointment_repo.go:53-111` - ListByLeadID with aggregation pipeline (REPLICATE THIS)
  - `internal/adapters/storage/mongo_lead_appointment_repo.go:113-136` - Update method (COPY THIS)
  - `internal/adapters/storage/mongo_lead_appointment_repo.go:138-157` - Delete method (COPY THIS)
  
  **Why Each Reference Matters**:
  - `17-30`: Shows struct definition and NewMongoLead... constructor pattern
  - `27-51`: Shows tenant isolation pattern via getTenantIDFromContext and scope filter
  - `53-111`: Shows aggregation pipeline pattern with user lookup - must replicate for Creator field
  - `113-136`: Shows update pattern with tenant isolation
  - `138-157`: Shows delete pattern with tenant isolation

  **Acceptance Criteria**:
  - [ ] File created: `internal/adapters/storage/mongo_lead_follow_up_repo.go`
  - [ ] Collection name: "lead_follow_ups"
  - [ ] All 5 interface methods implemented: Create, GetByID, Update, Delete, ListByLeadID
  - [ ] Tenant isolation in GetByID (filter includes tenant_id)
  - [ ] Scope filter in ListByLeadID (list_own support)
  - [ ] Aggregation pipeline joins users collection for Creator field
  - [ ] Error handling for ErrNoDocuments
  - [ ] `go build ./internal/adapters/storage` succeeds

  **QA Scenarios**:
  ```
  Scenario: Verify collection name
    Tool: Bash (grep)
    Steps:
      1. grep -n "lead_follow_ups" internal/adapters/storage/mongo_lead_follow_up_repo.go
      2. Assert found in NewMongoLeadFollowUpRepository
    Expected Result: Collection name is lead_follow_ups
    Evidence: .sisyphus/evidence/task-5-collection.txt
  
  Scenario: Verify tenant isolation
    Tool: Bash (grep)
    Steps:
      1. grep -n "tenant_id" internal/adapters/storage/mongo_lead_follow_up_repo.go
      2. grep -n "GetScopeFilter" internal/adapters/storage/mongo_lead_follow_up_repo.go
      3. Assert both found in GetByID, Update, Delete, ListByLeadID
    Expected Result: Tenant isolation present in all methods
    Evidence: .sisyphus/evidence/task-5-tenant-isolation.txt
  ```

  **Commit**: YES
  - Message: `feat(follow-ups): implement MongoDB repository`
  - Files: `internal/adapters/storage/mongo_lead_follow_up_repo.go`
  - Pre-commit: `go build ./internal/adapters/storage/...`

- [ ] 6. Implement service layer with TDD tests

  **What to do**:
  - Create `internal/core/services/lead_follow_up_service.go`
  - Create `internal/core/services/lead_follow_up_service_test.go` (TDD - write tests FIRST)
  - Implement `LeadFollowUpService` struct with repository dependency
  - Implement all 5 service methods:
    - **CreateLeadFollowUp**: Validate lead exists, auto-set CreatorID from context, validate dates, set default status
    - **GetLeadFollowUp**: Get by ID with tenant isolation
    - **UpdateLeadFollowUp**: ANY USER WITH PERMISSION (NOT creator-only), validate status if changing (cannot update closed), update timestamps
    - **DeleteLeadFollowUp**: ANY USER WITH PERMISSION (NOT creator-only)
    - **ListLeadFollowUps**: List by lead ID with pagination
  - **CRITICAL DIFFERENCE**: Remove organizer-only check from Update and Delete methods

  **Must NOT do**:
  - DO NOT add `OrganizerID != userID` check (KEY DIFFERENCE from appointments)
  - DO NOT allow status transition from closed → active
  - DO NOT forget to validate: StartTime < EndTime
  - DO NOT forget to validate lead existence on create
  - DO NOT forget to set CreatorID from context (NOT OrganizerID)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Complex business logic with critical authorization differences
  - **Skills**: []
    - No special skills needed, but high attention to authorization logic

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 5, 7, 8)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 7 (handler needs service interface)
  - **Blocked By**: Task 1 (domain model), Task 3 (service interface), Task 5 (repository)

  **References**:
  
  **Pattern References**:
  - `internal/core/services/lead_appointment_service.go:12-22` - Service struct and constructor (COPY STRUCTURE)
  - `internal/core/services/lead_appointment_service.go:24-70` - CreateLeadAppointment pattern (REPLICATE, change Organizer→Creator)
  - `internal/core/services/lead_appointment_service.go:72-74` - GetLeadAppointment (COPY)
  - `internal/core/services/lead_appointment_service.go:76-118` - UpdateLeadAppointment **CRITICAL: DO NOT COPY lines 88-90 (organizer check) - THIS IS THE KEY DIFFERENCE**
  - `internal/core/services/lead_appointment_service.go:120-137` - DeleteLeadAppointment **CRITICAL: DO NOT COPY lines 132-134 (organizer check) - THIS IS THE KEY DIFFERENCE**
  - `internal/core/services/lead_appointment_service.go:139-141` - ListLeadAppointments (COPY)
  
  **Why Each Reference Matters**:
  - `12-22`: Shows service struct pattern with repository dependency
  - `24-70`: Shows creation logic with validation - MUST change Organizer to Creator
  - `76-74`: Simple get by ID
  - `76-118`: **CRITICAL** - Shows update logic but MUST REMOVE organizer check (lines 88-90) - THIS IS THE KEY DIFFERENCE IN REQUIREMENTS
  - `120-137`: **CRITICAL** - Shows delete logic but MUST REMOVE organizer check (lines 132-134) - THIS IS THE KEY DIFFERENCE IN REQUIREMENTS
  - `139-141`: List pattern with pagination

  **TDD Test Cases** (Write these FIRST before implementation):
  1. `TestCreateLeadFollowUp_Success` - Valid creation
  2. `TestCreateLeadFollowUp_MissingTenant` - Error without tenant context
  3. `TestCreateLeadFollowUp_MissingUser` - Error without user context
  4. `TestCreateLeadFollowUp_InvalidLead` - Error if lead not found
  5. `TestCreateLeadFollowUp_InvalidStatus` - Error with invalid status value
  6. `TestUpdateLeadFollowUp_AnyUser` - **CRITICAL TEST**: Non-creator can update (KEY DIFFERENCE)
  7. `TestUpdateLeadFollowUp_ClosedFollowUp` - Error when updating closed follow-up
  8. `TestUpdateLeadFollowUp_InvalidStatus` - Error with invalid status
  9. `TestDeleteLeadFollowUp_AnyUser` - **CRITICAL TEST**: Non-creator can delete (KEY DIFFERENCE)
  10. `TestListLeadFollowUps_Pagination` - List with offset/limit

  **Acceptance Criteria**:
  - [ ] Test file created: `lead_follow_up_service_test.go`
  - [ ] All 10 test cases pass
  - [ ] Service file created: `lead_follow_up_service.go`
  - [ ] All 5 service methods implemented
  - [ ] CreateLeadFollowUp: Sets CreatorID (NOT OrganizerID) from context
  - [ ] UpdateLeadFollowUp: NO organizer-only check (KEY DIFFERENCE)
  - [ ] UpdateLeadFollowUp: Validates closed follow-ups cannot be updated
  - [ ] DeleteLeadFollowUp: NO organizer-only check (KEY DIFFERENCE)
  - [ ] Status validation: Only "active" or "closed" allowed
  - [ ] `go test ./internal/core/services/...` passes

  **QA Scenarios**:
  ```
  Scenario: Test non-creator can update follow-up (CRITICAL)
    Tool: Bash (go test)
    Steps:
      1. Create follow-up with user A (CreatorID = userA)
      2. Attempt to update with user B
      3. Assert update succeeds (NO organizer check)
      4. Verify follow-up updated
    Expected Result: Update succeeds, no authorization error
    Failure Indicators: "unauthorized" error, organizer check present
    Evidence: .sisyphus/evidence/task-6-auth-test.log
  
  Scenario: Test closed follow-up cannot be updated
    Tool: Bash (go test)
    Steps:
      1. Create follow-up with status "closed"
      2. Attempt to update title
      3. Assert error: "cannot update closed follow-up"
      4. Verify no changes made
    Expected Result: Update rejected with error
    Failure Indicators: Update succeeds on closed follow-up
    Evidence: .sisyphus/evidence/task-6-closed-test.log
  
  Scenario: Test non-creator can delete follow-up (CRITICAL)
    Tool: Bash (go test)
    Steps:
      1. Create follow-up with user A (CreatorID = userA)
      2. Attempt to delete with user B
      3. Assert delete succeeds (NO organizer check)
    Expected Result: Delete succeeds, no authorization error
    Failure Indicators: "unauthorized" error, organizer check present
    Evidence: .sisyphus/evidence/task-6-delete-test.log
  ```

  **Commit**: YES
  - Message: `feat(follow-ups): implement service layer with TDD tests`
  - Files: `internal/core/services/lead_follow_up_service.go, internal/core/services/lead_follow_up_service_test.go`
  - Pre-commit: `go test ./internal/core/services/...`

- [ ] 7. Implement HTTP handlers for follow-ups

  **What to do**:
  - Create `internal/adapters/handler/lead_follow_up_handler.go`
  - Implement `LeadFollowUpHandler` struct with service dependency
  - Implement all 5 HTTP handlers:
    - `CreateLeadFollowUp(c *fiber.Ctx) error` - POST /leads/:lead_id/follow-ups
    - `GetLeadFollowUp(c *fiber.Ctx) error` - GET /leads/:lead_id/follow-ups/:id
    - `UpdateLeadFollowUp(c *fiber.Ctx) error` - PUT /leads/:lead_id/follow-ups/:id
    - `DeleteLeadFollowUp(c *fiber.Ctx) error` - DELETE /leads/:lead_id/follow-ups/:id
    - `ListLeadFollowUps(c *fiber.Ctx) error` - POST /leads/:lead_id/follow-ups/list
  - Add Swagger annotations for each handler
  - Follow exact pattern from lead_appointment_handler.go

  **Must NOT do**:
  - DO NOT change URL patterns (must match appointments)
  - DO NOT forget Swagger annotations
  - DO NOT use different error messages than appointments pattern
  - DO NOT forget to parse lead_id from URL params

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Straightforward handler implementation following exact pattern
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 5, 6, 8)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 8 (route registration needs handlers)
  - **Blocked By**: Task 3 (needs service interface), Task 6 (needs service implementation)

  **References**:
  
  **Pattern References**:
  - `internal/adapters/handler/lead_appointment_handler.go:11-17` - Handler struct and constructor (COPY)
  - `internal/adapters/handler/lead_appointment_handler.go:19-51` - CreateLeadAppointment handler (COPY PATTERN)
  - `internal/adapters/handler/lead_appointment_handler.go:53-79` - GetLeadAppointment handler (COPY)
  - `internal/adapters/handler/lead_appointment_handler.go:81-114` - UpdateLeadAppointment handler (COPY)
  - `internal/adapters/handler/lead_appointment_handler.go:116-142` - DeleteLeadAppointment handler (COPY)
  - `internal/adapters/handler/lead_appointment_handler.go:144-188` - ListLeadAppointments handler (COPY)
  
  **Why Each Reference Matters**:
  - `11-17`: Shows handler struct with service dependency pattern
  - `19-51`: Shows complete create handler with Swagger docs, param parsing, body parsing, error handling
  - `53-79`: Shows get handler pattern
  - `81-114`: Shows update handler pattern
  - `116-142`: Shows delete handler pattern
  - `144-188`: Shows list handler with pagination pattern

  **Acceptance Criteria**:
  - [ ] File created: `internal/adapters/handler/lead_follow_up_handler.go`
  - [ ] `LeadFollowUpHandler` struct defined with service field
  - [ ] `NewLeadFollowUpHandler` constructor created
  - [ ] All 5 handler methods implemented
  - [ ] Swagger annotations present on all handlers
  - [ ] lead_id parsed from URL params
  - [ ] Error responses use Fiber status codes
  - [ ] `go build ./internal/adapters/handler` succeeds

  **Commit**: YES
  - Message: `feat(follow-ups): implement HTTP handlers`
  - Files: `internal/adapters/handler/lead_follow_up_handler.go`
  - Pre-commit: `go build ./internal/adapters/handler/...`

- [ ] 8. Register routes and wire dependencies in main.go

  **What to do**:
  - Edit `cmd/api/main.go`
  - Import `leadFollowUpHandler` and `leadFollowUpService`
  - Initialize `leadFollowUpRepo := storage.NewMongoLeadFollowUpRepository(db)`
  - Initialize `leadFollowUpService := services.NewLeadFollowUpService(leadFollowUpRepo, leadRepo)`
  - Initialize `leadFollowUpHandler := handler.NewLeadFollowUpHandler(leadFollowUpService)`
  - Register routes under `protected` group:
    - `protected.Post("/leads/:lead_id/follow-ups", authz, leadFollowUpHandler.CreateLeadFollowUp)`
    - `protected.Get("/leads/:lead_id/follow-ups/:id", authz, leadFollowUpHandler.GetLeadFollowUp)`
    - `protected.Put("/leads/:lead_id/follow-ups/:id", authz, leadFollowUpHandler.UpdateLeadFollowUp)`
    - `protected.Delete("/leads/:lead_id/follow-ups/:id", authz, leadFollowUpHandler.DeleteLeadFollowUp)`
    - `protected.Post("/leads/:lead_id/follow-ups/list", authz, leadFollowUpHandler.ListLeadFollowUps)`
  - Follow exact registration pattern from lead appointments

  **Must NOT do**:
  - DO NOT use different URL patterns than appointments
  - DO NOT forget to pass `leadRepo` to service (validate lead exists on create)
  - DO NOT forget `authz` middleware for all routes
  - DO NOT register routes on wrong group

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple route registration following exact pattern
  - **Skills**: []
    - No special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 5-7)
  - **Parallel Group**: Wave 2
  - **Blocks**: Final verification (need running server)
  - **Blocked By**: Task 7 (needs handler implementation)

  **References**:
  
  **Pattern References**:
  - `cmd/api/main.go:195-199` - Lead appointment route registration (COPY THIS PATTERN)
  - Look for lead appointment initialization to find where to add follow-up initialization (around line 100-150)
  
  **Why Each Reference Matters**:
  - `195-199`: Shows exact route registration pattern with authz middleware
  - Shows URL pattern: `/leads/:lead_id/follow-ups`

  **Acceptance Criteria**:
  - [ ] Repository initialized: `storage.NewMongoLeadFollowUpRepository(db)`
  - [ ] Service initialized: `services.NewLeadFollowUpService(leadFollowUpRepo, leadRepo)`
  - [ ] Handler initialized: `handler.NewLeadFollowUpHandler(leadFollowUpService)`
  - [ ] All5 routes registered under protected group
  - [ ] Routes use correct URL pattern: `/leads/:lead_id/follow-ups`
  - [ ] `authz` middleware applied to all routes
  - [ ] `go build ./cmd/api` succeeds
  - [ ] Server starts without errors: `go run cmd/api/main.go`

  **QA Scenarios**:
  ```
  Scenario: Test route registration
    Tool: Bash (curl)
    Preconditions: Server running
    Steps:
      1. curl -X POST http://localhost:3000/api/v1/leads/507f1f77bcf86cd799439011/follow-ups
      2. Expect 401 Unauthorized (no token) or 403 Forbidden (permission denied)
      3. Assert route exists (not 404)
    Expected Result: Route registered, authentication required
    Failure Indicators: 404 Not Found (route not registered)
    Evidence: .sisyphus/evidence/task-8-route-test.log
  
  Scenario: Verify handler initialization
    Tool: Bash (grep)
    Steps:
      1. grep -n "NewLeadFollowUpHandler" cmd/api/main.go
      2. grep -n "NewLeadFollowUpService" cmd/api/main.go
      3. grep -n "NewMongoLeadFollowUpRepository" cmd/api/main.go
    Expected Result: All three found
    Failure Indicators: Missing initialization
    Evidence: .sisyphus/evidence/task-8-init.txt
  ```

  **Commit**: YES
  - Message: `feat(follow-ups): register routes and dependencies`
  - Files: `cmd/api/main.go`
  - Pre-commit: `go build ./cmd/api && swag init`

---

## Final Verification Wave (MANDATORY)

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists. For each "Must NOT Have": search codebase for forbidden patterns. Check evidence files exist.

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go test ./...` + linter. Review all changed files for: code quality, consistent naming, proper error handling.

- [ ] F3. **Integration QA** — `unspecified-high`
  Execute all QA scenarios from each task. Test cross-task integration. Save evidence to `.sisyphus/evidence/final-qa/`.

- [ ] F4. **Scope Fidelity Check** — `deep`
  Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check "Must NOT do" compliance.

---

## Commit Strategy

- **Atomic commits**: One logical change per commit
- **Pre-commit tests**: Run `go test ./internal/core/services/...` before committing
- **Commit after each wave**: Complete functional unit

---

## Success Criteria

### Verification Commands
```bash
# Tests pass
go test ./internal/core/services/... -v

# Swagger docs generated
swag init

# Server starts
go run cmd/api/main.go

# Endpoints accessible
curl http://localhost:3000/api/v1/leads/{lead_id}/follow-ups/list
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] Authorization allows any user with permission
- [ ] Status validation prevents closed follow-up updates
- [ ] Tenant isolation working
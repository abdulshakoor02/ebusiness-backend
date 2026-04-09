# Facebook Ads Integration Backend

## TL;DR

> **Quick Summary**: Build Facebook Ads integration allowing admin users to connect their Facebook Ads accounts via OAuth, list running campaigns with forms, and receive leads via webhooks. Leads are auto-created with "Facebook Leads" category.

**Deliverables**:
- OAuth endpoints for Facebook authentication (per-tenant app)
- Facebook Ads Account management (multi-account support)
- Campaign listing endpoint
- Public webhook endpoint for lead ingestion
- Encrypted token storage (AES-256)
- Raw webhook logging for audit

**Estimated Effort**: Large (20+ tasks)

**Parallel Execution**: YES - 4 waves

**Critical Path**: Domain entities → Encryption utility → Repositories → Services → Handlers → Routes → Webhook

---

## Context

### Original Request
Build backend API to integrate Facebook Ads:
- Frontend can login to Facebook Ads accounts
- Store tokens with tenant ID (multi-tenant)
- Support multiple ads accounts per tenant
- Admin-only access via RBAC
- List running campaigns with forms
- Webhook receives leads on form submission
- Auto-create leads with "Facebook Leads" category

### Interview Summary
**Key Discussions**:
- **Facebook App**: Per-tenant apps (each tenant registers their own)
- **Token Storage**: Separate`facebook_ads_account` collection, encrypted with AES-256
- **Multi-Account**: Separate collection with `is_default` flag
- **Lead Source**: Auto-create "Facebook Ads" source
- **Lead Category**: Auto-create "Facebook Leads" category
- **Webhook**: Public endpoint with HMAC-SHA256 signature verification
- **Test Strategy**: TDD (tests first withstretchr/testify)

**Research Findings**:
- **OAuth Scopes**: ads_management, ads_read, leads_retrieval, business_management
- **Long-lived Tokens**: ~60 days, manual re-authorization when expired
- **Webhook Security**: HMAC-SHA256 with App Secret
- **Lead Expiry**: ~7 days to retrieve from Facebook

### Metis Review
**Identified Gaps** (addressed):
- Token expiration handling: Show status, manual re-authorization required
- Duplicate leads: Create all leads (no deduplication in MVP)
- Webhook security: HMAC signature + rate limiting consideration
- Missing fields: Create lead with partial data if incomplete
- Idempotent auto-creation: Handle existing source/category gracefully

---

## Work Objectives

### Core Objective
Enable admin users to connect Facebook Ads accounts and automatically create leads from Facebook Lead Ads webhooks.

### Concrete Deliverables
1. **Domain Entities**: `FacebookIntegration`, `FacebookAdsAccount`, `FacebookWebhookLog`
2. **Encryption Utility**: AES-256 encryption/decryption for sensitive data
3. **OAuth Flow**: Initiate, callback, token exchange
4. **Account Management**: Connect, disconnect, set default, list accounts
5. **Campaign Listing**: List running campaigns for connected accounts
6. **Webhook Handler**: Public endpoint for lead ingestion
7. **Permission Seeding**: RBAC permissions for admin role

### Definition of Done
- [ ] All domain entities created following existing patterns
- [ ] All repositories implemented with tenant isolation
- [ ] All services with business logic
- [ ] All handlers with Swagger documentation
- [ ] All routes registered in main.go
- [ ] Permission rules seeded in migrations.go
- [ ] All tests passing (TDD)
- [ ] Webhook signature verification working

### Must Have
- Per-tenant Facebook App (App ID, App Secret storage)
- AES-256 encrypted token storage
- HMAC-SHA256 webhook signature verification
- Multi-account support with default flag
- Auto-create "Facebook Ads" lead source
- Auto-create "Facebook Leads" lead category
- Raw webhook payload logging
- Admin-only access via RBAC

### Must NOT Have (Guardrails from Metis)
- NO custom field mapping (fixed mapping only)
- NO lead deduplication
- NO token refresh automation (manual re-authorization)
- NO campaign management (read-only)
- NO lead export to Facebook
- NO real-time notifications
- NO lead scoring
- NO A/B test tracking
- NO Facebook Business Manager management

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### TestDecision
- **Infrastructure exists**: YES (stretchr/testify)
- **Automated tests**: YES (TDD)
- **Framework**: `go test` with testify
- **TDD**: Each task follows RED (failing test) → GREEN (minimal impl) → REFACTOR

### QA Policy
Every task MUST include agent-executed QA scenarios.

- **API/Backend**: Use `curl` — Send requests, assert status + response fields
- **Database**: Use `go test` — Run test functions, assert collection state
- **Encryption**: Use `go test` — Assert encryption/decryption round-trip
- **Webhook**: Use `curl` — POST payloads, assert signature verification

---

## Execution Strategy

### Parallel Execution Waves

```
Wave1 (Foundation - 6 tasks):
├── Task 1: Domain entities [quick]
├── Task 2: Encryption utility [quick]
├── Task 3: Repository interfaces [quick]
├── Task 4:Permission rules seeding [quick]
├── Task 5: Service interfaces [quick]
└── Task6: Config updates (Facebook OAuth URLs) [quick]

Wave 2(after Wave 1 - Core Repos & Services -7 tasks):
├── Task 7: FacebookIntegration repository [quick]
├── Task 8: FacebookAdsAccount repository [quick]
├── Task 9: FacebookWebhookLog repository [quick]
├── Task 10: LeadSource repository update (add FindByName) [quick]
├── Task 11: LeadCategory repository update (add FindByName) [quick]
├── Task 12: FacebookIntegration service [unspecified-high]
└── Task 13: FacebookAdsAccount service [unspecified-high]

Wave 3 (after Wave 2 - Business Logic -5 tasks):
├── Task 14: Facebook webhook service [deep]
├── Task 15: Facebook campaign service [unspecified-high]
├── Task 16: Facebook integration handler (OAuth) [quick]
├── Task 17: Facebook ads account handler [quick]
└── Task 18: Facebook webhook handler [quick]

Wave 4 (after Wave 3 - Routes & Integration -3 tasks):
├── Task 19: Facebook campaign handler [quick]
├── Task 20: Route registration in main.go [quick]
└── Task 21: Integration tests [unspecified-high]

Wave FINAL (after ALL tasks -4 parallel reviews):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: Task1 → Task7 → Task12 → Task16 → Task20 → F1-F4 → user okay
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 7 (Wave 2)
```

### Dependency Matrix

- **Wave 1**: — —
- **Wave 2**: Depends on Wave 1complete
- **Wave 3**: Depends on Wave 2complete
- **Wave 4**: Depends on Wave 3 complete
- **Final**: Depends on all waves complete

---

## TODOs

- [ ] 1. Create Domain Entities

  **What to do**:
  - Create `internal/core/domain/facebook_integration.go` with `FacebookIntegration` entity (id, tenant_id, app_id, app_secret_encrypted, verify_token, created_at, updated_at)
  - Create `internal/core/domain/facebook_ads_account.go` with `FacebookAdsAccount` entity (id, tenant_id, facebook_integration_id, account_id, account_name, access_token_encrypted, is_default, expires_at, created_at, updated_at)
  - Create `internal/core/domain/facebook_webhook_log.go` with `FacebookWebhookLog` entity (id, tenant_id, ads_account_id, raw_payload, processed, lead_id, created_at)
  - Follow existing entity patterns from `lead_source.go` and `lead_category.go`
  - Add constructor functions for each entity

  **Must NOT do**:
  - Do NOT add custom field mapping
  - Do NOT add lead deduplication logic
  - Do NOT add complex validation beyond basic field existence

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Creating domain entities is straightforward following existing patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2-6)
  - **Blocks**: Tasks 7-9 (repositories)
  - **Blocked By**: None

  **References** (CRITICAL - Be Exhaustive):
  - `internal/core/domain/lead_source.go:1-27` - Entity structure pattern (ObjectID, TenantID, timestamps)
  - `internal/core/domain/lead_category.go:1-27` - Entity structure pattern
  - `internal/core/domain/lead.go:16-43` - Entity with multiple relations pattern

  **Acceptance Criteria**:
  - [ ] `internal/core/domain/facebook_integration.go` exists and compiles
  - [ ] `internal/core/domain/facebook_ads_account.go` exists and compiles
  - [ ] `internal/core/domain/facebook_webhook_log.go` exists and compiles
  - [ ] All entities have `bson` and `json` tags matching existing patterns
  - [ ] All entities have `New*()` constructor functions

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Domain entities compile and have correct structure
    Tool: Bash (go test -c)
    Preconditions: Clean working directory
    Steps:
      1. cd /Users/abdulshakooransari/projects/goCrmBackend
      2. go build ./internal/core/domain/...
    Expected Result: Build succeeds with no errors
    Failure Indicators: Compilation errors, missing imports
    Evidence: .sisyphus/evidence/task-01-domain-build.txt

  Scenario: FacebookIntegration entity has correct fields
    Tool: Bash (go test)
    Preconditions: Entities compiled
    Steps:
      1. Create test file `internal/core/domain/facebook_integration_test.go`
      2. Write test verifying NewFacebookIntegration creates entity with allrequired fields
      3. go test -v ./internal/core/domain/... -run TestFacebookIntegration
    Expected Result: Test passes, entity has id, tenant_id, app_id, app_secret_encrypted, verify_token fields
    Evidence: .sisyphus/evidence/task-01-entity-test.txt
  ```

  **Commit**: YES
  - Message: `feat(domain): add Facebook integration entities`
  - Files: `internal/core/domain/facebook_*.go`

---

- [ ] 2. Create Encryption Utility

  **What to do**:
  - Create `pkg/utils/encryption.go` with AES-256-GCM encryption/decryption functions
  - `Encrypt(plaintext string, key []byte) (string, error)` - Returns base64-encoded ciphertext
  - `Decrypt(ciphertext string, key []byte) (string, error)` - Takes base64-encoded ciphertext, returns plaintext
  - Use `crypto/aes` and `crypto/cipher` packages
  - Generate random nonce for each encryption
  - Add `ENCRYPTION_KEY` to config (32-byte key for AES-256)
  - Write comprehensive unit tests

  **Must NOT do**:
  - Do NOT use ECB mode (insecure)
  - DoNOT hardcode encryption keys
  - Do NOT skip nonce generation (security flaw)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Standard cryptographic utility following Go patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks1, 3-6)
  - **Blocks**: Task 12 (FacebookIntegration service)
  - **Blocked By**: None

  **References**:
  - `pkg/utils/jwt.go` - Existing utility pattern
  - Go docs: `crypto/aes` and `crypto/cipher`
  - `config/config.go:9-21` - Config structure pattern

  **Acceptance Criteria**:
  - [ ] `pkg/utils/encryption.go` exists and compiles
  - [ ] `pkg/utils/encryption_test.go` exists
  - [ ] `Encrypt` generates unique ciphertext for same input (nonce randomization)
  - [ ] `Decrypt(Encrypt(x, key), key) == x` for all inputs
  - [ ] `ENCRYPTION_KEY` added to config

  **QA Scenarios**:

  ```
  Scenario: Encryption round-trip succeeds
    Tool: Bash (go test)
    Preconditions: Encryption utility created
    Steps:
      1. Write test: encrypted, err := Encrypt("test-data", key)
      2. Write test: decrypted, err := Decrypt(encrypted, key)
      3. Assert decrypted == "test-data"
      4. go test -v ./pkg/utils/... -run TestEncryptDecrypt
    Expected Result: Test passes, encryption round-trip succeeds
    Evidence: .sisyphus/evidence/task-02-encrypt-test.txt

  Scenario: Unique nonce produces unique ciphertext
    Tool: Bash (go test)
    Preconditions: Encryption utility created
    Steps:
      1. Write test: enc1 := Encrypt("same", key)
      2. Write test: enc2 := Encrypt("same", key)
      3. Assert enc1 != enc2
    Expected Result: Test passes, different ciphertexts for same input
    Evidence: .sisyphus/evidence/task-02-nonce-test.txt
  ```

  **Commit**: YES
  - Message: `feat(utils): add AES-256 encryption utility`
  - Files: `pkg/utils/encryption.go`, `pkg/utils/encryption_test.go`

---

- [ ] 3. Create Repository Interfaces

  **What to do**:
  - Add `FacebookIntegrationRepository` interface to `internal/core/ports/repositories.go`
  - Add `FacebookAdsAccountRepository` interface to `internal/core/ports/repositories.go`
  - Add `FacebookWebhookLogRepository` interface to `internal/core/ports/repositories.go`
  - Follow existing repository interface patterns
  - Methods: Create, GetByID, GetByTenantID, Update, Delete, etc.

  **Must NOT do**:
  - Do NOT implement repositories yet (only interfaces)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 7-9 (repository implementations)
  - **Blocked By**: None

  **References**:
  - `internal/core/ports/repositories.go` - Existing repository interfaces
  - `LeadSourceRepository` interface pattern

  **Acceptance Criteria**:
  - [ ] `FacebookIntegrationRepository` interface in `ports/repositories.go`
  - [ ] `FacebookAdsAccountRepository` interface in `ports/repositories.go`
  - [ ] `FacebookWebhookLogRepository` interface in `ports/repositories.go`
  - [ ] Code compiles without errors

  **QA Scenarios**:

  ```
  Scenario: Repository interfaces compile
    Tool: Bash (go build)
    Steps:
      1. go build ./internal/core/ports/...
    Expected Result: Build succeeds
    Evidence: .sisyphus/evidence/task-03-interfaces-build.txt
  ```

  **Commit**: YES
  - Message: `feat(ports): add Facebook repository interfaces`
  - Files: `internal/core/ports/repositories.go`

---

- [ ] 4. Add Permission Rules Seeding

  **What to do**:
  - Add `facebook-integration` permission rules to `seedPermissionRules()` in `pkg/database/migrations.go`
  - Actions: `connect`, `disconnect`, `view-accounts`, `set-default`, `list-campaigns`
  - Add permission assignments to admin role in `seedRolePermissions()`
  - Follow existing permission rule patterns

  **Must NOT do**:
  - Do NOT add user role permissions (admin-only)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `pkg/database/migrations.go:143-293` - seedPermissionRules pattern
  - `pkg/database/migrations.go:295-471` - seedRolePermissions pattern

  **Acceptance Criteria**:
  - [ ] `facebook-integration` resource has 5 actions (connect, disconnect, view-accounts, set-default, list-campaigns)
  - [ ] All 5 actions assigned to admin role
  - [ ] Migration runs without errors

  **QA Scenarios**:

  ```
  Scenario: Permission rules seeded correctly
    Tool: Bash (go test)
    Steps:
      1. Write integration test verifying permission_rules collection has facebook-integration entries
      2. go test -v ./pkg/database/... -run TestSeedPermissionRules
    Expected Result: Test passes, facebook-integration rules exist
    Evidence: .sisyphus/evidence/task-04-permissions-test.txt
  ```

  **Commit**: YES
  - Message: `feat(migrations): add Facebook integration permissions`
  - Files: `pkg/database/migrations.go`

---

- [ ] 5. Create Service Interfaces

  **What to do**:
  - Add `FacebookIntegrationService` interface to `internal/core/ports/services.go`
  - Add `FacebookAdsAccountService` interface to `internal/core/ports/services.go`
  - Add `FacebookWebhookService` interface to `internal/core/ports/services.go`
  - Add `FacebookCampaignService` interface to `internal/core/ports/services.go`
  - Define required methods for OAuth, account management, webhook processing

  **Must NOT do**:
  - Do NOT implement services yet (only interfaces)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 12-15 (service implementations)
  - **Blocked By**: None

  **References**:
  - `internal/core/ports/services.go` - Existing service interfaces
  - `LeadSourceService` interface pattern

  **Commit**: YES
  - Message: `feat(ports): add Facebook service interfaces`
  - Files: `internal/core/ports/services.go`

---

- [ ] 6. Add Facebook Config

  **What to do**:
  - Add `FacebookOAuthURL`, `FacebookGraphAPIURL`, `EncryptionKey` to`config/config.go`
  - Add default values in `LoadConfig()`
  - Facebook OAuth URL: `https://www.facebook.com/v25.0/dialog/oauth`
  - Facebook Graph API URL: `https://graph.facebook.com/v25.0`
  - Encryption Key: Read from env var `ENCRYPTION_KEY` (32-byte hex string)

  **Must NOT do**:
  - Do NOT hardcode default encryption key

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: Task 12 (FacebookIntegration service)
  - **Blocked By**: None

  **References**:
  - `config/config.go:9-21` - Config struct pattern
  - `config/config.go:23-45` - LoadConfig pattern

  **Commit**: YES
  - Message: `feat(config): add Facebook OAuth and encryption config`
  - Files: `config/config.go`

---

Wave 2: Core Repositories & Services (depends on Wave 1)

---

- [ ] 7. Implement FacebookIntegration Repository

  **What to do**:
  - Create `internal/adapters/storage/mongo_facebook_integration_repo.go`
  - Implement `FacebookIntegrationRepository` interface
  - Methods: Create, GetByID, GetByTenantID, Update, Delete
  - Use tenant isolation via `getTenantIDFromContext(ctx)`
  - Add indexes on `tenant_id` and `app_id`
  - Write comprehensive unit tests

  **Must NOT do**:
  - Do NOT store unencrypted app_secret (must encrypt before storage)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8-13)
  - **Blocks**: Task 12 (FacebookIntegration service)
  - **Blocked By**: Task 1 (domain entity), Task 3 (interface)

  **References**:
  - `internal/adapters/storage/mongo_lead_source_repo.go` - Repository pattern
  - `internal/adapters/storage/mongo_tenant_repo.go` - Tenant isolation pattern

  **Commit**: YES
  - Message: `feat(storage): add FacebookIntegration repository`
  - Files: `internal/adapters/storage/mongo_facebook_integration_repo.go`, `internal/adapters/storage/mongo_facebook_integration_repo_test.go`

---

- [ ] 8. Implement FacebookAdsAccount Repository

  **What to do**:
  - Create `internal/adapters/storage/mongo_facebook_ads_account_repo.go`
  - Implement `FacebookAdsAccountRepository` interface
  - Methods: Create, GetByID, GetByTenantID, Update, Delete, SetDefault
  - Enforce single default account per tenant in `SetDefault`
  - Add indexes on `tenant_id` and `account_id`
  - Write comprehensive unit tests

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 13 (FacebookAdsAccount service)
  - **Blocked By**: Task 1 (domain entity), Task 3 (interface)

  **References**:
  - `internal/adapters/storage/mongo_lead_source_repo.go` - Repository pattern
  - `internal/adapters/storage/mongo_user_repo.go` - Multiple constraints pattern

  **Commit**: YES
  - Message: `feat(storage): add FacebookAdsAccount repository`
  - Files: `internal/adapters/storage/mongo_facebook_ads_account_repo.go`, `internal/adapters/storage/mongo_facebook_ads_account_repo_test.go`

---

- [ ] 9. Implement FacebookWebhookLog Repository

  **What to do**:
  - Create `internal/adapters/storage/mongo_facebook_webhook_log_repo.go`
  - Implement `FacebookWebhookLogRepository` interface
  - Methods: Create, GetByID, GetByTenantID, MarkProcessed
  - Add indexes on `tenant_id` and `created_at`
  - Write comprehensive unit tests

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 14 (FacebookWebhook service)
  - **Blocked By**: Task 1 (domain entity), Task 3 (interface)

  **References**:
  - `internal/adapters/storage/mongo_lead_source_repo.go` - Repository pattern

  **Commit**: YES
  - Message: `feat(storage): add FacebookWebhookLog repository`
  - Files: `internal/adapters/storage/mongo_facebook_webhook_log_repo.go`, `internal/adapters/storage/mongo_facebook_webhook_log_repo_test.go`

---

- [ ] 10. Update LeadSource Repository (Add FindByName)

  **What to do**:
  - Add `FindByName(ctx context.Context, tenantID primitive.ObjectID, name string) (*domain.LeadSource, error)` to `LeadSourceRepository` interface
  - Implement in `mongo_lead_source_repo.go`
  - Write test for FindByName

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 14 (FacebookWebhook service - needs to find "Facebook Ads" source)
  - **Blocked By**: None

  **References**:
  - `internal/core/ports/repositories.go` - Add to existing interface
  - `internal/adapters/storage/mongo_lead_source_repo.go` - Implement method

  **Commit**: YES
  - Message: `feat(storage): add FindByName to LeadSource repository`
  - Files: `internal/core/ports/repositories.go`, `internal/adapters/storage/mongo_lead_source_repo.go`

---

- [ ] 11. Update LeadCategory Repository (Add FindByName)

  **What to do**:
  - Add `FindByName(ctx context.Context, tenantID primitive.ObjectID, name string) (*domain.LeadCategory, error)` to `LeadCategoryRepository` interface
  - Implement in `mongo_lead_category_repo.go`
  - Write test for FindByName

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 14 (FacebookWebhook service - needs to find "Facebook Leads" category)
  - **Blocked By**: None

  **References**:
  - `internal/core/ports/repositories.go` - Add to existing interface
  - `internal/adapters/storage/mongo_lead_category_repo.go` - Implement method

  **Commit**: YES
  - Message: `feat(storage): add FindByName to LeadCategory repository`
  - Files: `internal/core/ports/repositories.go`, `internal/adapters/storage/mongo_lead_category_repo.go`

---

- [ ] 12. Implement FacebookIntegration Service

  **What to do**:
  - Create `internal/core/services/facebook_integration_service.go`
  - Implement `FacebookIntegrationService` interface
  - Methods:
    - `InitOAuth(tenantID, appID, appSecret, verifyToken)` - Store integration encrypted
    - `GetOAuthURL(integrationID)` - Return Facebook OAuth URL with proper scopes
    - `HandleCallback(code)` - Exchange code for long-lived token
    - `GetIntegrations(tenantID)` - List integrations for tenant
  - Use encryption utility for app_secret storage
  - Write comprehensive unit tests

  **Must NOT do**:
  - Do NOT store unencrypted app_secret
  - Do NOT skip token validation

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 16 (FacebookIntegration handler)
  - **Blocked By**: Task 2 (encryption utility), Task 7 (FacebookIntegration repository)

  **References**:
  - `internal/core/services/lead_source_service.go` - Service pattern
  - `pkg/utils/encryption.go` - Encryption utility
  - Facebook OAuth docs: https://developers.facebook.com/docs/facebook-login/guides/advanced/manual-flow

  **Commit**: YES
  - Message: `feat(service): add FacebookIntegration service with OAuth`
  - Files: `internal/core/services/facebook_integration_service.go`, `internal/core/services/facebook_integration_service_test.go`

---

- [ ] 13. Implement FacebookAdsAccount Service

  **What to do**:
  - Create `internal/core/services/facebook_ads_account_service.go`
  - Implement `FacebookAdsAccountService` interface
  - Methods:
    - `ConnectAccount(tenantID, integrationID, accountID, accountName, accessToken)` - Encrypt and store
    - `DisconnectAccount(tenantID, accountID)` - Delete account
    - `SetDefault(tenantID, accountID)` - Set default account
    - `GetAccounts(tenantID)` - List all accounts for tenant
    - `GetDefaultAccount(tenantID)` - Get default account
    - `GetDecryptedToken(accountID)` - Decrypt token for API calls
  - Use encryption utility for access_token storage
  - Write comprehensive unit tests

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 17 (FacebookAdsAccount handler), Task 15 (FacebookCampaign service)
  - **Blocked By**: Task 2 (encryption utility), Task 8 (FacebookAdsAccount repository)

  **References**:
  - `internal/core/services/lead_source_service.go` - Service pattern
  - `pkg/utils/encryption.go` - Encryption utility

  **Commit**: YES
  - Message: `feat(service): add FacebookAdsAccount service`
  - Files: `internal/core/services/facebook_ads_account_service.go`, `internal/core/services/facebook_ads_account_service_test.go`

---

Wave 3: Business Logic & Handlers (depends on Wave 2)

---

- [ ] 14. Implement Facebook Webhook Service

  **What to do**:
  - Create `internal/core/services/facebook_webhook_service.go`
  - Implement `FacebookWebhookService` interface
  - `VerifySignature(payload, signature, appSecret)` - HMAC-SHA256 verification
  - `ProcessWebhook(tenantID, payload)` - Parse lead data, create lead
  - Auto-create "Facebook Ads" lead source if not exists (idempotent)
  - Auto-create "Facebook Leads" lead category if not exists (idempotent)
  - Create lead with mapped fields (first_name, last_name, email, phone)
  - Store raw payload in `FacebookWebhookLog`
  - Use `FindByName` from LeadSourceRepository and LeadCategoryRepository
  - Write comprehensive unit tests

  **Must NOT do**:
  - Do NOT skip signature verification
  - Do NOT fail if source/category already exists (idempotent)
  - Do NOT deduplicate leads

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (critical business logic)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 18 (FacebookWebhook handler)
  - **Blocked By**: Task 9 (FacebookWebhookLog repository), Task 10 (LeadSource.FindByName), Task 11 (LeadCategory.FindByName)

  **References**:
  - `internal/core/services/lead_service.go` - Lead creation pattern
  - `pkg/utils/encryption.go` - For HMAC verification
  - Facebook Webhook docs: https://developers.facebook.com/docs/graph-api/webhooks/getting-started/

  **QA Scenarios**:

  ```
  Scenario: Webhook signature verification succeeds
    Tool: Bash (go test)
    Steps:
      1. Create test payload: {"object": "page", "entry": [...]}
      2. Generate valid HMAC-SHA256 with app secret
      3. Call VerifySignature with valid signature
      4. go test -v ./internal/core/services/... -run TestVerifySignature
    Expected Result: Verification returns true
    Evidence: .sisyphus/evidence/task-14-signature-valid.txt

  Scenario: Webhook signature verification fails
    Tool: Bash (go test)
    Steps:
      1. Create test payload
      2. Call VerifySignature with invalid signature
    Expected Result: Verification returns false
    Evidence: .sisyphus/evidence/task-14-signature-invalid.txt

  Scenario: Lead created with auto source/category
    Tool: Bash (go test)
    Steps:
      1. Mock repositories
      2. Call ProcessWebhook with valid payload
      3. Assert "Facebook Ads" source created or found
      4. Assert "Facebook Leads" category created or found
      5. Assert lead created with correct fields
    Expected Result: Lead created with source_id and category_id
    Evidence: .sisyphus/evidence/task-14-lead-creation.txt
  ```

  **Commit**: YES
  - Message: `feat(service): add Facebook webhook processing`
  - Files: `internal/core/services/facebook_webhook_service.go`, `internal/core/services/facebook_webhook_service_test.go`

---

- [ ] 15. Implement Facebook Campaign Service

  **What to do**:
  - Create `internal/core/services/facebook_campaign_service.go`
  - Implement `FacebookCampaignService` interface
  - `ListCampaigns(tenantID, accountID)` - Call Facebook Graph API
  - Use stored access token (decrypted) for API calls
  - Handle pagination (Facebook cursor-based)
  - Handle errors (token expired, rate limited, API down)
  - Return campaign name, status, objective
  - Write comprehensive unit tests with mocked HTTP

  **Must NOT do**:
  - Do NOT create campaigns (read-only)
  - Do NOT auto-refresh expired tokens (return error)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 19 (FacebookCampaign handler)
  - **Blocked By**: Task 13 (FacebookAdsAccount service - for token decryption)

  **References**:
  - Facebook Graph API docs: https://developers.facebook.com/docs/marketing-api/reference/ad-account/campaigns/
  - `net/http` - HTTP client for API calls

  **Commit**: YES
  - Message: `feat(service): add Facebook campaign listing`
  - Files: `internal/core/services/facebook_campaign_service.go`, `internal/core/services/facebook_campaign_service_test.go`

---

- [ ] 16. Implement Facebook Integration Handler

  **What to do**:
  - Create `internal/adapters/handler/facebook_integration_handler.go`
  - Implement handlers:
    - `POST /api/v1/facebook/integrations` - Create integration config
    - `GET /api/v1/facebook/integrations` - List integrations
    - `GET /api/v1/facebook/oauth/url` - Get OAuth URL
    - `GET /api/v1/facebook/oauth/callback` - Handle OAuth callback
  - Add Swagger documentation
  - Use RBAC middleware (admin only)
  - Write handler tests

  **Must NOT do**:
  - Do NOT skip RBAC middleware
  - Do NOT store unencrypted app_secret

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 20 (Route registration)
  - **Blocked By**: Task 12 (FacebookIntegration service)

  **References**:
  - `internal/adapters/handler/lead_source_handler.go` - Handler pattern
  - `internal/adapters/handler/auth_handler.go` - Auth/OAuth pattern

  **Commit**: YES
  - Message: `feat(handler): add Facebook integration OAuth handlers`
  - Files: `internal/adapters/handler/facebook_integration_handler.go`

---

- [ ] 17. Implement Facebook Ads Account Handler

  **What to do**:
  - Create `internal/adapters/handler/facebook_ads_account_handler.go`
  - Implement handlers:
    - `POST /api/v1/facebook/accounts` - Connect account
    - `DELETE /api/v1/facebook/accounts/:id` - Disconnect account
    - `PUT /api/v1/facebook/accounts/:id/default` - Set default
    - `GET /api/v1/facebook/accounts` - List accounts
  - Add Swagger documentation
  - Use RBAC middleware (admin only)
  - Write handler tests

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 20 (Route registration)
  - **Blocked By**: Task 13 (FacebookAdsAccount service)

  **References**:
  - `internal/adapters/handler/lead_source_handler.go` - Handler pattern

  **Commit**: YES
  - Message: `feat(handler): add Facebook ads account handlers`
  - Files: `internal/adapters/handler/facebook_ads_account_handler.go`

---

- [ ] 18. Implement Facebook Webhook Handler

  **What to do**:
  - Create `internal/adapters/handler/facebook_webhook_handler.go`
  - Implement handlers:
    - `GET /webhook/facebook` - Verification challenge (respond with hub.challenge)
    - `POST /webhook/facebook` - Webhook payload (verify signature, process lead)
  - **PUBLIC endpoint** (no JWT/RBAC middleware)
  - Verify `X-Hub-Signature-256` header
  - Verify `hub.mode=subscribe` and `hub.verify_token` for GET
  - Use tenant lookup by verify_token
  - Write handler tests

  **Must NOT do**:
  - Do NOT add JWT/RBAC middleware (public endpoint)
  - Do NOT skip signature verification

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 20 (Route registration)
  - **Blocked By**: Task 14 (FacebookWebhook service)

  **References**:
  - Facebook Webhook docs: https://developers.facebook.com/docs/graph-api/webhooks/getting-started/
  - `pkg/middleware/auth_middleware.go` - What NOT to use (public endpoint)

  **QA Scenarios**:

  ```
  Scenario: Webhook verification challenge succeeds
    Tool: Bash (curl)
    Steps:
      1. curl -X GET "http://localhost:3000/webhook/facebook?hub.mode=subscribe&hub.verify_token=<token>&hub.challenge=test123"
    Expected Result: Response body is "test123"
    Evidence: .sisyphus/evidence/task-18-challenge.txt

  Scenario: Webhook processes lead
    Tool: Bash (curl)
    Steps:
      1. Generate test payload
      2. curl -X POST http://localhost:3000/webhook/facebook -H "Content-Type: application/json" -H "X-Hub-Signature-256: sha256=<valid>" -d '<payload>'
    Expected Result: 200 OK, lead created
    Evidence: .sisyphus/evidence/task-18-webhook.txt
  ```

  **Commit**: YES
  - Message: `feat(handler): add Facebook webhook handler`
  - Files: `internal/adapters/handler/facebook_webhook_handler.go`

---

Wave 4: Routes & Integration (depends on Wave 3)

---

- [ ] 19. Implement Facebook Campaign Handler

  **What to do**:
  - Create `internal/adapters/handler/facebook_campaign_handler.go`
  - Implement handler:
    - `GET /api/v1/facebook/campaigns?account_id=<id>` - List campaigns
  - Use stored access token for API call
  - Handle errors (token expired, API unavailable)
  - Add Swagger documentation
  - Use RBAC middleware (admin only)
  - Write handler tests

  **Must NOT do**:
  - Do NOT skip error handling

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4
  - **Blocks**: Task 20 (Route registration)
  - **Blocked By**: Task 15 (FacebookCampaign service)

  **References**:
  - `internal/adapters/handler/lead_source_handler.go` - Handler pattern

  **Commit**: YES
  - Message: `feat(handler): add Facebook campaign handler`
  - Files: `internal/adapters/handler/facebook_campaign_handler.go`

---

- [ ] 20. Register Routes in main.go

  **What to do**:
  - Add Facebook integration routes to `cmd/api/main.go`
  - Protected routes (require JWT + RBAC):
    - `POST /api/v1/facebook/integrations`
    - `GET /api/v1/facebook/integrations`
    - `GET /api/v1/facebook/oauth/url`
    - `GET /api/v1/facebook/oauth/callback`
    - `POST /api/v1/facebook/accounts`
    - `GET /api/v1/facebook/accounts`
    - `DELETE /api/v1/facebook/accounts/:id`
    - `PUT /api/v1/facebook/accounts/:id/default`
    - `GET /api/v1/facebook/campaigns`
  - Public routes (no auth):
    - `GET /webhook/facebook` (verification)
    - `POST /webhook/facebook` (webhook)
  - Initialize repositories and services in main.go
  - Wire up all dependencies

  **Must NOT do**:
  - Do NOT add auth middleware to webhook routes

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (integration point)
  - **Parallel Group**: Wave 4
  - **Blocks**: None (final task)
  - **Blocked By**: Tasks 16-19 (all handlers)

  **References**:
  - `cmd/api/main.go:114-231` - Route registration pattern

  **QA Scenarios**:

  ```
  Scenario: All routes registered correctly
    Tool: Bash (curl)
    Steps:
      1. Start server: go run ./cmd/api/main.go
      2. GET /swagger/doc.json
      3. Assert all facebook routes appear in swagger
    Expected Result: All routes documented in swagger
    Evidence: .sisyphus/evidence/task-20-routes.txt

  Scenario: Webhook is public (no auth required)
    Tool: Bash (curl)
    Steps:
      1. curl -X GET "http://localhost:3000/webhook/facebook?hub.mode=subscribe&hub.verify_token=test&hub.challenge=test123"
    Expected Result: 200 OK, returns "test123" (not 401 Unauthorized)
    Evidence: .sisyphus/evidence/task-20-webhook-public.txt
  ```

  **Commit**: YES
  - Message: `feat(routes): register Facebook integration routes`
  - Files: `cmd/api/main.go`

---

- [ ] 21. Integration Tests

  **What to do**:
  - Create `integration_test.go` with full OAuth flow test
  - Create `webhook_test.go` with full webhook flow test
  - Create `campaign_test.go` with campaign listing test
  - Test with mocked Facebook API responses
  - Test end-to-end: OAuth → Account Connect → Campaign List → Webhook → Lead Created

  **Must NOT do**:
  - Do NOT test against real Facebook API (use mocks)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Task 20 (all routes registered)

  **References**:
  - `internal/core/services/auth_service_test.go` - Test pattern
  - `internal/core/services/lead_service_test.go` - Test pattern

  **Commit**: YES
  - Message: `test(integration): add Facebook integration tests`
  - Files: `internal/integration/facebook_test.go`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

>4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go build ./...` + `go test ./...`. Review all changed files for: commented-out code, unused imports, missing error handling, hardcoded values. Check AI slop: excessive comments, over-abstraction, generic names. Verify all handlers have Swagger docs.
  Output: `Build [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration: OAuth → Account → Campaign → Webhook. Test edge cases: invalid signature, missing fields, duplicate source/category.
  Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's files. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

Each task commit follows conventional commits format:

```
feat(domain): add Facebook integration entities
feat(utils): add AES-256 encryption utility
feat(ports): add Facebook repository interfaces
feat(migrations): add Facebook integration permissions
feat(ports): add Facebook service interfaces
feat(config): add Facebook OAuth and encryption config
feat(storage): add FacebookIntegration repository
feat(storage): add FacebookAdsAccount repository
feat(storage): add FacebookWebhookLog repository
feat(storage): add FindByName to LeadSource repository
feat(storage): add FindByName to LeadCategory repository
feat(service): add FacebookIntegration service with OAuth
feat(service): add FacebookAdsAccount service
feat(service): add Facebook webhook processing
feat(service): add Facebook campaign listing
feat(handler): add Facebook integration OAuth handlers
feat(handler): add Facebook ads account handlers
feat(handler): add Facebook webhook handler
feat(handler): add Facebook campaign handler
feat(routes): register Facebook integration routes
test(integration): add Facebook integration tests
```

---

## Success Criteria

### Verification Commands
```bash
# Build all packages
go build ./...

# Run all tests
go test -v ./...

# Run specific test packages
go test -v ./internal/core/services/... -run TestFacebook
go test -v ./internal/adapters/handler/... -run TestFacebook

# Check permission seeding
go test -v ./pkg/database/... -run TestSeedPermissionRules

# Verify webhook endpoint (public)
curl -X GET "http://localhost:3000/webhook/facebook?hub.mode=subscribe&hub.verify_token=test&hub.challenge=test123"
# Expected: test123

# Verify OAuth URL (requires auth token)
curl -X GET "http://localhost:3000/api/v1/facebook/oauth/url" -H "Authorization: Bearer <admin_token>"
# Expected: {"redirect_url": "https://www.facebook.com/v25.0/dialog/oauth?..."}
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] Webhook endpoint returns 200 without auth
- [ ] OAuth endpoints require admin auth
- [ ] Tokens are encrypted in database
- [ ] Leads created with Facebook source and category
- [ ] Campaign listing returns data from Facebook API
- [ ] Permission rules seeded for admin role
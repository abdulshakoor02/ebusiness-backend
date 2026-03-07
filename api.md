# Go CRM Backend API Documentation

Base URL: `http://localhost:3000/api/v1`

All protected endpoints require an Authorization header with a Bearer token:
`Authorization: Bearer <your_jwt_token>`

---

## API ID Reference Guide (Important for Frontend)

This section explains all the IDs used in the API and where they come from:

### ID Types and Their Sources

| ID Field | Type | Description | Where to Get It |
|----------|------|-------------|-----------------|
| `id` | MongoDB ObjectID | Unique document identifier | Returned in `_id` or `id` field from any GET request |
| `tenant_id` | MongoDB ObjectID | Tenant/organization ID | From login response or user's `tenant_id` |
| `user_id` | MongoDB ObjectID | User identifier | From login response or from `/users/:id` endpoint |
| `role` | String | User role (admin/user/superadmin) | From login response |
| `permission_rule_id` | MongoDB ObjectID | Reference to a permission rule | From `/permissions/available-rules` endpoint |
| `rule_id` | MongoDB ObjectID | Same as `permission_rule_id` | From `/permissions/available-rules` endpoint |

### Common Workflows for Frontend

#### 1. Getting Available Permission Rules (for building UI)
```
GET /permissions/available-rules
```
**Response:** Returns list of all permission rules with their `id` (permission_rule_id).
Use this to:
- Build the permissions management UI
- Populate dropdowns for selecting permissions
- Show available actions for each resource

#### 2. Assigning Permissions to a Role
```
POST /permissions/roles/:role/bulk
```
**Request body:**
```json
{
  "permissions": [
    {
      "id": "507f1f77bcf86cd799439011",  // ← permission_rule_id from /permissions/available-rules
      "assigned": true
    }
  ]
}
```

#### 3. Managing Entities (Leads, Categories, etc.)
All entity endpoints use the same ID pattern:
- **Create:** POST returns `{ "id": "...", ... }` - save this for updates/deletes
- **Read:** GET `/:id` - use the ID from create or list response
- **Update:** PUT `/:id` - use the ID from create or list response  
- **Delete:** DELETE `/:id` - use the ID from create or list response
- **List:** POST `/list` returns items with `id` field in each object

---

## Role-Based Access Control (RBAC)

The system uses custom RBAC with MongoDB-based permission storage:

**Role Hierarchy:**
```
superadmin (inherits all admin permissions)
    └── admin (full system access)
        └── user (limited access)
```

**Collections:**
- `permission_rules` - Defines available permissions (path, method, scope_type, filter_field)
- `role_permissions` - Maps roles to permission_rule_id (references permission_rules)
- `role_inheritances` - Defines role inheritance relationships

**Role Permissions Summary:**

| Role | Permissions |
|------|-------------|
| **superadmin** | Inherits all admin permissions |
| **admin** | Full CRUD on tenants, users, leads, lead-categories, lead-sources, lead-comments, lead-appointments, and permission management |
| **user** | View/update own tenant, view own user, full access to leads/comments/appointments, read-only access to categories/sources |

**Auth Requirements Legend:**
- **No Auth** - Public endpoint, no authentication required
- **JWT Auth** - Requires valid JWT token
- **JWT + RBAC: Admin** - Requires JWT + admin role
- **JWT + RBAC: All** - Requires JWT + any role (admin or user)

### Data Scoping (Row-Level Security)

The system supports **row-level security** via permission rules. Each permission rule can have a `scope_type` that controls data access:

| Scope Type | Description | Example Use Case |
|------------|-------------|------------------|
| `none` | No filtering - user sees all data within tenant | Admin sees all leads |
| `self` | Filters to current user's records only | User sees only their assigned leads |
| `group` | Filters to user's team/group (future) | Manager sees their team's leads |

**How it works:**
1. Admin assigns a permission rule with `scope_type: "self"` to a role (e.g., `list_own`)
2. When user calls the endpoint, middleware checks the permission's scope config
3. Repository automatically filters records where `filter_field` matches user's ID

> **Frontend Note:** When fetching `/permissions/available-rules`, look for rules with:
> - `action` ending in `_own` (e.g., `list_own`, `view_own`, `update_own`)
> - `scope_type: "self"` 
> - `filter_field`: the field name to filter by (e.g., `assigned_to`, `created_by`)
>
> These represent permissions that should only show the user's own data.

**Example - Available scoped rules from `/permissions/available-rules`:**
```json
{
  "resource": "leads",
  "action": "list_own",
  "scope_type": "self",
  "filter_field": "assigned_to"
}
```

**Example - Assigning own-scope permission via bulk update:**
```json
{
  "permissions": [
    {
      "id": "60a7e...123",  // permission_rule_id from /permissions/available-rules
      "assigned": true
    }
  ]
}
```

---

## 1. Authentication

### User Login
**Endpoint:** `POST /auth/login`
**Auth Required:** No

**Request:**
```json
{
  "email": "user@example.com",
  "password": "secretpassword"
}
```

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR...",
  "user": {
    "id": "60a7e...",
    "tenant_id": "60a7e...",
    "name": "Admin User",
    "email": "user@example.com",
    "mobile": "+1234567890",
    "role": "admin",
    "created_at": "2026-02-15T12:00:00Z",
    "updated_at": "2026-02-15T12:00:00Z"
  },
  "tax": 5.0,
  "currency": "AED",
  "next_cloud_folder": "ebusiness"
}
```

> **Note:** The `tax`, `currency`, and `next_cloud_folder` fields are only included for non-superadmin users (admin and user roles). These values are fetched from the user's tenant configuration.
> - `tax`: Tax percentage from tenant settings (e.g., 5.0 for 5%)
> - `currency`: Currency code from tenant's country (e.g., "AED", "USD")
> - `next_cloud_folder`: Tenant name for Nextcloud folder path (e.g., "ebusiness")

**Response for Superadmin:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR...",
  "user": {
    "id": "60a7e...",
    "name": "Super Admin",
    "email": "superadmin@example.com",
    "role": "superadmin",
    "created_at": "2026-02-15T12:00:00Z",
    "updated_at": "2026-02-15T12:00:00Z"
  }
}
```

---

## 2. Tenants (Organizations)

### Register a New Tenant
**Endpoint:** `POST /tenants`
**Auth Required:** No

**Request:**
```json
{
  "name": "Acme Corp",
  "email": "admin@acmecorp.com",
  "logo_url": "https://example.com/logo.png",
  "address": {
    "street": "123 Tech Lane",
    "address_line": "Suite 400",
    "city": "San Francisco",
    "state": "CA",
    "country": "USA",
    "zip_code": "94105"
  },
  "country_id": "60c9g...",
  "tax": 5.0,
  "admin_user": {
    "name": "Jane Doe",
    "email": "jane@acmecorp.com",
    "mobile": "+1234567890",
    "password": "strongpassword123",
    "role": "admin"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `country_id` | string | No | MongoDB ObjectID of the country |
| `tax` | number | No | Tax percentage (e.g., 5.0 for 5%) |

**Response (201 Created):**
```json
{
  "tenant_id": "60a7e...",
  "user_id": "60b8f...",
  "message": "Tenant created successfully"
}
```

**Response (409 Conflict):**
```json
{
  "error": "Email or mobile already exists"
}
```

### Get Tenant by ID
**Endpoint:** `GET /tenants/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)
*Users can only view their own tenant.*

**Response (200 OK):**
```json
{
  "id": "60a7e...",
  "name": "Acme Corp",
  "email": "admin@acmecorp.com",
  "logo_url": "https://example.com/logo.png",
  "address": {
    "street": "123 Tech Lane",
    "address_line": "Suite 400",
    "city": "San Francisco",
    "state": "CA",
    "country": "USA",
    "zip_code": "94105"
  },
  "country_id": "60c9g...",
  "tax": 5.0,
  "next_invoice_number": 1,
  "next_receipt_number": 1,
  "created_at": "2026-02-15T10:00:00Z",
  "updated_at": "2026-02-15T10:00:00Z"
}
```

### Update Tenant
**Endpoint:** `PUT /tenants/:id`
**Auth Required:** JWT + RBAC: Admin only

**Request:** (All fields are optional)
```json
{
  "name": "Acme Corporation Inc",
  "logo_url": "https://example.com/newlogo.png",
  "address": {
    "city": "San Jose"
  },
  "country_id": "60c9g...",
  "tax": 7.5
}
```

| Field | Type | Description |
|-------|------|-------------|
| `country_id` | string | MongoDB ObjectID of the country |
| `tax` | number | Tax percentage (e.g., 5.0 for 5%) |

**Response (200 OK):**
```json
{
  "id": "60a7e...",
  "name": "Acme Corporation Inc",
  "email": "admin@acmecorp.com",
  "logo_url": "https://example.com/newlogo.png",
  "address": {
    "street": "123 Tech Lane",
    "address_line": "Suite 400",
    "city": "San Jose",
    "state": "CA",
    "country": "USA",
    "zip_code": "94105"
  },
  "country_id": "60c9g...",
  "tax": 7.5,
  "next_invoice_number": 1,
  "next_receipt_number": 1,
  "created_at": "2026-02-15T10:00:00Z",
  "updated_at": "2026-02-15T14:30:00Z"
}
  "created_at": "2026-02-15T10:00:00Z",
  "updated_at": "2026-02-15T14:30:00Z"
}
```

### List Tenants
**Endpoint:** `POST /tenants/list`
**Auth Required:** JWT + RBAC: Admin only

**Request:**
```json
{
  "filters": {
    "name": "Acme"
  },
  "offset": 0,
  "limit": 10
}
```

**Response (200 OK):**
```json
{
  "data": [
    { /* Tenant Object 1 */ },
    { /* Tenant Object 2 */ }
  ],
  "total": 2,
  "offset": 0,
  "limit": 10
}
```

---

## 3. Users

### Create a User
**Endpoint:** `POST /users`
**Auth Required:** JWT + RBAC: Admin only

**Request:**
```json
{
  "name": "John Smith",
  "email": "john@acmecorp.com",
  "mobile": "+1987654321",
  "password": "securepassword123",
  "role": "user"
}
```

**Response (201 Created):**
```json
{
  "id": "60b8f...",
  "tenant_id": "60a7e...",
  "name": "John Smith",
  "email": "john@acmecorp.com",
  "mobile": "+1987654321",
  "role": "user",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T12:00:00Z"
}
```

**Response (409 Conflict):**
```json
{
  "error": "Email or mobile already exists"
}
```

### Get User by ID
**Endpoint:** `GET /users/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)
*Users can only view their own user profile.*

**Response (200 OK):**
```json
{
  "id": "60b8f...",
  "tenant_id": "60a7e...",
  "name": "John Smith",
  "email": "john@acmecorp.com",
  "mobile": "+1987654321",
  "role": "user",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T12:00:00Z"
}
```

### Update User
**Endpoint:** `PUT /users/:id`
**Auth Required:** JWT + RBAC: Admin only

**Request:** (All fields are optional)
```json
{
  "name": "John Smith Updated",
  "email": "john.updated@acmecorp.com",
  "mobile": "+1555555555",
  "role": "manager"
}
```

**Response (200 OK):**
```json
{
  "id": "60b8f...",
  "tenant_id": "60a7e...",
  "name": "John Smith Updated",
  "email": "john.updated@acmecorp.com",
  "mobile": "+1555555555",
  "role": "manager",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T14:30:00Z"
}
```

### List Users
**Endpoint:** `POST /users/list`
**Auth Required:** JWT + RBAC: Admin only

**Request:**
```json
{
  "filters": {
    "role": "user"
  },
  "offset": 0,
  "limit": 10
}
```

**Response (200 OK):** (Returns paginated Array of User objects)

---

## 4. Permissions & RBAC Management

### Get My UI Permissions
**Endpoint:** `GET /permissions/my-permissions`
**Auth Required:** Yes (Not RBAC restricted)
*Fetches allowed capabilities for the currently logged-in user to drive frontend UI.*

**Response (200 OK):**
```json
{
  "role": "admin",
  "permissions": {
    "can_create_user": true,
    "can_list_tenants": true,
    "can_list_users": true,
    "can_update_tenant": true,
    "can_update_user": true,
    "can_view_tenant": true,
    "can_view_user": true
  }
}
```

### Add a Permission to a Role
**Endpoint:** `POST /permissions`
**Auth Required:** Yes (RBAC Enforced - Admin only)

**Request:**
```json
{
  "role": "manager",
  "permission_rule_id": "507f1f77bcf86cd799439011"
}
```

**Response (201 Created):**
```json
{
  "message": "Permission added successfully"
}
```

### Remove a Permission from a Role
**Endpoint:** `DELETE /permissions`
**Auth Required:** Yes (RBAC Enforced - Admin only)

**Request:**
```json
{
  "role": "manager",
  "permission_rule_id": "507f1f77bcf86cd799439011"
}
```

**Response (200 OK):**
```json
{
  "message": "Permission removed successfully"
}
```

### Get All Roles
**Endpoint:** `GET /permissions/roles`
**Auth Required:** Yes (RBAC Enforced - Admin only)
*Retrieves all available roles in the system.*

**Response (200 OK):**
```json
{
  "data": ["admin", "user", "superadmin"]
}
```

### Get Role Permissions (UI Grouped)
**Endpoint:** `GET /permissions/roles/:role`
**Auth Required:** Yes (RBAC Enforced - Admin only)
*Retrieves all available rules in the system dynamically grouped by resource, alongside an 'assigned' boolean reflecting if this role possesses the rule.*

> **Frontend Tip:** Use this to show current permissions for a role in your admin UI. The `assigned: true` indicates the role already has that permission.

**Response (200 OK):**
```json
{
  "role": "admin",
  "resources": [
    {
      "resource": "users",
      "label": "User Management",
      "rules": [
        {
          "id": "60a7e...",          // ← permission_rule_id
          "resource": "users",
          "resource_label": "User Management",
          "action": "create",
          "action_label": "Create User",
          "path": "/api/v1/users",
          "method": "POST",
          "description": "Allows creating new users",
          "is_system": true,
          "scope_type": "none",
          "filter_field": "",
          "assigned": true            // ← true = permission is assigned to this role
        },
        {
          "id": "60a7f...",
          "resource": "leads",
          "resource_label": "Lead Management",
          "action": "list_own",
          "action_label": "List Own Leads",
          "path": "/api/v1/leads/list",
          "method": "POST",
          "description": "List leads assigned to self",
          "is_system": true,
          "scope_type": "self",       // ← scoping type
          "filter_field": "assigned_to", // ← field to filter by
          "assigned": false          // ← false = permission not assigned
        }
      ]
    }
  ]
}
```

### Bulk Update Role Permissions
**Endpoint:** `POST /permissions/roles/:role/bulk`
**Auth Required:** Yes (RBAC Enforced - Admin only)
*Allows adding or removing multiple permissions for a role in a single request. Set `assigned: true` to add a permission, or `assigned: false` to remove it.*

> **How to use:** 
> 1. First call `GET /permissions/available-rules` to get all available permission rules
> 2. Extract the `id` field from each rule - this is the `permission_rule_id`
> 3. Use that `id` in the `permissions` array below
> 4. Set `assigned: true` to grant the permission, or `assigned: false` to revoke

**Request:**
```json
{
  "permissions": [
    {
      "id": "507f1f77bcf86cd799439011",  // ← permission_rule_id from /permissions/available-rules
      "resource": "users",
      "action": "create",
      "assigned": true                   // ← true = grant, false = revoke
    },
    {
      "id": "507f1f77bcf86cd799439012",  // ← another permission_rule_id
      "resource": "leads",
      "action": "delete",
      "assigned": false                  // ← remove this permission
    }
  ]
}
```

| Field | Description |
|-------|-------------|
| `id` | Permission rule ID from `/permissions/available-rules` |
| `resource` | Resource name (e.g., "users", "leads") |
| `action` | Action name (e.g., "create", "list", "list_own") |
| `assigned` | `true` = add permission, `false` = remove |

| Field | Description |
|-------|-------------|
| `path` | API endpoint path (e.g., `/api/v1/leads/list`) |
| `method` | HTTP method (GET, POST, PUT, DELETE, or `*` for wildcard) |
| `assigned` | `true` = add permission, `false` = remove permission |

**Response (200 OK):**
```json
{
  "message": "Role permissions synchronized successfully"
}
```

### Get All Rules / Policies
**Endpoint:** `GET /permissions`
**Auth Required:** Yes (RBAC Enforced - Admin only)

**Response (200 OK):**
```json
{
  "data": [
    ["admin", "/api/v1/tenants/:id", "GET"],
    ["admin", "/api/v1/users", "POST"],
    ["user", "/api/v1/users/:id", "GET"]
  ]
}
```

### Assign Role Inheritance
**Endpoint:** `POST /permissions/roles/inherit`
**Auth Required:** Yes (RBAC Enforced - Admin only)
*Allows a child role to inherit all policies from a parent role.*

**Request:**
```json
{
  "child_role": "superadmin",
  "parent_role": "admin"
}
```

**Response (201 Created):**
```json
{
  "message": "Role inheritance assigned successfully"
}
```

### Get All Role Inheritances
**Endpoint:** `GET /permissions/roles/inherit`
**Auth Required:** Yes (RBAC Enforced - Admin only)

**Response (200 OK):**
```json
{
  "data": [
    ["superadmin", "admin"]
  ]
}
```

### Get Available Permission Rules
**Endpoint:** `GET /permissions/available-rules`
**Auth Required:** Yes (RBAC Enforced - Admin only)
*Returns all available permission rules organized by resource with human-readable labels for frontend dropdowns.*

> **Frontend Tip:** Use this endpoint to build your permissions management UI. The `id` field in each rule is the `permission_rule_id` you need when assigning permissions via bulk update.

**Response (200 OK):**
```json
{
  "resources": [
    {
      "resource": "tenants",
      "label": "Tenant Management",
      "rules": [
        {
          "id": "60a7e...",  // ← This is permission_rule_id - use this when assigning permissions
          "resource": "tenants",
          "resource_label": "Tenant Management",
          "action": "view",
          "action_label": "View Tenant Details",
          "path": "/api/v1/tenants/:id",
          "method": "GET",
          "description": "View tenant information",
          "is_system": true,
          "scope_type": "none",
          "filter_field": "",
          "created_at": "2026-02-27T10:00:00Z",
          "updated_at": "2026-02-27T10:00:00Z"
        },
        {
          "id": "60a7f...",  // ← permission_rule_id for "list_own" - includes scope_type
          "resource": "leads",
          "resource_label": "Lead Management",
          "action": "list_own",
          "action_label": "List Own Leads",
          "path": "/api/v1/leads/list",
          "method": "POST",
          "description": "List leads assigned to self",
          "is_system": true,
          "scope_type": "self",        // ← Data scoping type
          "filter_field": "assigned_to", // ← Field to filter by current user ID
          "created_at": "2026-02-27T10:00:00Z",
          "updated_at": "2026-02-27T10:00:00Z"
        }
      ]
    }
  ]
}
```

### Create Permission Rule
**Endpoint:** `POST /permissions/rules`
**Auth Required:** Yes (RBAC Enforced - Admin only)
*Creates a custom permission rule that can be endpoint-based or frontend-only (empty path/method). Supports data scoping via scope_type.*

**Request:**
```json
{
  "resource": "custom-dashboard",
  "resource_label": "Custom Dashboard",
  "action": "access",
  "action_label": "Access Dashboard",
  "path": "",
  "method": "",
  "description": "Access to custom analytics dashboard",
  "scope_type": "self",
  "filter_field": "user_id"
}
```

**Scope Types:**
| Scope Type | Description |
|------------|-------------|
| `none` | No data filtering - user sees all (default) |
| `self` | Filter by current user's ID in the specified field |
| `group` | Filter by group/team user IDs (future) |

**Response (201 Created):**
```json
{
  "message": "Permission rule created successfully",
  "data": {
    "id": "60b9c...",
    "resource": "custom-dashboard",
    "resource_label": "Custom Dashboard",
    "action": "access",
    "action_label": "Access Dashboard",
    "path": "",
    "method": "",
    "description": "Access to custom analytics dashboard",
    "is_system": false,
    "created_at": "2026-02-27T12:00:00Z",
    "updated_at": "2026-02-27T12:00:00Z"
  }
}
```

**Response (400 Bad Request):**
```json
{
  "error": "resource and action are required"
}
```

**Response (409 Conflict):**
```json
{
  "error": "permission rule already exists for this resource and action"
}
```

### Update Permission Rule
**Endpoint:** `PUT /permissions/rules/:id`
**Auth Required:** Yes (RBAC Enforced - Admin only)
*Updates an existing permission rule. System rules can only have labels/description updated. Custom rules can update all fields.*

**Request:**
```json
{
  "resource_label": "Updated Dashboard",
  "action_label": "Access Analytics Dashboard",
  "description": "Updated description"
}
```

**Response (200 OK):**
```json
{
  "message": "Permission rule updated successfully",
  "data": {
    "id": "60b9c...",
    "resource": "custom-dashboard",
    "resource_label": "Updated Dashboard",
    "action": "access",
    "action_label": "Access Analytics Dashboard",
    "path": "",
    "method": "",
    "description": "Updated description",
    "is_system": false,
    "created_at": "2026-02-27T12:00:00Z",
    "updated_at": "2026-02-27T12:30:00Z"
  }
}
```

**Response (400 Bad Request):**
```json
{
  "error": "Invalid rule ID"
}
```

### Delete Permission Rule
**Endpoint:** `DELETE /permissions/rules/:id`
**Auth Required:** Yes (RBAC Enforced - Admin only)
*Deletes a custom permission rule. System rules cannot be deleted.*

**Response (200 OK):**
```json
{
  "message": "Permission rule deleted successfully"
}
```

**Response (400 Bad Request):**
```json
{
  "error": "Invalid rule ID"
}
```

**Response (500 Internal Server Error):**
```json
{
  "error": "cannot delete system permission rules"
}
```

---

## 5. Leads

### Create Lead
**Endpoint:** `POST /leads`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Request:**
```json
{
  "assigned_to": "60b8f...",
  "category_id": "60b8f...",
  "source_id": "60b8f...",
  "country_id": "60c9g...",
  "qualification_id": "60d0h...",
  "first_name": "Alice",
  "last_name": "Johnson",
  "designation": "Software Engineer",
  "email": "alice@techinnovations.com",
  "phone": "+1987654321"
}
```

**Response (201 Created):**
```json
{
  "id": "60c9g...",
  "tenant_id": "60a7e...",
  "assigned_to": "60b8f...",
  "category_id": "60b8f...",
  "source_id": "60b8f...",
  "country_id": "60c9g...",
  "qualification_id": "60d0h...",
  "first_name": "Alice",
  "last_name": "Johnson",
  "designation": "Software Engineer",
  "email": "alice@techinnovations.com",
  "phone": "+1987654321",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T12:00:00Z"
}
```

### Get Lead by ID
**Endpoint:** `GET /leads/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Response (200 OK):**
```json
{
  "id": "60c9g...",
  "tenant_id": "60a7e...",
  "assigned_to": "60b8f...",
  "category_id": "60b8f...",
  "source_id": "60b8f...",
  "country_id": "60c9g...",
  "qualification_id": "60d0h...",
  "first_name": "Alice",
  "last_name": "Johnson",
  "designation": "Software Engineer",
  "email": "alice@techinnovations.com",
  "phone": "+1987654321",
  "status": "lead",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T12:00:00Z"
}
```

### Update Lead
**Endpoint:** `PUT /leads/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Request (Partial update supported):**
```json
{
  "source_id": "60b8f...",
  "category_id": "60b8f...",
  "designation": "Senior Software Engineer"
}
```

**Response (200 OK):**
```json
{
  "id": "60c9g...",
  "tenant_id": "60a7e...",
  "assigned_to": "60b8f...",
  "category_id": "60b8f...",
  "source_id": "60b8f...",
  "country_id": "60c9g...",
  "qualification_id": "60d0h...",
  "first_name": "Alice",
  "last_name": "Johnson",
  "designation": "Senior Software Engineer",
  "email": "alice@techinnovations.com",
  "phone": "+1987654321",
  "status": "lead",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T14:30:00Z"
}
```

### Toggle Lead Status
**Endpoint:** `PUT /leads/:id/status`
**Auth Required:** JWT + RBAC: Admin only

> **Note:** This endpoint only works on leads that have been converted to clients (status is `"active"` or `"inactive"`). Conversion happens automatically on first receipt. You cannot set a lead back to `"lead"` status.

**Request:**
```json
{
  "status": "inactive"
}
```

| Status Value | Description |
|---|---|
| `active` | Client is active |
| `inactive` | Client is inactive |

**Response (200 OK):**
```json
{
  "id": "60c9g...",
  "tenant_id": "60a7e...",
  "first_name": "Alice",
  "last_name": "Johnson",
  "status": "inactive",
  "converted_at": "2026-03-01T10:00:00Z",
  "updated_at": "2026-03-07T12:00:00Z"
}
```

**Error — Lead not yet converted (400):**
```json
{
  "error": "lead has not been converted to a client yet"
}
```

### List Leads
**Endpoint:** `POST /leads/list`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Request:**
```json
{
  "filters": {},
  "search": "",
  "offset": 0,
  "limit": 10
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `filters` | object | No | Key-value pairs for exact-match filtering (e.g., `country_id`, `qualification_id`) |
| `search` | string | No | Free-text search across lead name, email, and phone. Case-insensitive, partial match supported. |
| `offset` | number | No | Pagination offset (default: 0) |
| `limit` | number | No | Number of results per page (default: 10) |

> **Note:** The `search` field performs a single regex match against a pre-built, lowercase concatenation of `first_name`, `last_name`, `email`, and `phone`. This is optimized for use with a single search box in the UI.

**Search Examples:**

**Search by name (partial match):**
```json
{
  "search": "alice",
  "offset": 0,
  "limit": 10
}
```

**Search combined with filters:**
```json
{
  "filters": {
    "country_id": "60c9g..."
  },
  "search": "john",
  "offset": 0,
  "limit": 10
}
```

**Filter Examples:**

**Filter by country:**
```json
{
  "filters": {
    "country_id": "60c9g..."
  },
  "offset": 0,
  "limit": 10
}
```

**Filter by qualification:**
```json
{
  "filters": {
    "qualification_id": "60d0h..."
  },
  "offset": 0,
  "limit": 10
}
```

**Filter by status:**
```json
{
  "filters": {
    "status": "New"
  },
  "offset": 0,
  "limit": 10
}
```

**Response (200 OK):**

> **Note:** The list response resolves all referenced IDs into nested objects. The `Create`, `Get`, and `Update` endpoints still return raw ObjectIDs since those are used for edit forms.

```json
{
  "data": [
    {
      "id": "60c9g...",
      "tenant_id": "60a7e...",
      "first_name": "Alice",
      "last_name": "Johnson",
      "designation": "Software Engineer",
      "email": "alice@techinnovations.com",
      "phone": "+1987654321",
      "category": {
        "id": "60b8f...",
        "name": "High Priority"
      },
      "source": {
        "id": "60b8f...",
        "name": "Website"
      },
      "assigned_to_user": {
        "id": "60b8f...",
        "name": "John Doe"
      },
      "country": {
        "id": "60c9g...",
        "name": "United Arab Emirates",
        "iso2": "AE",
        "phone_code": "+971"
      },
      "qualification": {
        "id": "60d0h...",
        "name": "Bachelor's Degree"
      },
      "status": "lead",
      "created_at": "2026-02-15T12:00:00Z",
      "updated_at": "2026-02-15T12:00:00Z"
    }
  ],
  "total": 1,
  "offset": 0,
  "limit": 10
}
```

---

## 6. Lead Categories

### Create Lead Category
**Endpoint:** `POST /lead-categories`
**Auth Required:** JWT + RBAC: Admin only

**Request:**
```json
{
  "name": "High Priority",
  "description": "Leads that need immediate follow-up"
}
```

**Response (201 Created):**
```json
{
  "id": "60d0h...",
  "tenant_id": "60a7e...",
  "name": "High Priority",
  "description": "Leads that need immediate follow-up",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T12:00:00Z"
}
```

### Get Lead Category by ID
**Endpoint:** `GET /lead-categories/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Response (200 OK):**
```json
{
  "id": "60d0h...",
  "tenant_id": "60a7e...",
  "name": "High Priority",
  "description": "Leads that need immediate follow-up",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T12:00:00Z"
}
```

### Update Lead Category
**Endpoint:** `PUT /lead-categories/:id`
**Auth Required:** JWT + RBAC: Admin only

**Request:**
```json
{
  "name": "Urgent Priority",
  "description": "Requires contact within 2 hours"
}
```

**Response (200 OK):**
```json
{
  "id": "60d0h...",
  "tenant_id": "60a7e...",
  "name": "Urgent Priority",
  "description": "Requires contact within 2 hours",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T14:30:00Z"
}
```

### Delete Lead Category
**Endpoint:** `DELETE /lead-categories/:id`
**Auth Required:** JWT + RBAC: Admin only

**Response (200 OK):**
```json
{
  "message": "Lead category deleted successfully"
}
```

**Response (400 Bad Request):** (If category is in use by leads)
```json
{
  "error": "Cannot delete category currently in use by leads"
}
```

### List Lead Categories
**Endpoint:** `POST /lead-categories/list`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Request:**
```json
{
  "filters": {},
  "offset": 0,
  "limit": 10
}
```

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "60d0h...",
      "tenant_id": "60a7e...",
      "name": "High Priority",
      "description": "Leads that need immediate follow-up",
      "created_at": "2026-02-15T12:00:00Z",
      "updated_at": "2026-02-15T12:00:00Z"
    }
  ],
  "total": 1,
  "offset": 0,
  "limit": 10
}
```

---

## 7. Lead Comments

### Add Comment to Lead
**Endpoint:** `POST /leads/:lead_id/comments`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Request:**
```json
{
  "content": "Had a great phone screen with Alice. She's ready to sign."
}
```

**Response (201 Created):**
```json
{
  "id": "60e1i...",
  "tenant_id": "60a7e...",
  "lead_id": "60c9g...",
  "author_id": "60b8f...",
  "content": "Had a great phone screen with Alice. She's ready to sign.",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T12:00:00Z"
}
```

### Get Lead Comment by ID
**Endpoint:** `GET /leads/:lead_id/comments/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Response (200 OK):**
```json
{
  "id": "60e1i...",
  "tenant_id": "60a7e...",
  "lead_id": "60c9g...",
  "author_id": "60b8f...",
  "content": "Had a great phone screen with Alice. She's ready to sign.",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T12:00:00Z"
}
```

### Update Lead Comment
**Endpoint:** `PUT /leads/:lead_id/comments/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)
*Note: Users can only update their own comments. Admins can update any comment.*

**Request:**
```json
{
  "content": "Updated details: Alice will sign next week."
}
```

**Response (200 OK):**
```json
{
  "id": "60e1i...",
  "tenant_id": "60a7e...",
  "lead_id": "60c9g...",
  "author_id": "60b8f...",
  "content": "Updated details: Alice will sign next week.",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T14:30:00Z"
}
```

### Delete Lead Comment
**Endpoint:** `DELETE /leads/:lead_id/comments/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)
*Note: Users can only delete their own comments. Admins can delete any comment.*

**Response (200 OK):**
```json
{
  "message": "Comment deleted successfully"
}
```

### List Lead Comments
**Endpoint:** `POST /leads/:lead_id/comments/list`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Request:**
```json
{
  "filters": {},
  "offset": 0,
  "limit": 50
}
```

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "60e1i...",
      "tenant_id": "60a7e...",
      "lead_id": "60c9g...",
      "author": {
        "id": "60b8f...",
        "name": "John Doe"
      },
      "content": "Had a great phone screen with Alice. She's ready to sign.",
      "created_at": "2026-02-15T12:00:00Z",
      "updated_at": "2026-02-15T12:00:00Z"
    }
  ],
  "total": 1,
  "offset": 0,
  "limit": 50
}
```

---

## 8. Lead Appointments

### Schedule Appointment with Lead
**Endpoint:** `POST /leads/:lead_id/appointments`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Request:**
```json
{
  "title": "Initial Demo Call",
  "description": "Walkthrough of core CRM features with Alice",
  "start_time": "2024-05-15T14:30:00Z",
  "end_time": "2024-05-15T15:00:00Z",
  "status": "scheduled"
}
```

**Response (201 Created):**
```json
{
  "id": "60f2j...",
  "tenant_id": "60a7e...",
  "lead_id": "60c9g...",
  "organizer_id": "60b8f...",
  "title": "Initial Demo Call",
  "description": "Walkthrough of core CRM features with Alice",
  "start_time": "2024-05-15T14:30:00Z",
  "end_time": "2024-05-15T15:00:00Z",
  "status": "scheduled",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T12:00:00Z"
}
```

### Get Lead Appointment by ID
**Endpoint:** `GET /leads/:lead_id/appointments/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Response (200 OK):**
```json
{
  "id": "60f2j...",
  "tenant_id": "60a7e...",
  "lead_id": "60c9g...",
  "organizer_id": "60b8f...",
  "title": "Initial Demo Call",
  "description": "Walkthrough of core CRM features with Alice",
  "start_time": "2024-05-15T14:30:00Z",
  "end_time": "2024-05-15T15:00:00Z",
  "status": "scheduled",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T12:00:00Z"
}
```

### Update Lead Appointment
**Endpoint:** `PUT /leads/:lead_id/appointments/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)
*Note: Users can only update appointments they organized. Admins can update any appointment.*

**Request:**
```json
{
  "status": "completed"
}
```

**Response (200 OK):**
```json
{
  "id": "60f2j...",
  "tenant_id": "60a7e...",
  "lead_id": "60c9g...",
  "organizer_id": "60b8f...",
  "title": "Initial Demo Call",
  "description": "Walkthrough of core CRM features with Alice",
  "start_time": "2024-05-15T14:30:00Z",
  "end_time": "2024-05-15T15:00:00Z",
  "status": "completed",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T14:30:00Z"
}
```

### Delete Lead Appointment
**Endpoint:** `DELETE /leads/:lead_id/appointments/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)
*Note: Users can only delete appointments they organized. Admins can delete any appointment.*

**Response (200 OK):**
```json
{
  "message": "Appointment deleted successfully"
}
```

### List Lead Appointments
**Endpoint:** `POST /leads/:lead_id/appointments/list`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Request:**
```json
{
  "filters": {},
  "offset": 0,
  "limit": 50
}
```

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "60f2j...",
      "tenant_id": "60a7e...",
      "lead_id": "60c9g...",
      "organizer": {
        "id": "60b8f...",
        "name": "John Doe"
      },
      "title": "Initial Demo Call",
      "description": "Walkthrough of core CRM features with Alice",
      "start_time": "2024-05-15T14:30:00Z",
      "end_time": "2024-05-15T15:00:00Z",
      "status": "scheduled",
      "created_at": "2026-02-15T12:00:00Z",
      "updated_at": "2026-02-15T12:00:00Z"
    }
  ],
  "total": 1,
  "offset": 0,
  "limit": 50
}
```

---

## 9. Lead Sources

### Create Lead Source
**Endpoint:** `POST /lead-sources`
**Auth Required:** JWT + RBAC: Admin only

**Request:**
```json
{
  "name": "Website",
  "description": "Leads originating from the company website"
}
```

**Response (201 Created):**
```json
{
  "id": "60g3k...",
  "tenant_id": "60a7e...",
  "name": "Website",
  "description": "Leads originating from the company website",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T12:00:00Z"
}
```

### Get Lead Source by ID
**Endpoint:** `GET /lead-sources/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Response (200 OK):**
```json
{
  "id": "60g3k...",
  "tenant_id": "60a7e...",
  "name": "Website",
  "description": "Leads originating from the company website",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T12:00:00Z"
}
```

### Update Lead Source
**Endpoint:** `PUT /lead-sources/:id`
**Auth Required:** JWT + RBAC: Admin only

**Request:**
```json
{
  "description": "Leads originating from organic website traffic"
}
```

**Response (200 OK):**
```json
{
  "id": "60g3k...",
  "tenant_id": "60a7e...",
  "name": "Website",
  "description": "Leads originating from organic website traffic",
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T14:30:00Z"
}
```

### Delete Lead Source
**Endpoint:** `DELETE /lead-sources/:id`
**Auth Required:** JWT + RBAC: Admin only

**Response (200 OK):**
```json
{
  "message": "Lead source deleted successfully"
}
```

### List Lead Sources
**Endpoint:** `POST /lead-sources/list`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Request:**
```json
{
  "filters": {},
  "offset": 0,
  "limit": 10
}
```

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "60g3k...",
      "tenant_id": "60a7e...",
      "name": "Website",
      "description": "Leads originating from the company website",
      "created_at": "2026-02-15T12:00:00Z",
      "updated_at": "2026-02-15T12:00:00Z"
    }
  ],
  "total": 1,
  "offset": 0,
  "limit": 10
}
```

---

## 10. Qualifications

*Global reference data - shared across all tenants. No authentication required.*

### Create Qualification
**Endpoint:** `POST /qualifications`
**Auth Required:** No

**Request:**
```json
{
  "name": "Bachelor's Degree"
}
```

**Response (201 Created):**
```json
{
  "id": "60h4l...",
  "name": "Bachelor's Degree",
  "is_active": true,
  "created_at": "2026-03-06T12:00:00Z",
  "updated_at": "2026-03-06T12:00:00Z",
  "created_by": null,
  "updated_by": null
}
```

### Get Qualification by ID
**Endpoint:** `GET /qualifications/:id`
**Auth Required:** No

**Response (200 OK):**
```json
{
  "id": "60h4l...",
  "name": "Bachelor's Degree",
  "is_active": true,
  "created_at": "2026-03-06T12:00:00Z",
  "updated_at": "2026-03-06T12:00:00Z",
  "created_by": null,
  "updated_by": null
}
```

### Update Qualification
**Endpoint:** `PUT /qualifications/:id`
**Auth Required:** No

**Request:**
```json
{
  "name": "Master's Degree",
  "is_active": true
}
```

**Response (200 OK):**
```json
{
  "id": "60h4l...",
  "name": "Master's Degree",
  "is_active": true,
  "created_at": "2026-03-06T12:00:00Z",
  "updated_at": "2026-03-06T14:30:00Z",
  "created_by": null,
  "updated_by": null
}
```

### Delete Qualification
**Endpoint:** `DELETE /qualifications/:id`
**Auth Required:** No

**Response (200 OK):**
```json
{
  "message": "Qualification deleted successfully"
}
```

### List Qualifications
**Endpoint:** `POST /qualifications/list`
**Auth Required:** No
*Note: Returns only active qualifications by default.*

**Request:**
```json
{
  "filters": {
    "is_active": true
  },
  "offset": 0,
  "limit": 10
}
```

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "60h4l...",
      "name": "High School",
      "is_active": true,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    },
    {
      "id": "60h4m...",
      "name": "Associate's Degree",
      "is_active": true,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    },
    {
      "id": "60h4n...",
      "name": "Bachelor's Degree",
      "is_active": true,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    }
  ],
  "total": 10,
  "offset": 0,
  "limit": 10
}
```

### Filter Examples

**List all qualifications (including inactive):**
```json
{
  "filters": {
    "is_active": false
  },
  "offset": 0,
  "limit": 10
}
```

**List all qualifications (no filter):**
```json
{
  "filters": {},
  "offset": 0,
  "limit": 10
}
```

---

## 11. Countries

*Global reference data - shared across all tenants. No authentication required.*

### Create Country
**Endpoint:** `POST /countries`
**Auth Required:** No

**Request:**
```json
{
  "name": "Japan",
  "iso2": "JP",
  "iso3": "JPN",
  "phone_code": "+81",
  "currency": "JPY",
  "currency_name": "Japanese Yen"
}
```

**Response (201 Created):**
```json
{
  "id": "60i5m...",
  "name": "Japan",
  "iso2": "JP",
  "iso3": "JPN",
  "phone_code": "+81",
  "currency": "JPY",
  "currency_name": "Japanese Yen",
  "is_active": true,
  "created_at": "2026-03-06T12:00:00Z",
  "updated_at": "2026-03-06T12:00:00Z",
  "created_by": null,
  "updated_by": null
}
```

### Get Country by ID
**Endpoint:** `GET /countries/:id`
**Auth Required:** No

**Response (200 OK):**
```json
{
  "id": "60j6n...",
  "name": "United States",
  "iso2": "US",
  "iso3": "USA",
  "phone_code": "+1",
  "currency": "USD",
  "currency_name": "United States Dollar",
  "is_active": true,
  "created_at": "2026-03-06T12:00:00Z",
  "updated_at": "2026-03-06T12:00:00Z"
}
```

### Update Country
**Endpoint:** `PUT /countries/:id`
**Auth Required:** No

**Request:**
```json
{
  "currency": "USD",
  "currency_name": "US Dollar",
  "is_active": false
}
```

**Response (200 OK):**
```json
{
  "id": "60j6n...",
  "name": "United States",
  "iso2": "US",
  "iso3": "USA",
  "phone_code": "+1",
  "currency": "USD",
  "currency_name": "US Dollar",
  "is_active": false,
  "created_at": "2026-03-06T12:00:00Z",
  "updated_at": "2026-03-06T14:30:00Z"
}
```

### Delete Country
**Endpoint:** `DELETE /countries/:id`
**Auth Required:** No

**Response (200 OK):**
```json
{
  "message": "Country deleted successfully"
}
```

### List Countries
**Endpoint:** `POST /countries/list`
**Auth Required:** No
*Note: Returns only active countries by default. Sorted alphabetically by name.*

**Request:**
```json
{
  "filters": {
    "is_active": true
  },
  "offset": 0,
  "limit": 10
}
```

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "60j6n...",
      "name": "Afghanistan",
      "iso2": "AF",
      "iso3": "AFG",
      "phone_code": "+93",
      "currency": "AFN",
      "currency_name": "Afghan Afghani",
      "is_active": true,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    },
    {
      "id": "60j7o...",
      "name": "Albania",
      "iso2": "AL",
      "iso3": "ALB",
      "phone_code": "+355",
      "currency": "ALL",
      "currency_name": "Albanian Lek",
      "is_active": true,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    },
    {
      "id": "60j8p...",
      "name": "United States",
      "iso2": "US",
      "iso3": "USA",
      "phone_code": "+1",
      "currency": "USD",
      "currency_name": "United States Dollar",
      "is_active": true,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    }
  ],
  "total": 140,
  "offset": 0,
  "limit": 10
}
```

### Filter Examples

**Filter by ISO2 code:**
```json
{
  "filters": {
    "iso2": "US"
  },
  "offset": 0,
  "limit": 10
}
```

**Filter by currency:**
```json
{
  "filters": {
    "currency": "USD"
  },
  "offset": 0,
  "limit": 10
}
```

**List all countries (including inactive):**
```json
{
  "filters": {
    "is_active": false
  },
  "offset": 0,
  "limit": 10
}
```

---

## 12. Products

*Tenant-specific products for invoicing.*

### Create Product
**Endpoint:** `POST /products`
**Auth Required:** JWT + RBAC: Admin only

**Request:**
```json
{
  "name": "CRM License - Annual",
  "description": "Annual CRM software license for up to 10 users",
  "price": 1200.00
}
```

**Response (201 Created):**
```json
{
  "id": "60k7q...",
  "tenant_id": "60a7e...",
  "name": "CRM License - Annual",
  "description": "Annual CRM software license for up to 10 users",
  "price": 1200.00,
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T12:00:00Z"
}
```

### Get Product by ID
**Endpoint:** `GET /products/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Response (200 OK):**
```json
{
  "id": "60k7q...",
  "tenant_id": "60a7e...",
  "name": "CRM License - Annual",
  "description": "Annual CRM software license for up to 10 users",
  "price": 1200.00,
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T12:00:00Z"
}
```

### Update Product
**Endpoint:** `PUT /products/:id`
**Auth Required:** JWT + RBAC: Admin only

**Request:**
```json
{
  "price": 999.00,
  "description": "Annual CRM software license for up to 10 users - Special Offer"
}
```

**Response (200 OK):**
```json
{
  "id": "60k7q...",
  "tenant_id": "60a7e...",
  "name": "CRM License - Annual",
  "description": "Annual CRM software license for up to 10 users - Special Offer",
  "price": 999.00,
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T14:30:00Z"
}
```

### Delete Product
**Endpoint:** `DELETE /products/:id`
**Auth Required:** JWT + RBAC: Admin only

**Response (200 OK):**
```json
{
  "message": "Product deleted successfully"
}
```

### List Products
**Endpoint:** `POST /products/list`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Request:**
```json
{
  "filters": {},
  "offset": 0,
  "limit": 10
}
```

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "60k7q...",
      "tenant_id": "60a7e...",
      "name": "CRM License - Annual",
      "description": "Annual CRM software license",
      "price": 999.00,
      "created_at": "2026-03-07T12:00:00Z",
      "updated_at": "2026-03-07T14:30:00Z"
    }
  ],
  "total": 1,
  "offset": 0,
  "limit": 10
}
```

---

## 13. Invoices

*Invoices are linked to leads and contain products with automatic tax calculation based on tenant settings.*

### Invoice Calculation Logic
```
Subtotal = Sum(Product Quantity × Unit Price)
Taxable Amount = Subtotal - Discount
Tax Amount = Taxable Amount × (Tenant Tax Percentage / 100)
Total Amount = Taxable Amount + Tax Amount
```

### Create Invoice
**Endpoint:** `POST /leads/:lead_id/invoices`
**Auth Required:** JWT + RBAC: Admin only

**Request:**
```json
{
  "items": [
    {
      "product_id": "60k7q...",
      "quantity": 2
    },
    {
      "product_id": "60l8r...",
      "quantity": 1
    }
  ],
  "discount": 100.00,
  "due_date": "2026-04-07T23:59:59Z"
}
```

> **Note:** Tax percentage is automatically pulled from tenant settings. Ensure the tenant has a valid `tax` value configured.

**Response (201 Created):**
```json
{
  "id": "60m9s...",
  "tenant_id": "60a7e...",
  "lead_id": "60c9g...",
  "invoice_number": 1,
  "items": [
    {
      "product_id": "60k7q...",
      "product_name": "CRM License - Annual",
      "quantity": 2,
      "unit_price": 999.00,
      "total": 1998.00
    },
    {
      "product_id": "60l8r...",
      "product_name": "Implementation Fee",
      "quantity": 1,
      "unit_price": 500.00,
      "total": 500.00
    }
  ],
  "subtotal": 2498.00,
  "discount": 100.00,
  "tax_percentage": 5.0,
  "tax_amount": 119.90,
  "total_amount": 2517.90,
  "paid_amount": 0,
  "paid_amount_vat": 0,
  "due_date": "2026-04-07T23:59:59Z",
  "status": "pending",
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T12:00:00Z"
}
```

### Get Invoice by ID
**Endpoint:** `GET /invoices/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Response (200 OK):**
```json
{
  "id": "60m9s...",
  "tenant_id": "60a7e...",
  "lead_id": "60c9g...",
  "invoice_number": 1,
  "items": [...],
  "subtotal": 2498.00,
  "discount": 100.00,
  "tax_percentage": 5.0,
  "tax_amount": 119.90,
  "total_amount": 2517.90,
  "paid_amount": 1000.00,
  "paid_amount_vat": 1050.00,
  "due_date": "2026-04-07T23:59:59Z",
  "status": "partial",
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-07T14:00:00Z"
}
```

### Update Invoice
**Endpoint:** `PUT /invoices/:id`
**Auth Required:** JWT + RBAC: Admin only

*Updates invoice items, discount, or due date. Recalculates totals automatically.*

**Request:**
```json
{
  "items": [
    {
      "product_id": "60k7q...",
      "quantity": 3
    },
    {
      "product_id": "60l8r...",
      "quantity": 2
    }
  ],
  "discount": 200.00,
  "due_date": "2026-05-15T23:59:59Z"
}
```

> **Note:** 
> - All fields are optional - only provided fields will be updated
> - Updating items will recalculate subtotal, tax amount, and total amount
> - Cannot update a fully paid invoice

**Response (200 OK):**
```json
{
  "id": "60m9s...",
  "tenant_id": "60a7e...",
  "lead_id": "60c9g...",
  "invoice_number": 1,
  "items": [
    {
      "product_id": "60k7q...",
      "product_name": "CRM License - Annual",
      "quantity": 3,
      "unit_price": 999.00,
      "total": 2997.00
    },
    {
      "product_id": "60l8r...",
      "product_name": "Implementation Fee",
      "quantity": 2,
      "unit_price": 500.00,
      "total": 1000.00
    }
  ],
  "subtotal": 3997.00,
  "discount": 200.00,
  "tax_percentage": 5.0,
  "tax_amount": 189.85,
  "total_amount": 3986.85,
  "paid_amount": 1000.00,
  "paid_amount_vat": 1050.00,
  "due_date": "2026-05-15T23:59:59Z",
  "status": "partial",
  "created_at": "2026-03-07T12:00:00Z",
  "updated_at": "2026-03-08T10:00:00Z"
}
```

**Response (400 Bad Request):** (If invoice is fully paid)
```json
{
  "error": "cannot update a fully paid invoice"
}
```

### Update Invoice Due Date
**Endpoint:** `PUT /invoices/:id/due-date`
**Auth Required:** JWT + RBAC: Admin only

**Request:**
```json
{
  "due_date": "2026-05-07T23:59:59Z"
}
```

**Response (200 OK):**
```json
{
  "id": "60m9s...",
  "tenant_id": "60a7e...",
  "lead_id": "60c9g...",
  "invoice_number": 1,
  "due_date": "2026-05-07T23:59:59Z",
  "status": "partial",
  "updated_at": "2026-03-07T15:00:00Z"
}
```

### List Invoices
**Endpoint:** `POST /invoices/list`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Request:**
```json
{
  "filters": {
    "status": "pending"
  },
  "offset": 0,
  "limit": 10
}
```

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "60m9s...",
      "tenant_id": "60a7e...",
      "lead_id": "60c9g...",
      "invoice_number": 1,
      "subtotal": 2498.00,
      "discount": 100.00,
      "tax_percentage": 5.0,
      "tax_amount": 119.90,
      "total_amount": 2517.90,
      "paid_amount": 1000.00,
      "paid_amount_vat": 1050.00,
      "due_date": "2026-04-07T23:59:59Z",
      "status": "partial",
      "created_at": "2026-03-07T12:00:00Z",
      "updated_at": "2026-03-07T14:00:00Z"
    }
  ],
  "total": 1,
  "offset": 0,
  "limit": 10
}
```

### Get Invoices by Lead ID
**Endpoint:** `GET /leads/:lead_id/invoices`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "60m9s...",
      "tenant_id": "60a7e...",
      "lead_id": "60c9g...",
      "invoice_number": 1,
      "total_amount": 2517.90,
      "paid_amount_vat": 1050.00,
      "status": "partial",
      "created_at": "2026-03-07T12:00:00Z"
    }
  ]
}
```

### Invoice Status Values
| Status | Description |
|--------|-------------|
| `pending` | Invoice created, no payments received |
| `partial` | Partially paid (at least one receipt, not fully paid) |
| `paid` | Fully paid (total amount + tax received) |

---

## 14. Receipts

*Receipts represent payments made against an invoice. Each payment includes tax automatically calculated based on the invoice's tax percentage.*

### Receipt Calculation Logic
```
Tax on Payment = Amount Paid × (Invoice Tax Percentage / 100)
Total Paid = Amount Paid + Tax on Payment
```

### Create Receipt
**Endpoint:** `POST /invoices/:invoice_id/receipts`
**Auth Required:** JWT + RBAC: Admin only

**Guardrails:**
- Payment cannot exceed the remaining amount (including tax)
- Works with multiple partial payments - validates against total paid so far

**Request:**
```json
{
  "amount_paid": 1000.00,
  "payment_date": "2026-03-15T10:00:00Z"
}
```

**Response (201 Created):**
```json
{
  "id": "60n0t...",
  "tenant_id": "60a7e...",
  "invoice_id": "60m9s...",
  "receipt_number": 1,
  "amount_paid": 1000.00,
  "tax_amount": 50.00,
  "total_paid": 1050.00,
  "payment_date": "2026-03-15T10:00:00Z",
  "created_at": "2026-03-15T10:00:00Z"
}
```

### Get Receipt by ID
**Endpoint:** `GET /receipts/:id`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Response (200 OK):**
```json
{
  "id": "60n0t...",
  "tenant_id": "60a7e...",
  "invoice_id": "60m9s...",
  "receipt_number": 1,
  "amount_paid": 1000.00,
  "tax_amount": 50.00,
  "total_paid": 1050.00,
  "payment_date": "2026-03-15T10:00:00Z",
  "created_at": "2026-03-15T10:00:00Z"
}
```

### Update Receipt
**Endpoint:** `PUT /receipts/:id`
**Auth Required:** JWT + RBAC: Admin only

*Updates receipt amount or payment date. Automatically recalculates tax and validates against remaining invoice amount.*

**Request:**
```json
{
  "amount_paid": 1500.00,
  "payment_date": "2026-03-20T14:00:00Z"
}
```

> **Note:** All fields are optional - only provided fields will be updated. Tax is automatically recalculated based on the invoice's tax percentage.

**Response (200 OK):**
```json
{
  "id": "60n0t...",
  "tenant_id": "60a7e...",
  "invoice_id": "60m9s...",
  "receipt_number": 1,
  "amount_paid": 1500.00,
  "tax_amount": 75.00,
  "total_paid": 1575.00,
  "payment_date": "2026-03-20T14:00:00Z",
  "created_at": "2026-03-15T10:00:00Z"
}
```

**Response (400 Bad Request):** (If payment exceeds remaining)
```json
{
  "error": "updated payment exceeds remaining invoice amount"
}
```

### Delete Receipt
**Endpoint:** `DELETE /receipts/:id`
**Auth Required:** JWT + RBAC: Admin only

*Deletes a receipt and automatically recalculates the invoice totals and status.*

**Response (200 OK):**
```json
{
  "message": "Receipt deleted successfully"
}
```

**Note:** Deleting a receipt will:
1. Remove the receipt
2. Recalculate the invoice's `paid_amount` and `paid_amount_vat`
3. Update invoice status (`paid` → `partial` → `pending`) based on remaining payments

### List Receipts by Invoice ID
**Endpoint:** `POST /invoices/:invoice_id/receipts/list`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Request:**
```json
{}
```

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "60n0t...",
      "tenant_id": "60a7e...",
      "invoice_id": "60m9s...",
      "receipt_number": 1,
      "amount_paid": 1000.00,
      "tax_amount": 50.00,
      "total_paid": 1050.00,
      "payment_date": "2026-03-15T10:00:00Z",
      "created_at": "2026-03-15T10:00:00Z"
    },
    {
      "id": "60o1u...",
      "tenant_id": "60a7e...",
      "invoice_id": "60m9s...",
      "receipt_number": 2,
      "amount_paid": 1467.90,
      "tax_amount": 73.40,
      "total_paid": 1541.30,
      "payment_date": "2026-03-20T14:00:00Z",
      "created_at": "2026-03-20T14:00:00Z"
    }
  ]
}
```

### Error: Payment Exceeds Remaining Amount
**Response (400 Bad Request):**
```json
{
  "error": "payment exceeds remaining amount to be paid"
}
```

### Error: Invoice Already Paid
**Response (400 Bad Request):**
```json
{
  "error": "invoice is already fully paid"
}
```

### Receipt Summary Example

For an invoice with:
- Total Amount: 2517.90
- Tax Percentage: 5%
- Status: partial (first payment of 1000 + 50 tax = 1050 received)

**First Receipt:**
```json
{
  "amount_paid": 1000.00,
  "tax_amount": 50.00,
  "total_paid": 1050.00
}
```

**Remaining to pay:** 1467.90 (2517.90 - 1050)

**Second Receipt (full payment):**
```json
{
  "amount_paid": 1467.90,
  "tax_amount": 73.40,
  "total_paid": 1541.30
}
```

Invoice status automatically updates to `paid` after the final receipt.

---

## 15. Tenant Updates

### Address Fields

The Address struct now includes:

| Field | Type | Description |
|-------|------|-------------|
| `street` | string | Street address |
| `address_line` | string | Additional address line (e.g., suite, floor) |
| `city` | string | City |
| `state` | string | State/Province |
| `zip_code` | string | ZIP/Postal code |
| `country` | string | Country name |

### Tenant Fields for Products & Invoices

The following new fields have been added to the Tenant model:

| Field | Type | Description |
|-------|------|-------------|
| `country_id` | ObjectID | Reference to country |
| `tax` | float64 | Tax percentage (e.g., 5.0 for 5%) |
| `next_invoice_number` | int64 | Auto-increment counter for invoices |
| `next_receipt_number` | int64 | Auto-increment counter for receipts |

These fields are used internally for invoice/receipt numbering and tax calculation. Update tenant settings to configure the tax percentage for your organization.

---

## 16. System

### System Health Check
**Endpoint:** `GET /health`
**Auth Required:** No

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

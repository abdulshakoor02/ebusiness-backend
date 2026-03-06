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
    "city": "San Francisco",
    "state": "CA",
    "country": "USA",
    "zip_code": "94105"
  },
  "admin_user": {
    "name": "Jane Doe",
    "email": "jane@acmecorp.com",
    "mobile": "+1234567890",
    "password": "strongpassword123",
    "role": "admin"
  }
}
```

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
    "city": "San Francisco",
    "state": "CA",
    "country": "USA",
    "zip_code": "94105"
  },
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
  }
}
```

**Response (200 OK):**
```json
{
  "id": "60a7e...",
  "name": "Acme Corporation Inc",
  "email": "admin@acmecorp.com",
  "logo_url": "https://example.com/newlogo.png",
  "address": {
    "street": "123 Tech Lane",
    "city": "San Jose",
    "state": "CA",
    "country": "USA",
    "zip_code": "94105"
  },
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
  "created_at": "2026-02-15T12:00:00Z",
  "updated_at": "2026-02-15T14:30:00Z"
}
```

### List Leads
**Endpoint:** `POST /leads/list`
**Auth Required:** JWT + RBAC: All (Admin & User)

**Request:**
```json
{
  "filters": {},
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
(Returns paginated leads)

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
      "author_id": "60b8f...",
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
      "organizer_id": "60b8f...",
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

## 12. System

### System Health Check
**Endpoint:** `GET /health`
**Auth Required:** No

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

# Go CRM Backend API Documentation

Base URL: `http://localhost:3000/api/v1`

All protected endpoints require an Authorization header with a Bearer token:
`Authorization: Bearer <your_jwt_token>`

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
**Auth Required:** Yes (RBAC Enforced)

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
  "created_at": "...",
  "updated_at": "..."
}
```

### Update Tenant
**Endpoint:** `PUT /tenants/:id`
**Auth Required:** Yes (RBAC Enforced)

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

**Response (200 OK):** (Returns the updated Tenant object)

### List Tenants
**Endpoint:** `POST /tenants/list`
**Auth Required:** Yes (RBAC Enforced)

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
**Auth Required:** Yes (RBAC Enforced)

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

**Response (201 Created):** (Returns the created User object, excluding the password hash)

**Response (409 Conflict):**
```json
{
  "error": "Email or mobile already exists"
}
```

### Get User by ID
**Endpoint:** `GET /users/:id`
**Auth Required:** Yes (RBAC Enforced)

**Response (200 OK):** (Returns the User object)

### Update User
**Endpoint:** `PUT /users/:id`
**Auth Required:** Yes (RBAC Enforced)

**Request:** (All fields are optional)
```json
{
  "name": "John Smith Updated",
  "email": "john.updated@acmecorp.com",
  "mobile": "+1555555555",
  "role": "manager"
}
```

**Response (200 OK):** (Returns the updated User object)

### List Users
**Endpoint:** `POST /users/list`
**Auth Required:** Yes (RBAC Enforced)

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
  "path": "/api/v1/users",
  "method": "POST"
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
  "path": "/api/v1/users",
  "method": "POST"
}
```

**Response (200 OK):**
```json
{
  "message": "Permission removed successfully"
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

---

## 5. System

### System Health Check
**Endpoint:** `GET /health`
**Auth Required:** No

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

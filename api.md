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

### Get Available Permission Rules
**Endpoint:** `GET /permissions/available-rules`
**Auth Required:** Yes (RBAC Enforced - Admin only)
*Returns all available permission rules organized by resource with human-readable labels for frontend dropdowns.*

**Response (200 OK):**
```json
{
  "resources": [
    {
      "resource": "tenants",
      "label": "Tenant Management",
      "rules": [
        {
          "id": "60a7e...",
          "resource": "tenants",
          "resource_label": "Tenant Management",
          "action": "view",
          "action_label": "View Tenant Details",
          "path": "/api/v1/tenants/:id",
          "method": "GET",
          "description": "View tenant information",
          "is_system": true,
          "created_at": "2026-02-27T10:00:00Z",
          "updated_at": "2026-02-27T10:00:00Z"
        },
        {
          "id": "60a7f...",
          "resource": "tenants",
          "resource_label": "Tenant Management",
          "action": "update",
          "action_label": "Update Tenant",
          "path": "/api/v1/tenants/:id",
          "method": "PUT",
          "description": "Update tenant information",
          "is_system": true,
          "created_at": "2026-02-27T10:00:00Z",
          "updated_at": "2026-02-27T10:00:00Z"
        }
      ]
    },
    {
      "resource": "users",
      "label": "User Management",
      "rules": [
        {
          "id": "60a8e...",
          "resource": "users",
          "resource_label": "User Management",
          "action": "create",
          "action_label": "Create User",
          "path": "/api/v1/users",
          "method": "POST",
          "description": "Create a new user",
          "is_system": true,
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
*Creates a custom permission rule that can be endpoint-based or frontend-only (empty path/method).*

**Request:**
```json
{
  "resource": "custom-dashboard",
  "resource_label": "Custom Dashboard",
  "action": "access",
  "action_label": "Access Dashboard",
  "path": "",
  "method": "",
  "description": "Access to custom analytics dashboard"
}
```

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
**Auth Required:** Yes (RBAC Enforced)

**Request:**
```json
{
  "assigned_to": "60b8f...",
  "category_id": "60b8f...",
  "source_id": "60b8f...",
  "first_name": "Alice",
  "last_name": "Johnson",
  "company": "Tech Innovations",
  "title": "CTO",
  "email": "alice@techinnovations.com",
  "phone": "+1987654321",
  "status": "New"
}
```

**Response (201 Created):**
(Returns the created Lead object)

### Get Lead by ID
**Endpoint:** `GET /leads/:id`
**Auth Required:** Yes (RBAC Enforced)

**Response (200 OK):**
(Returns the Lead object)

### Update Lead
**Endpoint:** `PUT /leads/:id`
**Auth Required:** Yes (RBAC Enforced)

**Request (Partial update supported):**
```json
{
  "status": "Contacted",
  "source_id": "60b8f...",
  "category_id": "60b8f..."
}
```

**Response (200 OK):**
(Returns the updated Lead object)

### List Leads
**Endpoint:** `POST /leads/list`
**Auth Required:** Yes (RBAC Enforced)

**Request:**
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
**Auth Required:** Yes (RBAC Enforced)

**Request:**
```json
{
  "name": "High Priority",
  "description": "Leads that need immediate follow-up"
}
```

**Response (201 Created):**
(Returns the created LeadCategory object)

### Get Lead Category by ID
**Endpoint:** `GET /lead-categories/:id`
**Auth Required:** Yes (RBAC Enforced)

**Response (200 OK):**
(Returns the LeadCategory object)

### Update Lead Category
**Endpoint:** `PUT /lead-categories/:id`
**Auth Required:** Yes (RBAC Enforced)

**Request:**
```json
{
  "name": "Urgent Priority",
  "description": "Requires contact within 2 hours"
}
```

**Response (200 OK):**
(Returns the updated LeadCategory object)

### Delete Lead Category
**Endpoint:** `DELETE /lead-categories/:id`
**Auth Required:** Yes (RBAC Enforced)

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
**Auth Required:** Yes (RBAC Enforced)

**Request:**
```json
{
  "filters": {},
  "offset": 0,
  "limit": 10
}
```

**Response (200 OK):**
(Returns paginated lead categories)

---

## 7. Lead Comments

### Add Comment to Lead
**Endpoint:** `POST /leads/:lead_id/comments`
**Auth Required:** Yes (RBAC Enforced)

**Request:**
```json
{
  "content": "Had a great phone screen with Alice. She's ready to sign."
}
```

**Response (201 Created):**
(Returns the created LeadComment object containing author_id)

### Get Lead Comment by ID
**Endpoint:** `GET /leads/:lead_id/comments/:id`
**Auth Required:** Yes (RBAC Enforced)

**Response (200 OK):**
(Returns the LeadComment object)

### Update Lead Comment
**Endpoint:** `PUT /leads/:lead_id/comments/:id`
**Auth Required:** Yes (RBAC Enforced - only original author or admin)

**Request:**
```json
{
  "content": "Updated details: Alice will sign next week."
}
```

**Response (200 OK):**
(Returns the updated LeadComment object)

### Delete Lead Comment
**Endpoint:** `DELETE /leads/:lead_id/comments/:id`
**Auth Required:** Yes (RBAC Enforced - only original author or admin)

**Response (200 OK):**
```json
{
  "message": "Comment deleted successfully"
}
```

### List Lead Comments
**Endpoint:** `POST /leads/:lead_id/comments/list`
**Auth Required:** Yes (RBAC Enforced)

**Request:**
```json
{
  "filters": {},
  "offset": 0,
  "limit": 50
}
```

**Response (200 OK):**
(Returns paginated lead comments belonging to specific lead_id)

---

## 8. Lead Appointments

### Schedule Appointment with Lead
**Endpoint:** `POST /leads/:lead_id/appointments`
**Auth Required:** Yes (RBAC Enforced)

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
(Returns the created LeadAppointment object containing organizer_id)

### Get Lead Appointment by ID
**Endpoint:** `GET /leads/:lead_id/appointments/:id`
**Auth Required:** Yes (RBAC Enforced)

**Response (200 OK):**
(Returns the LeadAppointment object)

### Update Lead Appointment
**Endpoint:** `PUT /leads/:lead_id/appointments/:id`
**Auth Required:** Yes (RBAC Enforced)

**Request:**
```json
{
  "status": "completed"
}
```

**Response (200 OK):**
(Returns the updated LeadAppointment object)

### Delete Lead Appointment
**Endpoint:** `DELETE /leads/:lead_id/appointments/:id`
**Auth Required:** Yes (RBAC Enforced)

**Response (200 OK):**
```json
{
  "message": "Appointment deleted successfully"
}
```

### List Lead Appointments
**Endpoint:** `POST /leads/:lead_id/appointments/list`
**Auth Required:** Yes (RBAC Enforced)

**Request:**
```json
{
  "filters": {},
  "offset": 0,
  "limit": 50
}
```

**Response (200 OK):**
(Returns paginated lead appointments belonging to specific lead_id)

---

## 9. Lead Sources

### Create Lead Source
**Endpoint:** `POST /lead-sources`
**Auth Required:** Yes (RBAC Enforced)

**Request:**
```json
{
  "name": "Website",
  "description": "Leads originating from the company website"
}
```

**Response (201 Created):**
(Returns the created LeadSource object)

### Get Lead Source by ID
**Endpoint:** `GET /lead-sources/:id`
**Auth Required:** Yes (RBAC Enforced)

**Response (200 OK):**
(Returns the LeadSource object)

### Update Lead Source
**Endpoint:** `PUT /lead-sources/:id`
**Auth Required:** Yes (RBAC Enforced)

**Request:**
```json
{
  "description": "Leads originating from organic website traffic"
}
```

**Response (200 OK):**
(Returns the updated LeadSource object)

### Delete Lead Source
**Endpoint:** `DELETE /lead-sources/:id`
**Auth Required:** Yes (RBAC Enforced)

**Response (200 OK):**
```json
{
  "message": "Lead source deleted successfully"
}
```

### List Lead Sources
**Endpoint:** `POST /lead-sources/list`
**Auth Required:** Yes (RBAC Enforced)

**Request:**
```json
{
  "filters": {},
  "offset": 0,
  "limit": 10
}
```

**Response (200 OK):**
(Returns paginated lead sources)

---

## 10. System

### System Health Check
**Endpoint:** `GET /health`
**Auth Required:** No

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

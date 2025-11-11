# API Endpoints Documentation

## Base URL

- Local: `http://localhost:8000`
- Swagger UI: `http://localhost:8000/q/swagger-ui`

## Authentication APIs

### 1. Register

- **Endpoint**: `POST /api/v1/auth/register`
- **Authentication**: No (Public)
- **Request Body**:

```json
{
  "email": "user@example.com",
  "password": "password123",
  "full_name": "John Doe",
  "phone_number": "0123456789"
}
```

- **Response**:

```json
{
  "user": {
    "id": "user_id",
    "email": "user@example.com",
    "full_name": "John Doe",
    "phone_number": "0123456789",
    "created_at": "2024-01-01T00:00:00Z"
  },
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 900
}
```

### 2. Login

- **Endpoint**: `POST /api/v1/auth/login`
- **Authentication**: No (Public)
- **Request Body**:

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

- **Response**: Same as Register

### 3. Refresh Token

- **Endpoint**: `POST /api/v1/auth/refresh-token`
- **Authentication**: No (Public)
- **Request Body**:

```json
{
  "refresh_token": "eyJhbGc..."
}
```

- **Response**:

```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 900
}
```

### 4. Get Profile

- **Endpoint**: `GET /api/v1/auth/profile`
- **Authentication**: Required (Bearer Token)
- **Headers**: `Authorization: Bearer {access_token}`
- **Response**:

```json
{
  "id": "user_id",
  "email": "user@example.com",
  "full_name": "John Doe",
  "phone_number": "0123456789",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### 5. Update Profile

- **Endpoint**: `PUT /api/v1/auth/profile`
- **Authentication**: Required (Bearer Token)
- **Request Body**:

```json
{
  "full_name": "Jane Doe",
  "phone_number": "0987654321"
}
```

- **Response**: Same as Get Profile

### 6. Change Password

- **Endpoint**: `POST /api/v1/auth/change-password`
- **Authentication**: Required (Bearer Token)
- **Request Body**:

```json
{
  "old_password": "oldpassword123",
  "new_password": "newpassword456"
}
```

- **Response**:

```json
{
  "message": "Password changed successfully"
}
```

### 7. Logout

- **Endpoint**: `POST /api/v1/auth/logout`
- **Authentication**: Required (Bearer Token)
- **Response**:

```json
{
  "message": "Logged out successfully"
}
```

---

## Job Posting APIs

### 1. Create Job Posting

- **Endpoint**: `POST /api/v1/jobs`
- **Authentication**: Required (Bearer Token)
- **Request Body**:

```json
{
  "company_id": "company_id_here",
  "title": "Senior Backend Engineer",
  "level": "SENIOR",
  "job_type": "FULL_TIME",
  "salary_min": 2000,
  "salary_max": 3500,
  "salary_currency": "USD",
  "location": "Ho Chi Minh City, Vietnam",
  "experience_requirement": "5+ years in backend development",
  "description": "We are looking for a talented backend engineer...",
  "responsibilities": "- Design and develop APIs\n- Write clean code\n- Code review",
  "requirements": "- 5+ years Go experience\n- Strong SQL skills\n- Microservices architecture",
  "benefits": "- Competitive salary\n- Health insurance\n- Remote work",
  "job_tech": ["Go", "PostgreSQL", "Redis", "Docker", "Kubernetes"]
}
```

- **Response**:

```json
{
  "id": "job_id",
  "company_id": "company_id_here",
  "company": {
    "id": "company_id",
    "name": "Tech Company",
    "description": "Leading tech company",
    "website": "https://company.com",
    "logo_url": "https://company.com/logo.png",
    "industry": "Technology",
    "company_size": "51-200",
    "location": "Ho Chi Minh City",
    "founded_year": "2015"
  },
  "title": "Senior Backend Engineer",
  "level": "SENIOR",
  "job_type": "FULL_TIME",
  "salary_min": 2000,
  "salary_max": 3500,
  "salary_currency": "USD",
  "location": "Ho Chi Minh City, Vietnam",
  "posted_at": "2024-01-01T00:00:00Z",
  "experience_requirement": "5+ years in backend development",
  "description": "We are looking for a talented backend engineer...",
  "responsibilities": "- Design and develop APIs\n- Write clean code\n- Code review",
  "requirements": "- 5+ years Go experience\n- Strong SQL skills\n- Microservices architecture",
  "benefits": "- Competitive salary\n- Health insurance\n- Remote work",
  "job_tech": ["Go", "PostgreSQL", "Redis", "Docker", "Kubernetes"],
  "created_at": "2024-01-01T00:00:00Z"
}
```

### 2. Update Job Posting

- **Endpoint**: `PUT /api/v1/jobs/{id}`
- **Authentication**: Required (Bearer Token)
- **Request Body**: Same as Create Job Posting
- **Response**: Same as Create Job Posting

### 3. Delete Job Posting

- **Endpoint**: `DELETE /api/v1/jobs/{id}`
- **Authentication**: Required (Bearer Token)
- **Response**:

```json
{
  "message": "Job posting deleted successfully"
}
```

### 4. Get Job Posting

- **Endpoint**: `GET /api/v1/jobs/{id}`
- **Authentication**: No (Public)
- **Response**: Same as Create Job Posting

### 5. List Job Postings

- **Endpoint**: `GET /api/v1/jobs`
- **Authentication**: No (Public)
- **Query Parameters**:

  - `company_id` (optional): Filter by company ID
  - `location` (optional): Filter by location (partial match)
  - `job_type` (optional): Filter by job type (FULL_TIME, PART_TIME, CONTRACT, INTERNSHIP)
  - `level` (optional): Filter by level (ENTRY, JUNIOR, MID, SENIOR, LEAD)
  - `keyword` (optional): Search in title and description
  - `job_tech` (optional): Filter by technologies (can be multiple, comma-separated)
  - `page` (optional, default: 1): Page number
  - `page_size` (optional, default: 10): Items per page

- **Example**: `GET /api/v1/jobs?location=Ho Chi Minh&level=SENIOR&job_tech=Go,Docker&page=1&page_size=20`

- **Response**:

```json
{
  "jobs": [
    {
      "id": "job_id",
      "company_id": "company_id",
      "company": { ... },
      "title": "Senior Backend Engineer",
      ...
    }
  ],
  "total": 42,
  "page": 1,
  "page_size": 20
}
```

---

## Company APIs

### 1. Create Company

- **Endpoint**: `POST /api/v1/companies`
- **Authentication**: Required (Bearer Token)
- **Request Body**:

```json
{
  "name": "Tech Innovations Inc.",
  "description": "A leading technology company specializing in AI and ML solutions",
  "website": "https://techinnovations.com",
  "logo_url": "https://techinnovations.com/logo.png",
  "industry": "Technology",
  "company_size": "51-200",
  "location": "Ho Chi Minh City, Vietnam",
  "founded_year": "2015"
}
```

- **Response**:

```json
{
  "id": "company_id",
  "name": "Tech Innovations Inc.",
  "description": "A leading technology company specializing in AI and ML solutions",
  "website": "https://techinnovations.com",
  "logo_url": "https://techinnovations.com/logo.png",
  "industry": "Technology",
  "company_size": "51-200",
  "location": "Ho Chi Minh City, Vietnam",
  "founded_year": "2015"
}
```

### 2. Update Company

- **Endpoint**: `PUT /api/v1/companies/{id}`
- **Authentication**: Required (Bearer Token)
- **Request Body**: Same as Create Company
- **Response**: Same as Create Company

### 3. Delete Company

- **Endpoint**: `DELETE /api/v1/companies/{id}`
- **Authentication**: Required (Bearer Token)
- **Response**:

```json
{
  "success": true
}
```

### 4. Get Company

- **Endpoint**: `GET /api/v1/companies/{id}`
- **Authentication**: No (Public)
- **Response**: Same as Create Company

### 5. List Companies

- **Endpoint**: `GET /api/v1/companies`
- **Authentication**: No (Public)
- **Query Parameters**:

  - `industry` (optional): Filter by industry (partial match)
  - `location` (optional): Filter by location (partial match)
  - `keyword` (optional): Search in name and description
  - `page` (optional, default: 1): Page number
  - `page_size` (optional, default: 10): Items per page

- **Example**: `GET /api/v1/companies?industry=Technology&location=Ho Chi Minh&page=1&page_size=20`

- **Response**:

```json
{
  "companies": [
    {
      "id": "company_id",
      "name": "Tech Innovations Inc.",
      ...
    }
  ],
  "total": 15,
  "page": 1,
  "page_size": 20
}
```

---

## Enums

### Job Type

- `FULL_TIME`
- `PART_TIME`
- `CONTRACT`
- `INTERNSHIP`

### Level

- `ENTRY` - Entry level
- `JUNIOR` - Junior (1-2 years)
- `MID` - Mid level (3-5 years)
- `SENIOR` - Senior (5+ years)
- `LEAD` - Lead/Principal

### Company Size

- `1-10`
- `11-50`
- `51-200`
- `201-500`
- `501-1000`
- `1001+`

---

## Authentication

### Public Endpoints (No Token Required)

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh-token`
- `GET /api/v1/jobs` (List)
- `GET /api/v1/jobs/{id}` (Get)
- `GET /api/v1/companies` (List)
- `GET /api/v1/companies/{id}` (Get)

### Protected Endpoints (Token Required)

All other endpoints require Bearer token authentication.

**Header Format**:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

---

## Error Responses

### 400 Bad Request

```json
{
  "code": 400,
  "message": "Invalid request parameters",
  "reason": "INVALID_ARGUMENT"
}
```

### 401 Unauthorized

```json
{
  "code": 401,
  "message": "Unauthorized",
  "reason": "UNAUTHENTICATED"
}
```

### 404 Not Found

```json
{
  "code": 404,
  "message": "Resource not found",
  "reason": "NOT_FOUND"
}
```

### 409 Conflict

```json
{
  "code": 409,
  "message": "Email already exists",
  "reason": "ALREADY_EXISTS"
}
```

### 500 Internal Server Error

```json
{
  "code": 500,
  "message": "Internal server error",
  "reason": "INTERNAL"
}
```

---

## Testing with cURL

### Register

```bash
curl -X POST http://localhost:8000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "full_name": "Test User",
    "phone_number": "0123456789"
  }'
```

### Login

```bash
curl -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

### Get Profile (with token)

```bash
curl -X GET http://localhost:8000/api/v1/auth/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### List Jobs

```bash
curl -X GET "http://localhost:8000/api/v1/jobs?location=Ho%20Chi%20Minh&level=SENIOR&page=1&page_size=10"
```

### Create Company (with token)

```bash
curl -X POST http://localhost:8000/api/v1/companies \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Company",
    "description": "Great company",
    "industry": "Technology",
    "location": "Ho Chi Minh City"
  }'
```

### Create Job Posting (with token)

```bash
curl -X POST http://localhost:8000/api/v1/jobs \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": "COMPANY_ID_HERE",
    "title": "Backend Engineer",
    "level": "MID",
    "job_type": "FULL_TIME",
    "salary_min": 1500,
    "salary_max": 2500,
    "salary_currency": "USD",
    "location": "Remote",
    "description": "Looking for backend engineer",
    "job_tech": ["Go", "MongoDB"]
  }'
```

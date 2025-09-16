# Chirpy API

A Twitter-like social media API built with Go that allows users to create short messages called "chirps", manage user accounts, and handle authentication.

## Features

- **User Management**: Create accounts, login, and update user information
- **Chirp Management**: Create, read, and delete short messages (max 140 characters)
- **Authentication**: JWT-based authentication with refresh tokens
- **Premium Features**: Upgrade users to "Chirpy Red" status via webhooks
- **Sorting & Filtering**: Get chirps by author and sort by creation date

## Tech Stack

- **Language**: Go
- **Database**: PostgreSQL (local setup needed no Docker used!)
- **Authentication**: JWT tokens with refresh token rotation
- **Password Security**: Bcrypt hashing
- **Environment Config**: godotenv for environment variables

## Prerequisites

- Go 1.24+
- PostgreSQL database
- Environment variables configured (see Configuration section)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/Cheemx/chirpy.git
cd chirpy
```

2. Install dependencies:
```bash
go mod tidy
```

3. Set up your environment variables (see Configuration section)

4. Run the application:
```bash
go run .
```

The server will start on port 8080.

## Configuration

Create a `.env` file in the root directory with the following variables:

```env
DB_URL=postgres://username:password@localhost/chirpy?sslmode=disable
SECRET=your-jwt-secret-key
POLKA_KEY=your-polka-webhook-api-key
```

## API Endpoints

### Health Check
- `GET /api/healthz` - Health check endpoint

### User Management
- `POST /api/users` - Create a new user
- `PUT /api/users` - Update user information (requires authentication)
- `POST /api/login` - User login
- `POST /api/refresh` - Refresh access token
- `POST /api/revoke` - Revoke refresh token

### Chirp Management
- `POST /api/chirps` - Create a new chirp (requires authentication)
- `GET /api/chirps` - Get all chirps (supports sorting and filtering)
- `GET /api/chirps/{chirpID}` - Get a specific chirp by ID
- `DELETE /api/chirps/{chirpID}` - Delete a chirp (requires authentication and ownership)

### Premium Features
- `POST /api/polka/webhooks` - Webhook endpoint for upgrading users to Chirpy Red

### Admin
- `GET /admin/metrics` - View server metrics
- `POST /admin/reset` - Reset server metrics

## API Usage Examples

### Create a User
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "securepassword"}'
```

### Login
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "securepassword"}'
```

### Create a Chirp
```bash
curl -X POST http://localhost:8080/api/chirps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{"body": "This is my first chirp!"}'
```

### Get Chirps
```bash
# Get all chirps
curl http://localhost:8080/api/chirps

# Get chirps sorted by newest first
curl http://localhost:8080/api/chirps?sort=desc

# Get chirps by specific author
curl http://localhost:8080/api/chirps?author_id=USER_UUID
```

## Authentication

The API uses JWT tokens for authentication:

1. **Access Tokens**: Short-lived tokens (1 hour) for API access
2. **Refresh Tokens**: Long-lived tokens (60 days) for obtaining new access tokens

Include the access token in the Authorization header:
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

## Query Parameters

### GET /api/chirps
- `sort=desc` - Sort chirps by creation date (newest first)
- `author_id=UUID` - Filter chirps by author ID

## Response Formats

### User Response
```json
{
  "id": "uuid",
  "created_at": "timestamp",
  "updated_at": "timestamp", 
  "email": "user@example.com",
  "is_chirpy_red": false
}
```

### Chirp Response
```json
{
  "id": "uuid",
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "body": "Chirp content",
  "user_id": "uuid"
}
```

### Login Response
```json
{
  "id": "uuid",
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "email": "user@example.com",
  "token": "jwt_access_token",
  "refresh_token": "refresh_token",
  "is_chirpy_red": false
}
```

## Error Handling

The API returns appropriate HTTP status codes:
- `200` - Success
- `201` - Created
- `204` - No Content
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `500` - Internal Server Error

## Database Schema

The application expects the following database tables:
- `users` - User account information
- `chirps` - Chirp messages
- `refresh_tokens` - Refresh token storage

Thanks and Star the repository if you found it fascinating ⭐
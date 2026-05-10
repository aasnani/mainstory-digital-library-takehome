# API contract (frontend)

Browser-facing behavior only: URLs, JSON shapes, auth, errors, and status codes. **Update this file whenever routes, bodies, or HTTP semantics change.**

## Base URL

Use one configurable **API base URL** (for example `VITE_API_BASE_URL`). Every path below is appended to that origin.

- JSON APIs live under **`/api/v1`**.
- Example: base `https://api.example.com` → login at `POST https://api.example.com/api/v1/auth/login`.

Send **`Content-Type: application/json`** on requests with a body.

Use **HTTPS** in production; passwords are sent in JSON over TLS like a typical web app.

## Authentication

### Bearer token

Protected routes:

```http
Authorization: Bearer <access_token>
```

Use exactly one space after `Bearer`.

### Login and register

| Action | Method | Path | Body |
|--------|--------|------|------|
| Register | `POST` | `/api/v1/auth/register` | `{ "email": string, "password": string }` |
| Login | `POST` | `/api/v1/auth/login` | `{ "email": string, "password": string }` |

- Password **length**: **8–72** characters (enforced on register; use the same limits in the UI).
- Passwords are **never** returned in any response.

**Success**

| | HTTP status |
|--|-------------|
| Register | **201** |
| Login | **200** |

**Response body** (same shape for both):

```json
{
  "access_token": "<jwt>",
  "token_type": "Bearer",
  "expires_in": 86400,
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "role": "MEMBER"
  }
}
```

- **`expires_in`**: seconds until the access token expires (use for UX or refresh hints).
- **`user.role`**: `MEMBER`, `LIBRARIAN`, or `ADMIN`. Registration always returns **`MEMBER`**.

### JWT (optional client decode)

Tokens are standard JWTs. Useful claims if you decode for UI only (navigation, labels):

| Claim | Meaning |
|-------|---------|
| `sub` | User id (UUID string). |
| `role` | `MEMBER`, `LIBRARIAN`, or `ADMIN`. |
| `exp` | Expiry (Unix seconds). |
| `iat` | Issued at (Unix seconds). |

Treat **`403`** / missing data as the source of truth for permissions; do not rely on the token alone for security.

### Storing the token (browser)

The API does not use cookies for this MVP; attach **`Authorization`** yourself.

- **`sessionStorage`**: cleared when the tab closes.
- **`localStorage`**: survives refresh; clear on **401**.
- **In-memory only**: harder for XSS to read from storage; lost on full reload.

On **401**, clear the stored token and return the user to login/register.

## Cross-origin requests

If the SPA origin differs from the API origin, the server must allow it via CORS. This API expects **Bearer** tokens in headers (not cookie-based auth for MVP).

## Roles

- New accounts are **`MEMBER`** only.
- There is **no** public endpoint that creates an **`ADMIN`** user; admin screens assume an account that already has that role in the backend.
- You **cannot** change your own **`email`** or **`role`** through the API (including **`ADMIN`** on **`PATCH`** targeting yourself). **`ADMIN`** can change **`email`** / **`role`** only when **`PATCH`** targets **another** user’s id.

## Error format

Errors use this JSON shape:

```json
{
  "error": {
    "code": "validation_error",
    "message": "human-readable detail"
  }
}
```

## HTTP status codes (client handling)

| Code | Typical cause |
|------|----------------|
| **400** | Bad JSON, invalid path/query params, validation (e.g. password length). Empty **`PATCH`** body for self. **`current_password`** without **`new_password`** (or vice versa). Password fields on **`PATCH`** for someone else’s id. |
| **401** | Missing or bad `Authorization`, wrong login credentials, wrong **`current_password`** when changing password, expired token. |
| **403** | Logged in but not allowed (e.g. non-admin listing users). **`PATCH`** on yourself with **`email`** or **`role`**. |
| **404** | Resource not found (e.g. unknown user id). |
| **409** | Conflict (e.g. email already registered, email taken on update, cannot delete user). |
| **500** | Server error. |

## Endpoints

### Health (optional connectivity check)

| Method | Path | Auth | Success |
|--------|------|------|---------|
| `GET` | `/healthcheck` | none | **200**, body text `UP` |

### Auth

| Method | Path | Auth | Body | Success |
|--------|------|------|------|---------|
| `POST` | `/api/v1/auth/register` | none | `{ "email", "password" }` | **201** + token payload |
| `POST` | `/api/v1/auth/login` | none | `{ "email", "password" }` | **200** + token payload |

### Users

**Password change (your own account only)** — **`PATCH /api/v1/users/me`** or **`PATCH /api/v1/users/:id`** when **`id`** is your user id:

- Body: **`{ "current_password": string, "new_password": string }`** (both required together). **`new_password`** must satisfy **8–72** characters.
- Wrong **`current_password`** → **401**. Omitting one of the two password fields → **400**.

**Admin updating another user** (`ADMIN` only, **`id`** not your own):

- Body: **`{ "email"?: string, "role"?: string }`** — at least one field is typical; omitted fields stay unchanged.
- Do **not** send **`current_password`** / **`new_password`** here (**400**); password change is only for self.

| Method | Path | Auth | Body | Success |
|--------|------|------|------|---------|
| `GET` | `/api/v1/users/me` | Bearer | — | **200** user |
| `PATCH` | `/api/v1/users/me` | Bearer | Self: **`current_password`** + **`new_password`** only (see above). No **`email`** / **`role`**. | **200** user |
| `GET` | `/api/v1/users` | Bearer (**ADMIN**) | Query: `limit` (1–100, default 50), `offset` (≥0, default 0) | **200** `{ "users": User[] }` |
| `GET` | `/api/v1/users/:id` | Bearer (**ADMIN** or **self**) | — | **200** user |
| `PATCH` | `/api/v1/users/:id` | Bearer (**ADMIN** or **self**) | **Self**: same as **`PATCH /users/me`** (password only). **Admin** patching **another** user: **`email`** / **`role`** only. | **200** user |
| `DELETE` | `/api/v1/users/:id` | Bearer (**ADMIN**) | — | **204** empty body |

**User object**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "role": "MEMBER"
}
```

## Integration checklist

1. Read **`API base URL`** from build/runtime config and prefix all paths.
2. After login/register, save **`access_token`** and send **`Authorization: Bearer`** on **`/api/v1/***`** except **`/auth/*`**.
3. On **401**, clear the token and show auth UI.
4. Use **`GET /users/me`** as the source of current user identity.
5. Use **`role`** from **`/users/me`** (or JWT) only for UI; handle **403** from the API when actions are not allowed.
6. To change password, **`PATCH`** your profile with **`current_password`** and **`new_password`**; then use the new password on the next login (existing JWTs stay valid until expiry).

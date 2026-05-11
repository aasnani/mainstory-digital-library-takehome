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

### Public catalog (optional auth)

**`GET /api/v1/books`**, **`GET /api/v1/books/recent`**, and **`GET /api/v1/books/:id`** work **without** any **`Authorization`** header so anonymous visitors can browse search results, a small “new arrivals” strip, and book detail pages (metadata and pricing only; **no** full **`content`** until they are logged in and entitled).

- **No header**: treated as a **guest** — every book shows **`is_accessible`: false** and **`access_reason`**: **`LOCKED`** on list; detail has **no** **`content`**.
- **Valid `Authorization: Bearer …`**: same rules as a signed-in **MEMBER** / **LIBRARIAN** / **ADMIN** (subscription, purchases, staff preview). Use this on the book detail page after login so entitled users can read **`content`**.
- **Invalid or expired token** if a header is sent: **401** — clear the token and retry as guest or re-login.

Flows like “buy” / **`POST /entitlements`** still require registration and a Bearer token.

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

### Books and entitlements (summary)

| Role | Books | Entitlements |
|------|--------|----------------|
| **MEMBER** | **Read** only (`GET` list + detail). Detail **`content`** only if entitled (**subscription** or **purchase**). List/detail include **`is_accessible`** and **`access_reason`**. | **Create** (mock purchase/subscribe), **read** **own** rows, **cancel own subscription** (**`POST /users/me/subscription/cancel`**). |
| **LIBRARIAN** | **Create**, **read**, **update** (no **delete**). Always sees full **content** on detail. | **Read all** (list + get). **No** create / update / delete. |
| **ADMIN** | Full **CRUD** | **Create**, **read**, **update** (no **delete** endpoint). |

Access to **book text**: active **`SUBSCRIPTION`** (all books) or **`SINGLE_PURCHASE`** with matching **`book_id`**. Otherwise **`access_reason`** is **`LOCKED`** and **`content`** is omitted on detail for members.

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
| **400** | Bad JSON, invalid path/query params, validation (e.g. password length). Empty **`PATCH`** body for self. **`current_password`** without **`new_password`** (or vice versa). Password fields on **`PATCH`** for someone else’s id. Book list search: **`q`**, **`title`**, or **`author`** shorter than **2** characters. Invalid **`min_price_cents`** / **`max_price_cents`** range. |
| **401** | Missing or bad `Authorization`, wrong login credentials, wrong **`current_password`** when changing password, expired token. |
| **403** | Logged in but not allowed (e.g. non-admin listing users). **`PATCH`** on yourself with **`email`** or **`role`**. Creating/updating books or entitlements without permission. |
| **404** | Resource not found (e.g. unknown user id, unknown book when purchasing). **`POST /users/me/subscription/cancel`** when there is no subscription in the current paid window (**`ends_at`** in the future, **`status`** **`ACTIVE`**) — **`no_active_subscription`**. |
| **409** | Conflict (e.g. email already registered, duplicate entitlement / unique constraint, cannot delete book referenced by entitlements). |
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
| `GET` | `/api/v1/users/me/library` | Bearer | — | **200** “my purchases” payload (see below). |
| `POST` | `/api/v1/users/me/subscription/cancel` | Bearer | — | **200** entitlement: **`status`** stays **`ACTIVE`** until **`ends_at`**; **`cancelled_at`** set (non-null) when a cancel is first requested. **404** **`no_active_subscription`** if there is no current paid period (expired or never subscribed). Second call is idempotent (same row). |
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

### My library (single call)

**`GET /api/v1/users/me/library`** returns everything needed for a “my purchases / subscription” page in one response:

- **`subscription`**: subscription entitlement for the current **paid access window** (**`status`**: **`ACTIVE`**, **`ends_at`** in the future), or omitted / **`null`** if none. A user-cancelled subscription still appears here until **`ends_at`** (check **`cancelled_at`** for “won’t renew / ended at period end”).
- **`purchases`**: array of **`{ "entitlement", "book" }`** for each **active** per-book purchase. **`book`** is catalog metadata only (**no** **`content`**), with **`is_accessible`: true** and **`access_reason`**: **`PURCHASED`**.

Browsing the full catalog stays on **`GET /books`** (paginated + filterable). This endpoint does not load heavy **`content`** fields.

### Books

Catalog fields include **`title`**, **`description`**, **`author`**, **`genre`**, **`is_fiction`**, **`published_date`**, **`added_at`**, **`language`**, **`price_cents`**. **`GET /books`** and **`GET /books/:id`** for **MEMBER** when locked omit **`content`** as described above; the list endpoint **never** reads **`content`** from the database (safe for large blobs).

**`GET /api/v1/books`** — query parameters (all optional unless noted; combine with **AND**):

| Param | Meaning |
|-------|---------|
| **`limit`** | 1–100, default **50** |
| **`offset`** | ≥ 0, default **0** |
| **`q`** | Substring match on **title**, **author**, or **genre** (case-insensitive). **≥ 2** characters if present (use for typeahead: debounce client-side so you only call once the user typed enough). |
| **`title`**, **`author`** | Same minimum length as **`q`** when non-empty. |
| **`genre`**, **`language`** | Filter (genre substring; language case-insensitive exact). No minimum length. |
| **`is_fiction`** | **`true`** / **`false`** |
| **`min_price_cents`**, **`max_price_cents`** | Inclusive bounds; **`min`** must not exceed **`max`**. |

| Method | Path | Auth | Notes |
|--------|------|------|--------|
| `GET` | `/api/v1/books` | **Optional** Bearer | **Guest**: no header. Response **`{ "books": BookListItem[] }`**. No **`content`**. **`is_accessible`** / **`access_reason`** reflect entitlements when JWT present; else all **LOCKED**. |
| `GET` | `/api/v1/books/recent` | **Optional** Bearer | Same response shape as **`GET /books`**: **`{ "books": BookListItem[] }`**, at most **five** rows, **`added_at`** descending (newest catalog additions first). No query params. No **`content`**. Same **`is_accessible`** / **`access_reason`** rules as the full list. |
| `GET` | `/api/v1/books/:id` | **Optional** Bearer | **Guest**: catalog only, **`content`** omitted. **Logged-in**: same entitlement rules as before; staff always see **`content`**. |
| `POST` | `/api/v1/books` | Bearer **LIBRARIAN** or **ADMIN** | Create catalog row (JSON body includes **`title`**, **`price_cents`**, optional metadata and **`content`**). **201**. |
| `PATCH` | `/api/v1/books/:id` | Bearer **LIBRARIAN** or **ADMIN** | Full replacement-style payload (same shape as create). **200**. |
| `DELETE` | `/api/v1/books/:id` | Bearer **ADMIN** only | **204** if deleted; **409** if entitlements still reference the book. |

### Entitlements

Types: **`SINGLE_PURCHASE`** (requires **`book_id`**) or **`SUBSCRIPTION`** (**omit** **`book_id`**). Status values include **`ACTIVE`**, **`CANCELLED`**, **`PAST_DUE`**.

**Subscription period (MVP)**: **`POST`** with **`{ "type": "SUBSCRIPTION" }`** sets **`renewed_at`** to “now” (UTC) and **`ends_at`** to **`renewed_at` + 30 days**. Access is active while **`status`** is **`ACTIVE`** and **`ends_at`** is in the future. After **`ends_at`**, the row is moved to **`CANCELLED`** by the server when entitlements are read or created (no separate renewal endpoint yet; a future renewal would bump **`renewed_at`** and **`ends_at`** from that moment).

**MEMBER** **`POST`**: body **`{ "type": "SUBSCRIPTION" }`** or **`{ "type": "SINGLE_PURCHASE", "book_id": "<uuid>" }`**. Do **not** send **`user_id`** (always yourself). Idempotent conflicts → **409**.

**MEMBER** **cancel subscription**: **`POST /api/v1/users/me/subscription/cancel`** (no body). Sets **`cancelled_at`** but **keeps** **`status`**: **`ACTIVE`** until **`ends_at`**, so the member keeps catalog access for the rest of the period. Does **not** affect per-book purchases. Staff **`PATCH /entitlements/:id`** remains **ADMIN** only (can still force **`status`** / **`ends_at`** immediately if needed).

**ADMIN** **`POST`**: include **`user_id`** for whose entitlement to create (required).

**ADMIN** **`PATCH /entitlements/:id`**: optional **`status`**, **`ends_at`** (RFC3339).

| Method | Path | Auth | Body | Success |
|--------|------|------|------|---------|
| `GET` | `/api/v1/entitlements` | Bearer | Query **`limit`**, **`offset`**. **MEMBER**: own rows only. **LIBRARIAN** / **ADMIN**: all. | **200** `{ "entitlements": [...] }` |
| `GET` | `/api/v1/entitlements/:id` | Bearer | **MEMBER**: only if **`user_id`** is you. Staff: any. | **200** entitlement |
| `POST` | `/api/v1/entitlements` | Bearer (**MEMBER** or **ADMIN**, not **LIBRARIAN**) | See above | **201** |
| `PATCH` | `/api/v1/entitlements/:id` | Bearer **ADMIN** | **`status`**, **`ends_at`** optional | **200** |

**Entitlement object** (fields include **`id`**, **`user_id`**, **`book_id`**, **`type`**, **`status`**, **`ends_at`**, **`renewed_at`**, **`cancelled_at`**, **`created_at`**).

## Integration checklist

1. Read **`API base URL`** from build/runtime config and prefix all paths.
2. **Catalog** (**`GET /books`**, **`GET /books/recent`**, **`GET /books/:id`**) can be called **without** a token for the marketing/browse experience. After login/register, send **`Authorization: Bearer`** on protected routes (**`/users/*`**, **`/entitlements`**, mutations, **`/users/me/library`**, **`/users/me/subscription/cancel`**). You may attach the same Bearer on catalog **GET**s so entitled users see **`content`** and correct **`is_accessible`** flags.
3. On **401**, clear the token and show auth UI.
4. Use **`GET /users/me`** as the source of current user identity.
5. Use **`role`** from **`/users/me`** (or JWT) only for UI; handle **403** from the API when actions are not allowed.
6. To change password, **`PATCH`** your profile with **`current_password`** and **`new_password`**; then use the new password on the next login (existing JWTs stay valid until expiry).
7. Main catalog: **`GET /books`** with **`limit`** / **`offset`** and optional filters; debounce search inputs and only send **`q`** / **`title`** / **`author`** when length ≥ **2**. Use **`is_accessible`** / **`access_reason`** for locked UI. For a home-page “new” strip, **`GET /books/recent`** (no params, max five books by **`added_at`**).
8. “My library” page: **`GET /users/me/library`** once; no book **`content`** in that payload. Purchase or subscribe via **`POST /entitlements`** before reading **`GET /books/:id`** content as a **MEMBER**. To **stop renewing** at period end, **`POST /users/me/subscription/cancel`** (access continues until **`ends_at`**; UI can read **`cancelled_at`**). **404** **`no_active_subscription`** if there is no current period.

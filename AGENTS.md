# Mainstory Digital Library (Go) — agent context

This file is **tracked** in git so reviewers and collaborators share the same product rules and repo map.

Keep it **non-speculative**: only document directories, packages, and conventions that actually exist in the repository at the time.

This repo is a takehome MVP for a paid digital storybook library with two monetization models:

- **Subscription**: unlimited access
- **One-time purchase**: access to specific books only

This file exists to keep implementation decisions **clear, consistent, and reviewable** as the codebase grows.

> Note: when present in the working tree, this file is **binding** guidance for agents working in this repo.

---

## Repo map (quick navigation)

This section is intentionally **non-speculative**.

- Only list files/folders that **exist in the repo right now**.
- When new top-level directories are created (e.g. `cmd/`, `internal/`, `web/`), update this map in the same change series.
- **Every prompt**: update **`docs/submission.md`** with any new product/system/AI-usage decisions made during the prompt so the final submission write-up stays current.

- **API documentation**
  - **`docs/api-contract.md`** (tracked): **frontend-only** contract—base URL, endpoints, JSON, auth, errors, CORS for SPAs; no backend env or operator runbooks (those live in **README**). **Update in the same change series as any HTTP/API behavior change** (no code-first drift).

- **Entry points**
  - `README.md`: high-level overview, env vars, links to API contract.
  - `docs/submission.md`: product + system design write-up + prompt-by-commit log.
  - `docs/api-contract.md`: SPA/Lovable integration reference.
  - `AGENTS.md`: this file (repo map + conventions + agent memory log).
  - `.cursor/rules/agents-md.mdc`: Cursor rule for agent upkeep (tracked in git).

- **Application (`internal/`)**
  - `internal/config/`: environment configuration.
  - `internal/db/`: Postgres pool wiring.
  - `internal/domain/`: domain types and validation helpers (users, books, entitlements).
  - `internal/repository/`: data access (`users`, `books`, `entitlements`).
  - `internal/service/`: business logic (users, books with entitlement checks, entitlements).
  - `internal/handlers/`: HTTP handlers (Gin): auth, users, books, entitlements.
  - `internal/middleware/`: auth and CORS.
  - `internal/auth/`: JWT sign/parse; bcrypt password hashing/compare.
  - `internal/api/`: shared JSON error helpers.

- **Database**
  - `db/migration/`: Flyway versioned SQL migrations (`V1__initial_schema.sql`, `V2__users_password_hash.sql`, `V3__subscription_period_renewal_cancel_at.sql`).

Update this map whenever you add/rename top-level structure or domain boundaries.

---

## Product rules (must-haves)

### Access control (entitlements)

User can access a book iff **either**:

- user has an **active subscription**, OR
- user has purchased that **specific book**

Access checks must exist in **one** central place (policy/guard/service), and be used by:

- **Book content endpoints** (book details, read/download)
- **Book listing** (should clearly indicate locked vs accessible, and why)
- **CRUD endpoints** (if you support admin vs user roles, enforce it here)

### Subscription model (MVP assumptions)

- Subscription can be **active** or **inactive** (expired/cancelled).
- Only one subscription per user is “current” (even if you store history).
- Mock payment should be deterministic and testable (no random outcomes).

### One-time purchases (MVP assumptions)

- Purchase grants durable access to that book.
- Multiple purchases of the same book should be idempotent (return “already owned”).

### UX requirements surfaced through API

- List endpoints should include per-book fields like `isAccessible` and `accessReason` (e.g. `SUBSCRIPTION`, `PURCHASED`, `LOCKED`) so the frontend can render “locked vs unlocked” clearly.

---

## System design conventions (engineering)

### Architecture

- Keep **controllers thin** (parse/validate/request/response).
- Put business logic in **services** (subscription, purchase, entitlement evaluation).
- Put permission checks in **policies/guards** and keep them **pure** where possible.
- Model database access behind a small **repository/data-access** layer.

### Data model (non-speculative)

At minimum the backend should represent:

- `users`
- `books`
- `entitlements` (expresses access via subscription and/or book purchase)

Do not assume key shapes, field names, or specific schema until the database layer is implemented.

Prefer constraints that make idempotency easy (e.g. “purchase this book” is safe to retry), but document the actual approach once the DB is chosen.

### Validation and errors

- Validate all inputs (body/query/params) and return consistent error shapes.
- Prefer explicit HTTP codes:
  - `400` validation
  - `401` unauthenticated
  - `403` authenticated but not entitled/authorized
  - `404` not found (avoid leaking existence when it matters; document trade-off)
  - `409` conflicts (e.g. duplicate purchase if not idempotent)

### Auth (mocked is fine)

- Use a simple approach (e.g. “login returns token”, token maps to user).
- Never trust client-provided `userId` directly; derive from auth context.

---

## Go conventions (engineering)

- **Formatting**: `gofmt` always.
- **Imports**: standard library, then third-party, then module-local.
- **Errors**: wrap with context (e.g. `%w`) at boundaries; return typed/domain errors where it helps HTTP mapping.
- **Testing**: favor table-driven tests for access-control and entitlement evaluation.
- **Package design**: avoid cyclic imports; keep domain logic in `internal/` once it exists.

---

## AI usage conventions (for the takehome)

The submission requires describing AI usage. Capture this in **`docs/submission.md`**:

- prompts you used
- iterations and what improved
- any guardrails / checks you applied

Do **not** add “AI-generated” attribution to commit messages or PR text.

---

## Agent memory log (append-only)

### Format (one line each)

- **YYYY-MM-DD** `(<scope>) <what changed> — <why it matters> [files: ...]`

### Log

- **2026-05-08** `(docs) Add AGENTS.md and Cursor rule scaffold — establishes repo map, product rules, and upkeep expectations for takehome MVP [files: AGENTS.md, .cursor/rules/agents-md.mdc]`
- **2026-05-08** `(docs) Make AGENTS.md Go-specific and non-speculative — removes assumed structure/schema; map updates only when directories actually exist [files: AGENTS.md, .cursor/rules/agents-md.mdc]`
- **2026-05-08** `(backend) Add Gin healthcheck endpoint — minimal HTTP service with GET /healthcheck returning UP [files: main.go, go.mod, go.sum, README.md]`
- **2026-05-08** `(ci) Create build and test workflow on PR — runs go build and go test on pull requests targeting main [files: .github/workflows/go.yml]`
- **2026-05-10** `(db) Flyway V1 initial schema — users (role MEMBER default + CHECK MEMBER/LIBRARIAN/ADMIN), books (title, price_cents, content TEXT), entitlements; subscription/purchase access model [files: db/migration/V1__initial_schema.sql, README.md]`
- **2026-05-10** `(db) Books catalog metadata + entitlement invariants — genre/fiction/published/added/synopsis/reading_level/language/cover_url; CHECK type/status + type vs book_id; partial unique purchase + single active subscription [files: db/migration/V1__initial_schema.sql]`
- **2026-05-10** `(db) Remove books synopsis, reading_level, cover_image_url — slimmer catalog schema [files: db/migration/V1__initial_schema.sql]`
- **2026-05-10** `(db) Add books.description TEXT — catalog/marketing blurb separate from full content [files: db/migration/V1__initial_schema.sql]`
- **2026-05-10** `(docs) submission.md prompt-by-commit log — reverse-chronological Prompt/Result per git hash for takehome AI traceability [files: submission.md]`
- **2026-05-10** `(ci) setup-go use go.mod — fixes PR build (was Go 1.20 vs module 1.22) [files: .github/workflows/go.yml, .gitignore]`
- **2026-05-10** `(api) Users REST + JWT auth + docs/api-contract — register/login, CRUD, Bearer middleware; submission moved to docs/submission.md [files: internal/*, main.go, docs/api-contract.md, README.md, .gitignore, .cursor/rules/agents-md.mdc]`
- **2026-05-10** `(testing, ci) Drop Postgres repo integration tests; CI workflow renamed Build and test — tests stay dependency-free; workflow labels compile vs test steps [files: internal/repository/user_repo_integration_test.go (removed), README.md, docs/api-contract.md, .github/workflows/go.yml]`
- **2026-05-10** `(auth, db) Bcrypt passwords — Flyway V2 password_hash, register/login JSON email+password, repository AuthCredentials for login [files: db/migration/V2__users_password_hash.sql, internal/auth/password.go, internal/domain/password.go, internal/repository/*, internal/service/user_service.go, internal/handlers/auth.go, docs/api-contract.md, README.md]`
- **2026-05-10** `(docs) api-contract.md frontend-only — drop backend env tables and operator/curl runbook from contract; README remains operator reference [files: docs/api-contract.md, AGENTS.md, docs/submission.md]`
- **2026-05-10** `(api) Self password change; forbid self email/role — PATCH with current_password/new_password; GetAuthCredentialsByID + UpdatePasswordHash [files: internal/service/user_service.go, internal/repository/*, internal/handlers/users.go, internal/api/json.go, internal/domain/password.go, docs/api-contract.md, docs/submission.md]`
- **2026-05-10** `(api) Books and entitlements CRUD — role-based catalog vs content access; member entitlement create/read; librarian book CRU + entitlement read; admin book CRUD + entitlement CRU [files: internal/domain/book.go, internal/domain/entitlement.go, internal/repository/book_repo.go, internal/repository/entitlement_repo.go, internal/service/book_service.go, internal/service/entitlement_service.go, internal/handlers/books.go, internal/handlers/entitlements.go, main.go, docs/api-contract.md, docs/submission.md]`
- **2026-05-10** `(api) Paginated filterable book list without content; users/me/library — ListCatalog + GetCatalogByIDs, entitlement helpers for active sub/purchases [files: internal/repository/book_repo.go, internal/repository/entitlement_repo.go, internal/service/book_service.go, internal/handlers/books.go, main.go, docs/api-contract.md, docs/submission.md]`
- **2026-05-10** `(api) OptionalBearerAuth for GET /books — anonymous catalog + detail; invalid token if header present returns 401 [files: internal/middleware/auth.go, internal/handlers/books.go, main.go, docs/api-contract.md, docs/submission.md]`
- **2026-05-10** `(git) Commits 07fb1c7 + 5eaf51e on feat/books-entitlements-api — split catalog data/library vs public optional-JWT routing; push + PR [files: see those commits]`
- **2026-05-10** `(api) POST /users/me/subscription/cancel — members end ACTIVE subscription (CANCELLED); ErrNoActiveSubscription 404; admin PATCH unchanged [files: internal/service/entitlement_service.go, internal/handlers/entitlements.go, main.go, internal/api/json.go, internal/domain/entitlement.go, docs/api-contract.md, internal/service/*_test.go]`
- **2026-05-10** `(git) Push feat/self-service-subscription-cancel; PR #6 — self-service subscription cancel [files: —]`
- **2026-05-10** `(db, api) Subscription period + cancel at period end — V3 renewed_at/cancelled_at; ends_at = renewed_at+30d; cancel sets cancelled_at; access until ends_at [files: db/migration/V3__subscription_period_renewal_cancel_at.sql, internal/repository/entitlement_repo.go, internal/service/entitlement_service.go, internal/domain/entitlement.go, docs/api-contract.md]`
- **2026-05-11** `(docs, dx) Track AGENTS.md and docs/submission.md; WHY-only code comments — reviewers get agent context + submission log in-repo; comments explain intent/rationale across layers [files: .gitignore, AGENTS.md, docs/submission.md, main.go, internal/**/*.go]`
- **2026-05-11** `(dx) More WHY/WHAT comments on handlers, services, repositories — easier review of HTTP vs domain vs SQL responsibilities [files: internal/handlers/*.go, internal/service/*.go, internal/repository/*.go, internal/api/json.go, internal/middleware/auth.go, docs/submission.md]`
- **2026-05-11** `(api) GET /books/recent — optional-auth top five books by added_at for home-page new arrivals; same BookListItem shape as catalog list [files: main.go, internal/repository/book_repo.go, internal/service/book_service.go, internal/handlers/books.go, internal/service/book_entitlement_service_test.go, docs/api-contract.md]`
- **2026-05-11** `(api) Staff user + entitlement filters — GET /users for LIBRARIAN+ADMIN with q/role/user_id; GET /entitlements/staff with user_id/book_id/type/status; members keep GET /entitlements [files: main.go, internal/domain/*.go, internal/repository/*.go, internal/service/*.go, internal/handlers/*.go, docs/api-contract.md]`
- **2026-05-11** `(docs, dx) Track .cursor/rules/agents-md.mdc — selective .gitignore so agent rule ships with repo; submission log PR #9 links; remove “gitignored” doc drift [files: .gitignore, .cursor/rules/agents-md.mdc, AGENTS.md, docs/submission.md]`
- **2026-05-11** `(docs) README overview — product/features, Mermaid architecture + ER, testing/CI; strip env/secrets/curl/flyway recipes from public README [files: README.md, docs/submission.md]`


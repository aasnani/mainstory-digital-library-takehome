# Mainstory Digital Library (Go) — agent context

This repo is a takehome MVP for a paid digital storybook library with two monetization models:

- **Subscription**: unlimited access
- **One-time purchase**: access to specific books only

This file exists to keep implementation decisions **clear, consistent, and reviewable** as the codebase grows.

> Note: `AGENTS.md` may be listed in `.gitignore` for local iteration; however, when present in the working tree it is **binding** guidance for agents working in this repo.

---

## Repo map (quick navigation)

This section is intentionally **non-speculative**.

- Only list files/folders that **exist in the repo right now**.
- When new top-level directories are created (e.g. `cmd/`, `internal/`, `web/`), update this map in the same change series.

- **Entry points**
  - `README.md`: high-level overview.
  - `submission.md` (**gitignored**, local): product + system design write-up required by the takehome prompt.
  - `AGENTS.md`: this file (repo map + conventions + agent memory log).
  - `.cursor/rules/agents-md.mdc`: Cursor rule to enforce this file’s upkeep (may be gitignored).

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

The submission requires describing AI usage. Capture this in `submission.md`:

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


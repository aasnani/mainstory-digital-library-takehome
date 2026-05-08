# Mainstory Digital Library — agent context

This repo is a takehome MVP for a paid digital storybook library with two monetization models:

- **Subscription**: unlimited access
- **One-time purchase**: access to specific books only

This file exists to keep implementation decisions **clear, consistent, and reviewable** as the codebase grows.

> Note: `AGENTS.md` may be listed in `.gitignore` for local iteration; however, when present in the working tree it is **binding** guidance for agents working in this repo.

---

## Repo map (quick navigation)

This repo will evolve, but keep this structure stable unless there’s a clear reason to change it.

- **Entry points**
  - `README.md`: high-level overview, how to run locally, env vars, and deployment links.
  - `submission.md` (**gitignored**, local): product + system design write-up required by the takehome prompt.
  - `AGENTS.md`: this file (repo map + conventions + agent memory log).
  - `.cursor/rules/agents-md.mdc`: Cursor rule to enforce this file’s upkeep (may be gitignored).

- **Backend**
  - `backend/` (preferred) or `server/`: Node.js service
    - `src/` application code
    - `src/modules/` (or `src/domains/`): `auth`, `users`, `books`, `payments`, `entitlements`
    - `src/db/`: database client + migrations / schema
    - `src/api/`: HTTP routing/controllers
    - `src/services/`: domain services (purchase/subscription/access control)
    - `src/policies/` (or `src/access/`): access-control checks / guards
    - `src/lib/`: shared utils (error handling, validation, time, ids)
    - `test/` or `src/**/__tests__/`: tests

- **Frontend**
  - `frontend/`: Lovable (or similar) generated UI connecting to backend
    - book list/detail, login, subscribe/purchase flows, CRUD admin-ish flows

- **Infra / deploy (optional for MVP)**
  - `docker-compose.yml`: local DB + services
  - `scripts/`: dev scripts (seed, reset DB, etc.)

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

### Data model (target)

At minimum the backend should represent:

- `users`
- `books`
- `subscriptions` (current status + period fields if needed)
- `purchases` (user ↔ book)

Prefer unique constraints that make idempotency easy (e.g. unique `(userId, bookId)` in purchases).

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


# Submission write-up

Path: **`docs/submission.md`**. This file is **tracked** in git alongside `AGENTS.md` so reviewers can read product/system notes in one clone.

## Key product decisions & assumptions

- **Target users**:
- **Primary jobs-to-be-done**:
- **Assumptions**:

## High-level roadmap

- **MVP (this takehome)**:
- **Next**:
- **Later**:

## Success metrics & measurement

- **Activation**:
- **Engagement**:
- **Conversion**:
- **Retention**:

## Access control design (include edge cases)

- **Rules**:
- **Where checks live in code**:
- **Edge cases handled**:
- **Trade-offs**:

## System design

- **Architecture overview**:
- **Data model**:
  - **2026-05-10**: Flyway **V2** (`db/migration/V2__users_password_hash.sql`): **`users.password_hash`** TEXT NOT NULL — bcrypt via **`golang.org/x/crypto/bcrypt`**; clients send plaintext **`password`** over HTTPS; **never** stored or returned except as hash in DB.
  - **2026-05-10**: Flyway V1 (`db/migration/V1__initial_schema.sql`): **`users`** — id UUID, email unique, `role` + CHECK (MEMBER/LIBRARIAN/ADMIN). **`books`** — title, `description` (catalog/marketing copy), author, genre, `is_fiction`, `published_date`, `added_at`, language, price_cents, content (full text). **`entitlements`** — CHECK on `type` (SINGLE_PURCHASE/SUBSCRIPTION) and `status` (ACTIVE/CANCELLED/PAST_DUE); CHECK that SINGLE_PURCHASE ⇒ `book_id` NOT NULL and SUBSCRIPTION ⇒ `book_id` NULL (all-books subscription model). **Indexes**: partial UNIQUE on `(user_id, book_id)` WHERE SINGLE_PURCHASE (idempotent “already owned” / no duplicate purchase rows); partial UNIQUE on `user_id` WHERE SUBSCRIPTION + ACTIVE (one active subscription per user). Postgres `gen_random_uuid()` where used (PG13+).
- **API surface**:
  - **2026-05-10**: **`docs/api-contract.md`** documents **browser integration only** (URLs, JSON, auth, errors). Backend env vars, Flyway, and shell workflows stay in **README**, not the contract.
  - **2026-05-10**: **Self-service password change** — **`PATCH /users/me`** (and self **`PATCH /users/:id`**) accept **`current_password`** + **`new_password`** only; **cannot** **`PATCH`** own **`email`**/**`role`**. **ADMIN** changes another user’s **`email`**/**`role`** via **`PATCH /users/:id`**; password fields rejected on that path.
  - **2026-05-10**: **Books + entitlements HTTP API** — **`MEMBER`**: read books (content gated by **`SUBSCRIPTION`** / **`SINGLE_PURCHASE`**); create/read **own** entitlements only. **`LIBRARIAN`**: CRU books (no delete), read all entitlements. **`ADMIN`**: CRUD books, CRU entitlements (no delete). Access evaluation in **`BookService`** + **`EntitlementRepository`** (subscription OR per-book purchase).
  - **2026-05-10**: **Catalog list** — **`GET /books`** uses **`ListCatalog`** (no **`content`** column in SQL); optional filters **`q`**, **`title`**, **`author`**, **`genre`**, **`language`**, **`is_fiction`**, price range; **`q`/`title`/`author`** require **≥ 2** runes when set. **`GET /users/me/library`** returns subscription + purchased books (metadata only) in one response for “my purchases” UI.
  - **2026-05-10**: **Public catalog browse** — **`GET /books`** and **`GET /books/:id`** require no JWT (**`OptionalBearerAuth`**); guests see metadata only and **LOCKED** access; valid Bearer applies entitlement/staff rules. Purchase/entitlement APIs remain authenticated.
  - **2026-05-11**: **`GET /api/v1/books/recent`** — returns at most **five** **`BookListItem`** rows ordered by **`added_at`** descending (newest catalog additions first). Same optional Bearer and **`is_accessible`** / **`access_reason`** rules as **`GET /books`**; no **`content`** in SQL. Route registered before **`GET /books/:id`** so **`recent`** is not treated as a UUID.
  - **2026-05-10**: **Self-service subscription cancel** — **`POST /users/me/subscription/cancel`** (Bearer): sets **`cancelled_at`**; **`status`** stays **`ACTIVE`** until **`ends_at`** (30-day window from **`renewed_at`** on subscribe). **`no_active_subscription`** (**404**) when no current paid period. Flyway **V3** adds **`renewed_at`**, **`cancelled_at`**; backfills **`ends_at`** for existing subs.
- **Error model**:
- **Automated testing (2026-05-10)**: `go test ./...` is **fully in-process**—fakes/mocks for stores, no live Postgres in CI or required locally. Repository integration tests against `DATABASE_URL` were removed so the suite has **no external dependencies**. GitHub Actions workflow **Build and test** runs **`go build ./...`** then **`go test ./...`** with steps named accordingly.

## AI tooling usage

- **Tools used**:
- **Prompts / iterations**: See **Prompt & outcome log (by commit)** below—append there after each commit so prompts stay tied to shipped diffs.
- **Improvements made**:
- **Verification / guardrails**:

### How I prompt (habits)

Keep this short; **details live in the commit log.**

- Work in **Cursor** with **`AGENTS.md`** + **`.cursor/rules/agents-md.mdc`** (tracked in git) for product/architecture guardrails.
- Ask for **scoped** work (**database only**, **review only**, etc.) when you want a tight answer.
- After **`git commit`** (or right before push), add **one block** under **Prompt & outcome log** for that commit hash: what you asked → what landed.
- Prefer **one prompt ↔ one commit** when possible so the snapshot stays honest.

## Prompt & outcome log (by commit)

_Order: newest commit first. Copy the short hash from `git log --oneline`. Update after each commit._

### 2026-05-11 — README product + architecture + testing (no env/secrets)

- **Prompt**: On new branch from synced `main`, expand README with description, features, architecture and schema diagrams, testing/CI; omit env vars and sensitive operational copy; commit, push, PR.
- **Result**: README rewritten around **docs/api-contract.md** + **AGENTS.md** links; Mermaid **flowchart** + **erDiagram**; feature table; testing table + `go test` / Actions; removed prior env tables, curl/flyway/psql bootstrap recipes, and secret-generation hints; operator note points to **`internal/config`** without naming keys. PR: https://github.com/aasnani/mainstory-digital-library-takehome/pull/11

### 2026-05-11 — Track Cursor agent rule; align submission after merge

- **Prompt**: Merged PR omitted some `AGENTS.md` / `docs/submission.md` updates; remove rules that blocked shipping them; sync `main`, commit docs + rule visibility, open PR.
- **Result**: `.gitignore` uses negated patterns so **`.cursor/rules/agents-md.mdc`** is tracked; rule text no longer implies `submission.md` is gitignored; **PR #9** links in submission log; **AGENTS.md** repo map notes the rule is versioned. PR: https://github.com/aasnani/mainstory-digital-library-takehome/pull/10

### 2026-05-11 — Staff-filtered GET /users and GET /entitlements/staff

- **Prompt**: Add filters for entitlements and users like books; librarian+admin only; rename branch and update PR title/body; frontend summary including top-5 recent books.
- **Result**: **`GET /users`** now **`RequireAnyRole(LIBRARIAN, ADMIN)`** with query **`q`** (email substring, ≥2), **`role`**, **`user_id`**; **`UserRepository.ListFiltered`**. **`GET /entitlements/staff`** same role gate with **`user_id`**, **`book_id`**, **`type`**, **`status`**; **`ListAllFiltered`**. **`GET /entitlements`** unchanged for members. Shipped on **`main`** via https://github.com/aasnani/mainstory-digital-library-takehome/pull/9 (PR #8 closed when its head branch was deleted).

### 2026-05-11 — GET /books/recent (top five by added_at)

- **Prompt**: Sync main, add endpoint for top five recently added books, tests, commit, push, PR; summarize API for frontend.
- **Result**: **`GET /api/v1/books/recent`** under optional Bearer; **`BookRepository.ListRecentCatalogTop5`**, **`BookService.RecentlyAdded`**, **`BooksHandler.RecentlyAdded`**; service tests for order/limit and subscription flags; **`docs/api-contract.md`** updated. Landed with staff filters in https://github.com/aasnani/mainstory-digital-library-takehome/pull/9

### 2026-05-11 — Additional handler/service/repository comments

- **Prompt**: Add more comments; use short WHAT when no strong WHY.
- **Result**: Expanded comments on user/book/entitlement handlers, `api` error types, `UserService`/`BookService`/`EntitlementService`, and repository methods; noted orphan purchase skip in `MyLibrary`. Commit: `66b3f3a` on PR #7.

### 2026-05-11 — Track agent docs, add WHY-only code comments

- **Prompt**: Sync `main`, branch, add rationale comments across Go code (why structs/types/functions exist), stop gitignoring `AGENTS.md` and `docs/submission.md`, commit, push, open PR.
- **Result**: `.gitignore` no longer excludes those docs; `AGENTS.md` + `docs/submission.md` tracked; `main.go` + `internal/**` carry intent-focused comments; `.cursor/rules/agents-md.mdc` later tracked via selective `.gitignore` (see 2026-05-11 docs PR). PR: https://github.com/aasnani/mainstory-digital-library-takehome/pull/7

### `24b98a2` — Subscription period and cancel-at-period-end

- **Prompt**: Subscription should stay active until end of period from last renewal; cancel same day still runs until renewed_at + 30 days.
- **Result**: Flyway **V3** (`renewed_at`, `cancelled_at`, backfill **`ends_at`**); **`POST /entitlements`** SUBSCRIPTION sets **`renewed_at`** + **`ends_at`**; cancel sets **`cancelled_at`** only; **`ExpireStaleSubscriptionsForUser`** flips **`CANCELLED`** when **`ends_at`** passed; access queries require **`ends_at > NOW()`**.

### `1ad0d3f` — Add self-service subscription cancel for members

- **Prompt**: New branch; support self-service subscription cancel; include tests.
- **Result**: **`POST /api/v1/users/me/subscription/cancel`** → **`EntitlementService.CancelMySubscription`**, domain **`ErrNoActiveSubscription`**, **`docs/api-contract.md`** + **`internal/service/entitlement_service_test.go`**; test fake **`GetActiveSubscriptionEntitlement`** reads **`entByID`** and **`Update`** syncs **`subs`** map. PR: https://github.com/aasnani/mainstory-digital-library-takehome/pull/6

### `5eaf51e` — Expose catalog GETs with optional JWT for anonymous browse

- **Prompt**: Allow unauthenticated catalog browse; optional Bearer when present so entitled users still see **`content`** and correct **`is_accessible`** on list/detail. Later in the same thread: stage in smaller commits, push, open a PR, curls for **`you@example.com`** / **`SecondPass456`**.
- **Result**: **`OptionalBearerAuth`**; catalog **`GET`**s on a route group without mandatory Bearer; bad token when header sent → **401**. **`docs/api-contract.md`**: public catalog + integration checklist. PR https://github.com/aasnani/mainstory-digital-library-takehome/pull/5 (branch stacks on **`07fb1c7`**).

### `07fb1c7` — Add filterable catalog list, omit content in list SQL, and /users/me/library

- **Prompt**: Paginated filterable **`GET /books`** without loading **`content`** in list queries; **`GET /users/me/library`** for “my purchases” in one response; min length on search query params; keep contract aligned.
- **Result**: **`ListCatalog`** / **`GetCatalogByIDs`**, entitlement repo helpers (active sub, purchases), **`BookListFilter`** validation, **`MyLibrary`** in **`BookService`**, handler + route; service tests with fakes; **`docs/api-contract.md`** filters + library (catalog still Bearer-only until **`5eaf51e`**).

### `c743f30` — Add books and entitlements APIs with role-based access control

- **Prompt**: Books + entitlements CRUD with RBAC: members list/read books with content gated by entitlements, create/read own entitlements; librarians CRU books and read entitlements; admins full book CRUD and entitlements CRU; centralize access checks; tests in line with users API.
- **Result**: Domain **`Book`** / **`Entitlement`**, **`BookRepository`** / **`EntitlementRepository`**, **`BookService`** (list/get/create/update/delete with **`is_accessible`** / **`access_reason`**, subscription OR purchase), **`EntitlementService`**, Gin handlers + **`main.go`** routes; **`docs/api-contract.md`** books and entitlements sections; **`go test ./...`** for service layer. Parent for the catalog follow-ups **`07fb1c7`** / **`5eaf51e`**.

### `f667c5e` — Add users REST API with JWT auth and API contract

- **Prompt**: Implement the users REST + mocked JWT plan (internal layout, CRUD, `docs/api-contract.md`, `docs/submission.md` path, AGENTS/cursor rules).
- **Result**: `POST/GET/PATCH/DELETE` under `/api/v1`, pgx + HS256 JWT, `docs/api-contract.md` tracked, `.gitignore` → `docs/submission.md`, local `AGENTS.md` + `.cursor/rules/agents-md.mdc` updated (not in commit).

### `9fb6c71` — Align CI Go toolchain with go.mod

- **Prompt**: PR CI Go build failing—fix it.
- **Result**: `setup-go@v5` + **`go-version-file: go.mod`** so Actions uses Go **1.22** (matches `go.mod`); previous workflow pinned **1.20**, which cannot satisfy the module. `.gitignore`: ignore default binary name `mainstory-digital-library-takehome`.

### `8e103b6` — Add books.description for catalog copy

- **Prompt**: Add a description field to the book schema.
- **Result**: `books.description TEXT NOT NULL DEFAULT ''` in `V1__initial_schema.sql`; pushed on schema branch.

### `4bff9c4` — Drop synopsis, reading_level, cover_image_url from books schema

- **Prompt**: Remove synopsis, reading level, and cover image URL from books.
- **Result**: Those three columns removed from `books` in Flyway V1.

### `87ad7ee` — Extend books metadata and enforce entitlement invariants

- **Prompt**: Enrich books with genre, fiction flag, dates, language, etc.; enforce idempotent purchases, one active subscription per user, and CHECK constraints linking entitlement type to `book_id` shape.
- **Result**: Expanded `books`; `entitlements` CHECKs on type/status/shape; partial unique indexes for purchase dedupe and single active subscription.

### `3c30aa9` — Bundle schema into Flyway V1 only

- **Prompt**: Fold prior “V2” changes into a single V1 migration (Flyway not applied / PR not merged yet).
- **Result**: `V1__initial_schema.sql` contains roles + content + constraints; `V2__…` file removed.

### `abf7841` — Add Flyway V2 for user roles and book content

- **Prompt**: _(historical — superseded by `3c30aa9`)_ Add users.role with CHECK; book content; separate V2 migration.
- **Result**: Introduced `V2__user_roles_and_book_content.sql` (later deleted when bundled into V1).

### `86201ad` — Add Flyway V1 database schema

- **Prompt**: Add Flyway schema v1 (`users`, `books`, `entitlements`); branch, commit, push, PR; document remote Flyway command.
- **Result**: `db/migration/V1__initial_schema.sql` + README Flyway section; PR opened.

### `79565fc` — Create build and test workflow on PR

- **Prompt**: _(if filled via AI:)_ CI for Go on PRs to `main`.
- **Result**: `.github/workflows/go.yml` runs `go build` and `go test` on pull requests.

### `f0fee14` — Merge pull request #1 from aasnani/chore/agents-and-rules

- **Prompt**: _N/A (GitHub merge commit)._
- **Result**: Merged backend skeleton / agents-related PR into `main`.

### `72e8b01` — Add Gin healthcheck endpoint

- **Prompt**: Minimal Go + Gin app with `/healthcheck` → `UP`; remove `AGENTS.md` / Cursor rule from PR tracking but keep locally gitignored; PR title focused on app skeleton.
- **Result**: `main.go`, `go.mod`, `.gitignore` updates; health-only server; deployment build flags in README.

### `1107826` — Make AGENTS.md Go-specific and non-speculative

- **Prompt**: Backend is Go; don’t assume repo layout before it exists; data model `users` / `books` / `entitlements` only without key/schema speculation.
- **Result**: `AGENTS.md` rewritten for Go and non-speculative repo map; Cursor rule aligned.

### `2643bb9` — Add AGENTS.md and Cursor rule scaffold

- **Prompt**: Add `AGENTS.md`, `.cursor/rules/agents-md.mdc`, `submission.md` (gitignored); establish conventions; branch, commit, push, PR.
- **Result**: Initial agent docs + rules + `.gitignore` entries; takehome-oriented structure.

### `33c2961` — Initial commit

- **Prompt**: _N/A (repo bootstrap)._
- **Result**: `README.md` placeholder.

---

_Add new entries **above** `8e103b6` (or above whatever your latest commit is) so the log stays reverse-chronological._

## Database layer review (Flyway V1)

_Scope: persistence only—not the Go app or frontend._

- **2026-05-10** — **Strengths**
  - **`users`**: UUID PKs, unique email, **`role`** with CHECK (`MEMBER` / `LIBRARIAN` / `ADMIN`) supports librarian/admin UX and server-side authorization later without a separate permissions table for MVP.
  - **`books`**: Enough catalog fields (`description`, genre, fiction flag, dates, language, `price_cents`, `content`) to support list/detail and paywall presentation; `added_at` vs `published_date` separates catalog ingestion from publication date.
  - **`entitlements`**: Single table with **`type`** / **`status`** CHECKs plus **`entitlements_type_book_shape`** aligns rows with the product model: **subscription = all books** (`book_id` NULL), **purchase = one book** (`book_id` NOT NULL).
  - **Idempotent purchases**: Partial unique index on `(user_id, book_id)` for **`SINGLE_PURCHASE`** prevents duplicate purchase rows; retries can surface as unique violations → map to “already owned.”
  - **One active subscription per user**: Partial unique index on **`user_id`** where **`SUBSCRIPTION` + `ACTIVE`** matches the MVP “single current subscription” rule while allowing historical **`CANCELLED` / `PAST_DUE`** rows.
- **2026-05-10** — **Gaps / deferrals (acceptable at DB layer if documented in app)**
  - **`ends_at`**: No DB constraint tying **`ACTIVE`** subscription to `ends_at` (e.g. future end date)—renewals, expiry, and **`PAST_DUE`** transitions stay **application/jobs** concerns unless you add triggers or scheduled checks later.
  - **`price_cents`**: Not constrained to **≥ 0**; optional CHECK if you want DB-level sanity.
  - **Payments**: No separate **payments** / **orders** table—fine if **`entitlements`** is the ledger for a mocked-payment MVP; add if you need audit or reconciliation.
  - **PostgreSQL**: **`gen_random_uuid()`** assumes **PG13+** (or equivalent); hosted version should match.
- **2026-05-10** — **Verdict (database only)**: The schema **supports** subscription vs one-off purchase, **enforces** duplicate-purchase and single-active-subscription **invariants**, and documents **row shape** for access evaluation once the app layer queries these tables. Runtime “can read this book?” logic remains **application responsibility**.

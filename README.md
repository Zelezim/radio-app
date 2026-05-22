# RadioApp

Live-radio player with a lightweight CRM, built in **Go + Echo + HTMX**, backed by **Supabase** (PostgreSQL + Auth).

- 🎧 Mobile-first live audio player (in-car friendly).
- 👤 Email/password auth via Supabase.
- 📻 Browse programs, vote 👍/👎, rate ★1–5.
- 📬 Contact / CRM form (visits, CDs/DVDs, books, other).
- 🐳 One-command Docker deploy.

## Stack

| Layer    | Choice                                   |
|----------|------------------------------------------|
| Backend  | Go 1.22, [Echo v4](https://echo.labstack.com/) |
| DB / Auth| Supabase (PostgREST + GoTrue)            |
| Frontend | Server-rendered Go templates + HTMX + CSS |
| Deploy   | Docker / Docker Compose                  |

## Folder layout

```
radioapp/
├── cmd/server/main.go        # entrypoint, router, template loader
├── internal/
│   ├── handlers/             # one file per feature (auth, player, programs, votes, ratings, contact)
│   ├── models/               # shared structs
│   └── supabase/             # tiny HTTP wrapper over Supabase REST + Auth
├── web/
│   ├── templates/            # base.html + page templates
│   └── static/               # style.css, player.js
├── migrations/001_init.sql   # schema + seed data
├── .env.example
├── Dockerfile
└── docker-compose.yml
```

---

## Prerequisites

- **Go ≥ 1.22** (for local runs)
- **Docker** (optional, for containerised runs)
- A **Supabase project**. Grab the **Project URL**, **anon key**, and **service role key** (called `SUPABASE_SECRET_KEY` here) from *Project Settings → API*.

---

## 1. Configure environment variables

```bash
cp .env.example .env
# then edit .env and fill in:
#   SUPABASE_URL=...
#   SUPABASE_ANON_KEY=...
#   SUPABASE_SECRET_KEY=...
#   STREAM_URL=...
#   SESSION_SECRET=$(openssl rand -hex 32)
```

`SESSION_SECRET` signs the session cookie — set it to any long random string. Keep it stable across restarts, otherwise existing sessions get invalidated.

---

## 2. Apply the database migrations to Supabase

You have three options — pick the one you prefer.

### Option A — Supabase SQL Editor (easiest)

1. Open your project at <https://app.supabase.com>.
2. Go to **SQL Editor → New query**.
3. Paste the contents of [`migrations/001_init.sql`](migrations/001_init.sql).
4. Click **Run**.

### Option B — Supabase CLI

```bash
# install once: https://supabase.com/docs/guides/local-development
supabase link --project-ref <your-project-ref>
supabase db push       # if you keep the file under supabase/migrations
# or apply it directly:
psql "$DATABASE_URL" -f migrations/001_init.sql
```

### Option C — `psql`

Grab the connection string from *Project Settings → Database → Connection string (URI)* and run:

```bash
psql "postgres://postgres:[PASSWORD]@db.[PROJECT_REF].supabase.co:5432/postgres" -f migrations/001_init.sql
```

The migration creates `profiles`, `programs`, `votes`, `ratings`, `contact_requests`, two helper views (`vote_totals`, `program_rating_avg`), enables RLS on user tables, and seeds three demo programs.

---

## 3. Run locally (without Docker)

```bash
go mod tidy
go run ./cmd/server
```

Open <http://localhost:8080>.

> The server reads `.env` via `godotenv`, so you don't need to `export` anything manually.

---

## 4. Run with Docker

```bash
docker compose up --build
```

This builds a small Alpine-based image (~20 MB) and starts the container on `:8080`, reading `.env` from the project root.

To stop:

```bash
docker compose down
```

---

## How features map to code

| Feature              | Endpoint(s)                     | Handler file                          |
|----------------------|---------------------------------|---------------------------------------|
| Live player          | `GET /`                         | `internal/handlers/player.go`         |
| Sign up / sign in    | `GET/POST /register`, `/login`  | `internal/handlers/auth.go`           |
| Logout               | `POST /logout`                  | `internal/handlers/auth.go`           |
| Programs listing     | `GET /programs`                 | `internal/handlers/programs.go`       |
| Vote (HTMX)          | `POST /vote` (auth)             | `internal/handlers/votes.go`          |
| Rating (HTMX)        | `POST /rate` (auth)             | `internal/handlers/ratings.go`        |
| Contact / CRM form   | `GET/POST /contact`             | `internal/handlers/contact.go`        |
| Health probe         | `GET /healthz`                  | `cmd/server/main.go`                  |

Protected endpoints are wrapped by `Handlers.RequireAuth` and read the session from a signed HTTP-only cookie (`rb_session`).

---

## Security notes

- The backend uses Supabase's **service role key** for trusted writes — never expose it to the browser. RLS is enabled so that even if someone calls PostgREST with the **anon key**, they can only read the public `programs` table.
- Sessions are stored in an **HTTP-only, SameSite=Lax cookie** signed with `SESSION_SECRET` (HMAC-SHA256). Behind TLS in production, set `Secure` on the cookie (see `internal/handlers/handlers.go`).
- Passwords never touch our backend storage — Supabase Auth handles them.

---

## Troubleshooting

| Symptom                                          | Likely cause / fix                                                                 |
|--------------------------------------------------|------------------------------------------------------------------------------------|
| `missing required env var SUPABASE_URL`          | `.env` not loaded — ensure it lives next to the binary or pass via `--env-file`.   |
| `supabase auth 400: Email not confirmed`         | Disable email confirmation in *Auth → Providers → Email* during development.       |
| Programs list is empty                           | Migration didn't run, or seed `insert` was skipped. Re-run `001_init.sql`.         |
| Audio button does nothing on iOS                 | Safari requires a user tap before audio plays — the button itself triggers `play`. |
| HTMX vote/rate buttons return 401                | Session expired. Click **Login** again.                                            |

---

## License

MIT — do whatever you want, just keep the notice.

# ARTEX × CyberNeurova

> **ARTEX** is an open-source, LLM-driven penetration-testing platform. This fork ([CyberNeurova/artex_cn](https://github.com/CyberNeurova/artex_cn)) adds first-class support for [CyberNeurova](https://cyberneurova.com) models and authentication — no OpenAI or Anthropic account required.

Based on [BaaRai/ARTEX](https://github.com/BaaRai/ARTEX) · License: AGPL-3.0

---

## What's different from upstream

| Feature | Upstream ARTEX | This fork |
|---|---|---|
| CyberNeurova OpenRoute provider | — | Built-in (`cyberneurova` format) |
| Dynamic model list | — | Fetched live from `/v1/models` |
| Sign in with CyberNeurova | — | OAuth 2.0 PKCE (optional) |
| OpenRoute API key env var | — | `OPENROUTE_API_KEY` |

Everything else is identical to upstream.

---

## Prerequisites

- **Docker + Docker Compose** (recommended) — or Go 1.22+ and Node.js 20+ for a manual build
- A [CyberNeurova OpenRoute](https://cyberneurova.com/api/) API key (`sk-cnor-…`)
- 2 GB RAM minimum; 4 GB recommended

---

## Quick start (Docker Compose)

### 1 · Clone

```bash
git clone https://github.com/CyberNeurova/artex_cn.git
cd artex_cn
```

### 2 · Configure

```bash
cp .env.example .env
```

Open `.env` and fill in at minimum:

```env
POSTGRES_PASSWORD=change-me-please   # pick something strong

# CyberNeurova OpenRoute
OPENROUTE_API_KEY=sk-cnor-…
ARTEX_LLM_PROVIDER=cyberneurova
# Leave ARTEX_LLM_MODEL empty — the UI will let you pick from live models
```

### 3 · Start

```bash
docker compose up -d
```

ARTEX is now running at **http://localhost:8787**.

### 4 · Set a password

The first time the UI opens you will be shown a **setup page**. Enter a strong password (12+ characters) and the **setup token** printed in the backend log:

```bash
docker compose logs artex | grep "Setup token"
```

---

## Choosing a CyberNeurova model

After signing in, go to **System → LLM** and click **+ New profile**:

1. Set **Format** to **Cyberneurova (OpenRoute)** — the base URL fills in automatically.
2. Paste your `OPENROUTE_API_KEY` in the **API key** field.
3. Click the **fetch ↗** button next to the model field to load available models live.
4. Pick a model (e.g. `cyberneurova/qwen3-8b-flash`) and click **Save**.
5. Click **Activate** on the profile row to make it the global default.

You can also set `ARTEX_LLM_PROVIDER=cyberneurova` and `ARTEX_LLM_MODEL=<model-id>` in `.env` to skip the UI step entirely.

---

## Sign in with CyberNeurova (OAuth — optional)

Instead of a local password, you can allow login via your CyberNeurova account.

### Register your OAuth app

1. Open [chat.cyberneurova.ai](https://chat.cyberneurova.ai) → **Settings → Developer** (requires a Pro or Pro Max plan).
2. Click **New application**.
3. Set the **Redirect URI** to `http://localhost:8787/api/auth/cyberneurova/callback`  
   (replace `localhost:8787` with your actual host and port if running behind a reverse proxy).
4. Note the **Client ID** and **Client secret**.

### Add the credentials to `.env`

```env
ARTEX_OAUTH_CLIENT_ID=cyberneurova_…
ARTEX_OAUTH_CLIENT_SECRET=cnsec_…
ARTEX_OAUTH_REDIRECT_URI=http://localhost:8787/api/auth/cyberneurova/callback
```

Restart ARTEX (`docker compose restart artex`) and the **Sign in with CyberNeurova** button appears on the login page.

> **Security note:** The OAuth flow uses PKCE (S256). Anyone who completes the OAuth flow gets full admin access to your ARTEX instance, so protect the redirect URI and keep the client secret out of version control. Password login remains available alongside OAuth.

---

## Environment variable reference

| Variable | Default | Description |
|---|---|---|
| `ARTEX_BIND_ADDR` | `127.0.0.1` | Address to listen on. Use `0.0.0.0` to expose on all interfaces. |
| `ARTEX_TRUSTED_PROXIES` | _(empty)_ | Comma-separated CIDRs of reverse proxies whose `X-Forwarded-For` the login rate limiter trusts. |
| `ARTEX_SECRET_KEY` | auto-generated | 32-byte base64/hex key for encrypting secrets in the database. Required when `./keys` is not persistent. |
| `POSTGRES_USER` | `artex` | PostgreSQL username. |
| `POSTGRES_PASSWORD` | — | PostgreSQL password (**required**). |
| `POSTGRES_DB` | `artex` | PostgreSQL database name. |
| `ARTEX_LLM_PROVIDER` | _(none)_ | LLM provider. Set to `cyberneurova` to use CyberNeurova OpenRoute. Other values: `anthropic`, `openai`, `openai-responses`, `ollama`. |
| `ARTEX_LLM_MODEL` | _(none)_ | Model ID (e.g. `cyberneurova/qwen3-8b-flash`). Leave empty to pick via the UI. |
| `ARTEX_LLM_BASE_URL` | provider default | Override the provider base URL. |
| `ARTEX_LLM_PROXY` | _(none)_ | HTTP/SOCKS5 proxy for LLM requests. |
| `OPENROUTE_API_KEY` | _(none)_ | CyberNeurova OpenRoute API key (`sk-cnor-…`). |
| `ANTHROPIC_API_KEY` | _(none)_ | Anthropic API key. |
| `OPENAI_API_KEY` | _(none)_ | OpenAI API key. |
| `ARTEX_OAUTH_CLIENT_ID` | _(none)_ | CyberNeurova OAuth client ID. |
| `ARTEX_OAUTH_CLIENT_SECRET` | _(none)_ | CyberNeurova OAuth client secret. |
| `ARTEX_OAUTH_REDIRECT_URI` | _(none)_ | OAuth redirect URI (must match the one registered in your app). |
| `ARTEX_TASK_BUDGET_TOKENS` | unlimited | Maximum tokens per task across all rounds and workers. |
| `ARTEX_TASK_BUDGET_TURNS` | unlimited | Maximum planner rounds per task. |
| `ARTEX_INTENT_BUDGET_TOKENS` | unlimited | Maximum tokens for a single worker run. |
| `ARTEX_INTENT_BUDGET_TURNS` | unlimited | Maximum turns for a single worker run. |
| `ARTEX_NOISE_PROFILE` | off | Request noise profile (`quiet`, `balanced`, `aggressive`). |
| `ARTEX_UPDATE_CHECK` | off | Set `1` to enable the update banner (checks GitHub releases at most once per 24 h). |

---

## Manual build (without Docker)

### Requirements

- Go 1.22+
- Node.js 20+, pnpm 9+
- PostgreSQL 15+

### Steps

```bash
# 1. Build the Next.js frontend
cd web
pnpm install
pnpm build          # outputs to web/out/
cd ..

# 2. Build the Go binary
go build -o artex .

# 3. Run (set env vars first or export them)
export ARTEX_PG_DSN="postgres://artex:password@localhost:5432/artex?sslmode=disable"
export OPENROUTE_API_KEY="sk-cnor-…"
export ARTEX_LLM_PROVIDER="cyberneurova"
./artex
```

The binary serves the built frontend from `web/out/` automatically.

---

## Upgrading

This fork tracks the upstream `main` branch. To pull upstream changes:

```bash
git remote add upstream https://github.com/BaaRai/ARTEX.git
git fetch upstream
git merge upstream/main
```

Resolve any conflicts (usually none — this fork's changes are additive), then rebuild with `docker compose up -d --build`.

---

## Available CyberNeurova models

Models are fetched live from the OpenRoute API, so the list is always current. At the time of writing the lineup includes:

- `cyberneurova/qwen3-8b-flash` — fast, low cost
- `cyberneurova/qwen3-32b` — mid-tier reasoning
- `cyberneurova/deepseek-v3` — strong code and analysis
- `cyberneurova/kimi-k3-1` — long-context tasks

Use the **fetch ↗** button in the LLM profile editor for the live list.

---

## Troubleshooting

**Backend log says "no admin password is set"**  
The setup token is printed at startup. Run `docker compose logs artex | grep "Setup token"` and use it on the `/setup` page.

**OAuth callback returns "invalid or expired state"**  
The state cookie expires after 10 minutes. Start the sign-in flow again.

**Models list is empty when I click fetch ↗**  
Check that your `OPENROUTE_API_KEY` is valid and that the base URL is `https://openroute.cyberneurova.com/v1`. You can verify the key with:
```bash
curl -H "Authorization: Bearer $OPENROUTE_API_KEY" https://openroute.cyberneurova.com/v1/models
```

**Docker Compose build fails**  
Make sure you have at least 4 GB of disk space and that the Docker daemon is running. The image is built locally (there is no pre-built image in a registry).

---

## Security

ARTEX is a penetration-testing tool. Use it only on systems you own or have explicit written permission to test. See [`web/src/app/(auth)/login/page.tsx`](web/src/app/(auth)/login/page.tsx) for the full Terms of Use.

---

## License

AGPL-3.0 — same as upstream ARTEX. Any derivative work must be open-sourced under the same license.

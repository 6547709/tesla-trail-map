# TeslaMate Trail Map

**English** · [简体中文](./README.md)

> A drive-trail playback tool that pairs with [TeslaMate](https://github.com/teslamate-org/teslamate).
> Backend: Go + Gin + pgx. Frontend: Leaflet + leaflet.motion.
> Current version: **v1.6.6** · see the [CHANGELOG](./CHANGELOG.md)

---

## ✨ Features

- 🚗 **Multi-vehicle switcher** — auto-detected from your DB; switch by **car name**, never by raw `car_id`.
- 🗺️ **Animated playback** of historical drives on Leaflet + OpenStreetMap with a Tesla-style car marker.
- ⏱️ **Time-range queries** with 0.5x / 1x / 3x / custom playback speed.
- 🧠 **Smart speed recommendation** based on total drive duration (≥3h→3x, 1-3h→1x, short-and-dense→0.5x).
- 🚀 **Server-side downsampling**: default `max_points=2000` per drive — 45k-point trips become 2k-point silky animations.
- 🛡️ **Production-grade safety**: strict input validation, friendly timeout messages, observable pgxpool stats, AbortController to defeat races.
- 📦 **One-shot Docker build** — multi-stage, alpine runtime.

---

## 📋 Requirements

| Component | Version |
|---|---|
| Go | 1.25+ (build) |
| PostgreSQL | 12+ (verified on PG 18.3) |
| TeslaMate | any recent version (provides `cars` / `drives` / `positions`) |

---

## 🚀 Quick Start

### Run with Go

```bash
git clone https://github.com/6547709/tesla-trail-map.git
cd tesla-trail-map

export DATABASE_HOST=192.168.66.200
export DATABASE_PORT=54320
export DATABASE_USER=teslamate
export DATABASE_PASS=your_password
export DATABASE_NAME=teslamate

go run .
```

Open http://localhost:8080.

### Run with Docker

#### Pull the official image (recommended)

Every `vX.Y` tag triggers a GitHub Actions workflow that builds and pushes a
multi-arch image (`linux/amd64` + `linux/arm64`) to GHCR:

```bash
docker pull ghcr.io/6547709/tesla-trail-map:1.6.6
# or :latest (kept in sync with the most recent tag)
docker pull ghcr.io/6547709/tesla-trail-map:latest

docker run -d --name tesla-trail-map -p 8080:8080 \
  -e DATABASE_HOST=host.docker.internal \
  -e DATABASE_PORT=5432 \
  -e DATABASE_USER=teslamate \
  -e DATABASE_PASS=your_password \
  -e DATABASE_NAME=teslamate \
  ghcr.io/6547709/tesla-trail-map:1.6.6
```

> Docker picks the right layer automatically — amd64 hosts get amd64,
> Apple Silicon / Raspberry Pi get arm64.

#### Build locally

```bash
docker build -t tesla-trail-map:1.6.6 .
docker run -d --name tesla-trail-map -p 8080:8080 \
  -e DATABASE_HOST=host.docker.internal \
  -e DATABASE_PORT=5432 \
  -e DATABASE_USER=teslamate \
  -e DATABASE_PASS=your_password \
  -e DATABASE_NAME=teslamate \
  tesla-trail-map:1.6.6
```

#### Build multi-arch locally (optional)

```bash
docker buildx create --name multi --use 2>/dev/null || docker buildx use multi
docker buildx build --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=v1.6.6 -t tesla-trail-map:1.6.6 --load .
```

---

### Docker Compose (recommended for long-running setups)

Save the file below as `docker-compose.yml`, then `docker compose up -d`.

#### Mode A — standalone (connects to an existing TeslaMate DB)

```yaml
# docker-compose.yml
services:
  tesla-trail-map:
    image: ghcr.io/6547709/tesla-trail-map:1.6.6
    container_name: tesla-trail-map
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      # ───── required (no defaults — startup aborts if any is missing) ─────
      DATABASE_HOST: 192.168.66.200      # your TeslaMate PG host
      DATABASE_PORT: 54320               # default exposed port
      DATABASE_USER: teslamate
      DATABASE_PASS: ${DATABASE_PASS}    # injected from .env, never hard-coded
      DATABASE_NAME: teslamate
      # ───── optional ─────
      DATABASE_SSLMODE: disable
      TZ: Asia/Shanghai
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://127.0.0.1:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s
```

Put secrets in a sibling `.env` file (**add it to `.gitignore`**):

```bash
# .env
DATABASE_PASS=your_real_password
```

Operate:
```bash
docker compose pull            # pull latest image
docker compose up -d           # start in background
docker compose logs -f         # tail logs
docker compose ps              # show health
docker compose down            # stop & remove
```

#### Mode B — full TeslaMate stack in one compose

If you don't have TeslaMate running yet, this brings up TeslaMate + Postgres + Grafana + this app together:

```yaml
# docker-compose.yml — full TeslaMate stack + Trail Map
services:
  teslamate:
    image: teslamate/teslamate:latest
    restart: unless-stopped
    environment:
      ENCRYPTION_KEY: ${TESLAMATE_ENCRYPTION_KEY}
      DATABASE_USER: teslamate
      DATABASE_PASS: ${DATABASE_PASS}
      DATABASE_NAME: teslamate
      DATABASE_HOST: database
      MQTT_HOST: mosquitto
    ports:
      - "4000:4000"
    cap_drop: [all]

  database:
    image: postgres:17
    restart: unless-stopped
    environment:
      POSTGRES_USER: teslamate
      POSTGRES_PASSWORD: ${DATABASE_PASS}
      POSTGRES_DB: teslamate
    volumes:
      - teslamate-db:/var/lib/postgresql/data

  grafana:
    image: teslamate/grafana:latest
    restart: unless-stopped
    environment:
      DATABASE_USER: teslamate
      DATABASE_PASS: ${DATABASE_PASS}
      DATABASE_NAME: teslamate
      DATABASE_HOST: database
    ports:
      - "3000:3000"
    volumes:
      - teslamate-grafana-data:/var/lib/grafana

  mosquitto:
    image: eclipse-mosquitto:2
    restart: unless-stopped
    command: mosquitto -c /mosquitto-no-auth.conf
    volumes:
      - mosquitto-conf:/mosquitto/config
      - mosquitto-data:/mosquitto/data

  # ★ Trail Map (this project) — connects directly to the database service
  tesla-trail-map:
    image: ghcr.io/6547709/tesla-trail-map:1.6.6
    container_name: tesla-trail-map
    restart: unless-stopped
    depends_on:
      - database
    ports:
      - "8080:8080"
    environment:
      DATABASE_HOST: database          # docker network DNS, no IP needed
      DATABASE_PORT: 5432              # PG container's internal port (not 54320)
      DATABASE_USER: teslamate
      DATABASE_PASS: ${DATABASE_PASS}
      DATABASE_NAME: teslamate
      DATABASE_SSLMODE: disable
      TZ: Asia/Shanghai
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://127.0.0.1:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s

volumes:
  teslamate-db:
  teslamate-grafana-data:
  mosquitto-conf:
  mosquitto-data:
```

Matching `.env`:
```bash
# .env
DATABASE_PASS=use_a_strong_one
TESLAMATE_ENCRYPTION_KEY=use_another_strong_one
```

> ⚠️ **Mode A vs Mode B**
> - Mode A connects to an **external** TeslaMate DB — `DATABASE_HOST` is an IP/hostname, `DATABASE_PORT` is usually `54320` (TeslaMate's host-exposed port).
> - Mode B lives in the **same compose network** — `DATABASE_HOST` is the service name `database`, `DATABASE_PORT` is PG's **internal** `5432`.

#### Upgrading

```bash
docker compose pull tesla-trail-map
docker compose up -d tesla-trail-map
```

The container ships with a `HEALTHCHECK` calling `/health` every 30s; `docker ps` will show `(healthy)` once steady.

---

## ⚙️ Configuration

### Required (no defaults — startup aborts if any is missing)

| Variable | Required | Notes |
|---|---|---|
| `DATABASE_HOST` | ✅ | TeslaMate PG host (IP or hostname) |
| `DATABASE_USER` | ✅ | PG user |
| `DATABASE_PASS` | ✅ | PG password — **never** bake into images or commit to git |
| `DATABASE_NAME` | ✅ | DB name |

> Since v1.6, the four core connection parameters have **no hard-coded defaults**. Missing any of them aborts startup with an explicit error — preventing accidental connections to the wrong environment.

### Optional (safe defaults)

| Variable | Default | Notes |
|---|---|---|
| `DATABASE_PORT` | `5432` | PG port |
| `DATABASE_SSLMODE` | `disable` | `disable / require / verify-full` |

### Connection pool

| Variable | Default | Notes |
|---|---|---|
| `DATABASE_MAX_CONNS` | `max(20, 4×CPU)` | hard cap |
| `DATABASE_MIN_CONNS` | `max(4, CPU)` | warm pool |
| `DATABASE_MAX_LIFETIME` | `1h` | |
| `DATABASE_LIFETIME_JITTER` | `5m` | spreads reconnects |
| `DATABASE_MAX_IDLE` | `30m` | |
| `DATABASE_HEALTHCHECK_PERIOD` | `30s` | background health probe |
| `DATABASE_STMT_CACHE` | `256` | prepared-statement cache size |
| `DATABASE_DESC_CACHE` | `256` | row-description cache size |

### Postgres session safeguards

| Variable | Default |
|---|---|
| `DATABASE_STMT_TIMEOUT_MS` | `15000` |
| `DATABASE_IDLE_TX_TIMEOUT_MS` | `30000` |
| `DATABASE_LOCK_TIMEOUT_MS` | `5000` |
| `DATABASE_TCP_KEEPALIVES_IDLE` | `30` (s) |

### Server

| Variable | Default | Notes |
|---|---|---|
| `PORT` | `8080` | |
| `GIN_MODE` | `release` | `release / debug / test` |
| `TRUSTED_PROXIES` | _(empty)_ | comma-separated IP list |

---

## 🌐 HTTP API

### `GET /`
Serves the SPA `map.html`.

### `GET /cars`
Lists every vehicle for the current account.
```json
{
  "data": [
    { "id": 1, "name": "Model 3P", "model": "3",
      "marketing_name": "LR AWD Performance", "drive_count": 5665 }
  ],
  "count": 1
}
```
Display-name fallback: `cars.name → marketing_name → "Model "+model → "Car #id"`.

### `GET /trips`
Returns drives + downsampled position points within the requested window.

| Param | Required | Notes |
|---|---|---|
| `start_date` | ✅ | accepts `YYYY-MM-DD`, `YYYY-MM-DDTHH:MM[:SS]`, with optional timezone |
| `end_date` | ✅ | must be later than `start_date` |
| `car_id` | optional | omit = all vehicles |
| `max_points` | optional | per-drive cap; omit = no downsampling |

Hard window cap: **365 days**, otherwise 400.

| HTTP | Meaning |
|---|---|
| 200 | OK |
| 400 | Validation failure (`message` field describes it) |
| 504 | Query timed out — narrow the window or pick one car |
| 500 | Other internal error (DB internals are never leaked to the client) |

### `GET /health`
DB ping + pgxpool live stats:
```json
{
  "status": "healthy",
  "pool": { "total": 12, "acquired": 0, "idle": 12, "max": 40,
            "acquire_count": 67, "empty_acquire": 0, "...": "..." }
}
```
`empty_acquire > 0` ⇒ pool was saturated at some point — consider raising `DATABASE_MAX_CONNS`.

### `GET /version`
```json
{
  "name": "tesla-trail-map",
  "version": "1.6.6",
  "latest_version": "1.6.6",
  "is_latest": true,
  "go": "go1.25.x"
}
```
The frontend hits this endpoint to compare the running build against the latest known release: matched → green `v1.6 (latest)` pill in the header; mismatched → red badge prompting an update.

---

## 🏗️ Architecture Highlights

### Query performance (validated on a real 19.9M-row positions table)

| Scenario | Original | v1.5 rewrite + downsample |
|---|---|---|
| 7-day / all cars | 28.7s | **2.0s** (14× faster) |
| 7-day / car=1 | — | 1.07s |
| 7-day / car=2 | 60s (NestedLoop disaster) | **1.51s** (fixed via `AS MATERIALIZED`) |

Why it works:
1. **Move WHERE focus from `drive_id =` to `date BETWEEN`** so the BRIN(date) range scan kicks in;
2. **`AS MATERIALIZED` CTE** forces a hash semi-join and dodges the planner's miscalculated NestedLoop;
3. **Window-function downsampling**: `row_number() / count() OVER (PARTITION BY drive_id)` + modulo, keeps every ⌈total/N⌉-th point plus first/last.

### Connection pool
- Auto-sized by `NumCPU`;
- `QueryExecModeCacheStatement` for prepared-statement cache;
- DSN-level `application_name`, `statement_timeout`, `lock_timeout`, `idle_in_transaction_session_timeout`, `tcp_user_timeout`;
- Startup warm-up to MinConns;
- `MaxConnLifetimeJitter=5m` to spread reconnect storms.

---

## 🔒 Production hardening checklist

1. **Mandatory env vars** — never ship with the default `DATABASE_PASS=secret`.
2. **Tighten CORS** — currently `*`; restrict to your front-end origin.
3. **Add auth** — wrap behind Nginx + Basic Auth or an OAuth proxy.
4. **Rate-limit** large-window requests at the reverse proxy.
5. **TLS** termination at the reverse proxy.

---

## 📁 Layout

```
.
├── main.go              # HTTP routes + DB pool + SQL
├── map.html             # SPA frontend
├── go.mod / go.sum
├── Dockerfile           # multi-stage build
├── README.md            # 中文 default
├── README_EN.md         # this file
├── CHANGELOG.md         # bilingual release notes
└── .gitignore
```

---

## 🤝 Contributing
Issues and PRs welcome. Open an Issue first for non-trivial features.

## 📄 License
No license declared yet; consider MIT or Apache-2.0 if you plan to open-source.

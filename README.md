# TeslaMate Trail Map

[English](./README_EN.md) · **简体中文**

> 配套 [TeslaMate](https://github.com/teslamate-org/teslamate) 使用的车辆行驶轨迹回放工具。
> 后端 Go + Gin + pgx，前端 Leaflet + leaflet.motion。
> 当前版本：**v1.6.2** · 见 [CHANGELOG](./CHANGELOG.md)

---

## ✨ 主要功能

- 🚗 **多车切换**：自动从数据库读取你的车辆列表，按车辆**名字**切换（不暴露内部 car_id）。
- 🗺️ **轨迹回放**：在 Leaflet + OpenStreetMap 上沿历史路径动画播放，带 Tesla 风格小车标记。
- ⏱️ **时间窗查询**：起止时间 + 0.5x / 1x / 3x / 自定义速度。
- 🧠 **智能速度推荐**：根据驾驶时长自动建议播放倍速（≥3h→3x，1-3h→1x，短而密集→0.5x）。
- 🚀 **服务端抽稀**：默认 `max_points=2000` / 车，单趟 4 万点 → 2 千点，动画 60fps，不卡。
- 🛡️ **强健性**：参数严格校验、超时友好提示、数据库连接池可观测、AbortController 防 race。
- 📦 **Docker 一键部署**：多阶段构建，运行镜像基于 alpine。

---

## 📋 系统要求

| 组件 | 版本 |
|---|---|
| Go | 1.25+（构建） |
| PostgreSQL | 12+（运行时；已在 PG 18.3 上验证） |
| TeslaMate | 任意近期版本（提供 `cars` / `drives` / `positions` 表） |

---

## 🚀 快速开始

### 1. 直接用 Go 运行

```bash
git clone https://github.com/6547709/tesla-trail-map.git
cd tesla-trail-map

# 必填：数据库连接信息（生产环境务必通过环境变量传，不要写死）
export DATABASE_HOST=192.168.66.200
export DATABASE_PORT=54320
export DATABASE_USER=teslamate
export DATABASE_PASS=your_password
export DATABASE_NAME=teslamate

go run .
```

打开 http://localhost:8080。

### 2. Docker 部署

#### 直接拉官方镜像（推荐）

每次发 `vX.Y` tag，GitHub Actions 自动构建并推送 `linux/amd64` + `linux/arm64` 多架构镜像到 GHCR：

```bash
docker pull ghcr.io/6547709/tesla-trail-map:1.6.2
# 也可以用 :latest（最新 tag 发布时同步更新）
docker pull ghcr.io/6547709/tesla-trail-map:latest

docker run -d --name tesla-trail-map -p 8080:8080 \
  -e DATABASE_HOST=host.docker.internal \
  -e DATABASE_PORT=5432 \
  -e DATABASE_USER=teslamate \
  -e DATABASE_PASS=your_password \
  -e DATABASE_NAME=teslamate \
  ghcr.io/6547709/tesla-trail-map:1.6.2
```

> Docker 会自动选你机器架构对应的层（x86_64 主机拿 amd64，Apple Silicon / 树莓派拿 arm64）。

#### 自己构建

```bash
docker build -t tesla-trail-map:1.6.2 .
docker run -d --name tesla-trail-map -p 8080:8080 \
  -e DATABASE_HOST=host.docker.internal \
  -e DATABASE_PORT=5432 \
  -e DATABASE_USER=teslamate \
  -e DATABASE_PASS=your_password \
  -e DATABASE_NAME=teslamate \
  tesla-trail-map:1.6.2
```

#### 本地多架构构建（可选）

```bash
docker buildx create --name multi --use 2>/dev/null || docker buildx use multi
docker buildx build --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=v1.6.2 -t tesla-trail-map:1.6.2 --load .
```

---

### 3. Docker Compose 部署（推荐用于长期运行）

把以下内容保存为 `docker-compose.yml`，然后 `docker compose up -d` 一键起。

#### 方式 A：独立部署（应用已自带连接到现有 TeslaMate 数据库）

```yaml
# docker-compose.yml
services:
  tesla-trail-map:
    image: ghcr.io/6547709/tesla-trail-map:1.6.2
    container_name: tesla-trail-map
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      # ───── 必填（无默认值，少一个直接退出）─────
      DATABASE_HOST: 192.168.66.200      # 改成你的 TeslaMate PG 地址
      DATABASE_PORT: 54320               # TeslaMate 默认对外 54320
      DATABASE_USER: teslamate
      DATABASE_PASS: ${DATABASE_PASS}    # 走 .env 注入，不要明文写进 yml
      DATABASE_NAME: teslamate
      # ───── 可选 ─────
      DATABASE_SSLMODE: disable
      TZ: Asia/Shanghai
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://127.0.0.1:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s
```

同目录下放一个 `.env` 文件保存敏感信息（**记得加进 `.gitignore`**）：

```bash
# .env
DATABASE_PASS=your_real_password
```

启动：
```bash
docker compose pull            # 拉最新镜像
docker compose up -d           # 后台运行
docker compose logs -f         # 看日志
docker compose ps              # 看健康状态
docker compose down            # 停止 + 删除容器
```

#### 方式 B：与 TeslaMate 整合（共用同一个 compose）

如果你还没部署 TeslaMate，可以一份 yml 把 TeslaMate + Postgres + Grafana + 本应用全起来：

```yaml
# docker-compose.yml — 完整 TeslaMate 套件 + Trail Map
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

  # ★ Trail Map（本项目）— 通过容器网络直连 database 服务
  tesla-trail-map:
    image: ghcr.io/6547709/tesla-trail-map:1.6.2
    container_name: tesla-trail-map
    restart: unless-stopped
    depends_on:
      - database
    ports:
      - "8080:8080"
    environment:
      DATABASE_HOST: database          # docker 内网 DNS，无需 IP
      DATABASE_PORT: 5432              # PG 容器内端口（不是 54320）
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

对应的 `.env`：
```bash
# .env
DATABASE_PASS=use_a_strong_one
TESLAMATE_ENCRYPTION_KEY=use_another_strong_one
```

> ⚠️ **方式 A vs 方式 B 的关键区别**
> - 方式 A 连**外部已有**的 TeslaMate 库，`DATABASE_HOST` 用 IP/域名、`DATABASE_PORT` 通常是 `54320`（TeslaMate 暴露到宿主机的端口）。
> - 方式 B 在**同一 compose 网络**里，`DATABASE_HOST` 用 service 名 `database`、`DATABASE_PORT` 是 PG 容器**内**端口 `5432`。

#### 升级镜像

```bash
docker compose pull tesla-trail-map
docker compose up -d tesla-trail-map
```

容器内置 `HEALTHCHECK` 会调 `/health`，30 秒一次；`docker ps` 状态列会显示 `(healthy)`。

---

## ⚙️ 完整配置项

### 必填（无默认值，缺任一启动失败）

| 环境变量 | 必填 | 说明 |
|---|---|---|
| `DATABASE_HOST` | ✅ | TeslaMate PG 主机（IP 或域名） |
| `DATABASE_USER` | ✅ | PG 用户名 |
| `DATABASE_PASS` | ✅ | PG 密码（**绝不能**写进镜像或 git） |
| `DATABASE_NAME` | ✅ | 数据库名 |

> 自 v1.6 起，4 个连接核心参数已**完全移除硬编码默认值**。任一缺失服务会立即退出并打印缺失项，避免连错环境造成数据泄露。

### 可选（带安全默认值）

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `DATABASE_PORT` | `5432` | PG 端口 |
| `DATABASE_SSLMODE` | `disable` | `disable / require / verify-full` |

### 数据库连接池

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `DATABASE_MAX_CONNS` | `max(20, 4×CPU)` | 最大连接数 |
| `DATABASE_MIN_CONNS` | `max(4, CPU)` | 最小常驻连接数 |
| `DATABASE_MAX_LIFETIME` | `1h` | 连接最长生命周期 |
| `DATABASE_LIFETIME_JITTER` | `5m` | 生命周期抖动（防惊群）|
| `DATABASE_MAX_IDLE` | `30m` | 空闲连接最长生存 |
| `DATABASE_HEALTHCHECK_PERIOD` | `30s` | 后台健康检查周期 |
| `DATABASE_STMT_CACHE` | `256` | 预编译语句缓存上限 |
| `DATABASE_DESC_CACHE` | `256` | 列描述符缓存上限 |

### PostgreSQL 会话级保护

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `DATABASE_STMT_TIMEOUT_MS` | `15000` | 单语句超时 ms |
| `DATABASE_IDLE_TX_TIMEOUT_MS` | `30000` | 空闲事务超时 ms |
| `DATABASE_LOCK_TIMEOUT_MS` | `5000` | 锁等待超时 ms |
| `DATABASE_TCP_KEEPALIVES_IDLE` | `30` | TCP keepalive 秒 |

### 服务

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `PORT` | `8080` | HTTP 监听端口 |
| `GIN_MODE` | `release` | `release / debug / test` |
| `TRUSTED_PROXIES` | _（空）_ | 反代 IP 列表，逗号分隔 |

---

## 🌐 HTTP API

### `GET /`
返回前端页面 `map.html`。

### `GET /cars`
列出当前账号下所有车辆。
```json
{
  "data": [
    {
      "id": 1,
      "name": "Model 3P",
      "model": "3",
      "marketing_name": "LR AWD Performance",
      "drive_count": 5665
    }
  ],
  "count": 1
}
```
名字回退链：`cars.name → marketing_name → "Model "+model → "Car #id"`。

### `GET /trips`
查询时间窗内的行驶轨迹。

| 参数 | 是否必填 | 说明 |
|---|---|---|
| `start_date` | ✅ | 起始时间。支持 `YYYY-MM-DD`、`YYYY-MM-DDTHH:MM`、`YYYY-MM-DDTHH:MM:SS`、含时区。|
| `end_date` | ✅ | 结束时间，必须晚于 start_date |
| `car_id` | 否 | 车辆 ID，缺省 = 全部车辆 |
| `max_points` | 否 | 服务端抽稀：每辆车最多保留 N 个点（含首末），缺省 = 不抽稀 |

时间窗 **上限 365 天**，超过返回 400。

错误码：
| HTTP | 含义 |
|---|---|
| 200 | 成功 |
| 400 | 参数错误（信息在 `message` 字段） |
| 504 | 查询超时（建议缩小时间窗或选单辆车） |
| 500 | 其他内部错误（DB 内部错误**不会**透传给客户端） |

### `GET /health`
返回数据库连接状态 + pgxpool 统计：
```json
{
  "status": "healthy",
  "pool": { "total": 12, "acquired": 0, "idle": 12, "max": 40,
            "acquire_count": 67, "empty_acquire": 0, ... }
}
```
`empty_acquire > 0` 说明池打满过，考虑扩容。

### `GET /version`
```json
{
  "name": "tesla-trail-map",
  "version": "1.6.2",
  "latest_version": "1.6.2",
  "is_latest": true,
  "go": "go1.25.x"
}
```
前端使用此接口比对当前版本与最新版本：相同则在 header 显示绿色 `v1.6 (latest)`，否则显示红色提示更新。

---

## 🏗️ 架构亮点

### SQL 性能（基于真实 19.9M 行 positions 表）

| 场景 | 原版 SQL | v1.5 改写 + 抽稀 |
|---|---|---|
| 7 天 / 全部车辆 | 28.7s | **2.0s**（14× 提速） |
| 7 天 / car=1 | — | 1.07s |
| 7 天 / car=2 | 60s（NestedLoop 灾难） | **1.51s**（`AS MATERIALIZED` 修复） |

关键改写：
1. **WHERE 重心从 `drive_id =` 移到 `date BETWEEN`**，让 BRIN(date) 范围扫起作用；
2. **CTE 加 `AS MATERIALIZED`**，强制 hash semi-join，绕开规划器的 NestedLoop 误判；
3. **窗口函数抽稀**：`row_number() / count() OVER (PARTITION BY drive_id)` + 模运算，每车保留 ⌈total/N⌉ 比例 + 首末。

### 连接池
- 池容量随 NumCPU 自适应；
- 启用 `QueryExecModeCacheStatement`（预编译语句缓存）；
- DSN 注入 `application_name`、`statement_timeout`、`lock_timeout` 等会话级保护；
- 启动期主动 warm-up 至 MinConns，避免首请求付握手代价；
- `MaxConnLifetimeJitter=5m` 防惊群重连。

---

## 🔒 安全建议（生产部署前）

1. **强制环境变量**：不要使用 `DATABASE_PASS=secret` 默认值；
2. **CORS 收紧**：当前 `Access-Control-Allow-Origin: *`，请按需限制；
3. **加鉴权**：建议套一层反代（Nginx + Basic Auth / OAuth Proxy）；
4. **限频**：建议在反代层加 rate limit，防止恶意大窗口请求耗尽连接池；
5. **TLS**：建议反代终结 HTTPS。

---

## 📁 仓库结构

```
.
├── main.go              # 后端：HTTP 路由 + DB 连接池 + SQL 查询
├── map.html             # 前端：单页 HTML + Leaflet
├── go.mod / go.sum      # Go 模块
├── Dockerfile           # 多阶段镜像构建
├── README.md            # 本文档（中文）
├── README_EN.md         # English version
├── CHANGELOG.md         # 版本变更记录（双语）
└── .gitignore
```

---

## 🤝 贡献

欢迎 PR！修 Bug、加功能、补文档都可以。
建议先开 Issue 讨论方向，避免重复造轮子。

## 📄 License

仓库当前未声明 License；按 GitHub 默认私有版权处理。如要开源，建议加 MIT 或 Apache-2.0。

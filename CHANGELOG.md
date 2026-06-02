# Changelog · 变更记录

All notable changes are documented here. 所有重要变更记录在此。
Format follows [Keep a Changelog](https://keepachangelog.com/) and Semantic Versioning.

---

## [v1.5] – 2026-06-02

### 🇨🇳 中文

#### 新增（Added）
- **多车切换器**：UI 顶部新增 Vehicle 段，按车辆名字（不是 ID）切换；副标题显示行驶次数，hover tooltip 显示市场名称。
- `GET /cars` 路由：返回当前账号下所有车辆，带 `name` / `model` / `marketing_name` / `drive_count`。
- `GET /version` 路由：返回构建版本与 Go 版本，便于部署校验。
- `/trips` 新增 `?car_id=N` 参数（缺省 = 全部车辆）。
- `/trips` 新增 `?max_points=N` 服务端抽稀，单趟最多 N 点 + 首末。
- 时间格式更宽松：支持 `YYYY-MM-DD` / `YYYY-MM-DDTHH:MM` / `YYYY-MM-DDTHH:MM:SS` / 带时区。
- 14 个 `DATABASE_*` 环境变量（池大小、生命周期、超时、缓存等）全量可调。
- `/health` 输出 pgxpool 完整指标（acquire_count、empty_acquire、new_conns 等）。
- 双语 README + CHANGELOG。

#### 修改（Changed）
- **SQL 大改写**：把 WHERE 重心从 `drive_id =` 转到 `date BETWEEN`，让 BRIN(date) 起作用；7 天窗口从 28.7s → **2.0s**（14× 提速）。
- **CTE 加 `AS MATERIALIZED`** 强制 hash semi-join，修复 car_id=2 时 60s NestedLoop 问题（→ 1.51s）。
- 池容量随 `NumCPU` 自适应，启用预编译语句缓存，DSN 注入 `application_name` 与会话级超时。
- 启动期 warm-up 至 MinConns，避免首请求付握手代价。
- `MaxConnLifetimeJitter=5m` 防惊群重连。
- 默认 gin `release` 模式，`SetTrustedProxies(nil)` 消除警告。
- 错误响应统一 `{"message":"..."}`，**不再透传**数据库错误 / SQLSTATE / 表名给客户端。
- 504 友好提示替代 500 暴露 timeout。
- 404 改为 JSON 而非纯文本。

#### 修复（Fixed）
- 🔴 `/trips` 无参数会拉全表打挂 PG → 现在 400 缺参提示。
- 🔴 错误信息透传 SQLSTATE / 表结构 → 现在仅 `message`，原 error 写日志。
- 🟡 非法日期 / start>end / car_id=-1 / car_id=abc / 窗口>365天 全部静默放行 → 现在 400 严格拒绝。
- 🟡 大窗口（≥30 天）触发 `statement_timeout` 后返回 500 → 现在 504 友好提示。
- 🟢 速度推荐 `userHasChangedSpeed` 一旦置 true 永不复位 → 每次查询起始重置。
- 🟢 切换车辆不会自动重画轨迹 → 现在自动 refetch（首次加载后）。
- 🟢 大请求无法中断 / 旧请求竞态 → 加 `AbortController`。

#### 性能数据（19.9M 行 positions 真实库实测）
| 场景 | v1.4 之前 | v1.5 |
|---|---|---|
| 7 天 / 全部车辆 | 28.7s | **2.0s** |
| 7 天 / car=1 | — | 1.07s |
| 7 天 / car=2 | 60s | **1.51s** |
| 单 drive 最大点数 | 45,746 | ≤ 2,000 |
| 7 天响应大小 | ~16 MB | ~5 MB |

---

### 🇬🇧 English

#### Added
- **Multi-vehicle switcher** in the header — pick by car **name**, not by raw `car_id`. Drive-count subscript and marketing-name tooltip included.
- `GET /cars` returns every vehicle with `name`, `model`, `marketing_name`, `drive_count`.
- `GET /version` for deploy-time verification.
- `/trips` accepts `?car_id=N` (omit = all cars).
- `/trips` accepts `?max_points=N` for server-side downsampling (keeps first/last + every ⌈total/N⌉-th point).
- Flexible time-input parser: `YYYY-MM-DD`, `YYYY-MM-DDTHH:MM[:SS]`, with optional timezone.
- 14 `DATABASE_*` env vars covering pool size, lifetimes, timeouts, caches.
- `/health` exposes the full pgxpool stat block.
- Bilingual README + CHANGELOG.

#### Changed
- **SQL rewrite**: shift WHERE focus from `drive_id =` to `date BETWEEN` so BRIN(date) wins; 7-day window 28.7s → **2.0s** (14× faster).
- **`AS MATERIALIZED` CTE** forces a hash semi-join and fixes the 60s NestedLoop for car=2 (→ 1.51s).
- Pool size auto-derives from `NumCPU`; prepared-statement cache enabled; DSN injects `application_name` and session-level timeouts.
- Startup warm-up to MinConns avoids the first-request handshake tax.
- `MaxConnLifetimeJitter=5m` to defeat reconnect stampedes.
- Default to `gin.ReleaseMode` and `SetTrustedProxies(nil)`.
- Error responses unified to `{"message":"..."}`; DB internals **no longer leak** to clients.
- 504 with friendly text replaces 500-with-SQLSTATE for timeouts.
- 404 returns JSON instead of plain text.

#### Fixed
- 🔴 `/trips` with no params used to attempt a full-table scan and time out at 15s — now 400 with a clear message.
- 🔴 SQLSTATE / table names used to leak to the client — sanitised.
- 🟡 invalid date / `start > end` / negative car_id / non-numeric car_id / window > 365 days — all silently accepted before, now strictly 400.
- 🟡 Large windows that hit `statement_timeout` returned 500 — now 504 with hint.
- 🟢 Speed-recommendation lock (`userHasChangedSpeed`) was sticky once set — reset on every query.
- 🟢 Switching vehicle did not auto-redraw — now auto-refetch after first load.
- 🟢 Long requests had no abort path — added `AbortController`.

#### Performance (measured against a real 19.9M-row positions DB)
| Scenario | pre-v1.4 | v1.5 |
|---|---|---|
| 7-day / all cars | 28.7s | **2.0s** |
| 7-day / car=1 | — | 1.07s |
| 7-day / car=2 | 60s | **1.51s** |
| Max points per drive | 45,746 | ≤ 2,000 |
| 7-day response size | ~16 MB | ~5 MB |

---

## [previous]

未记录的早期版本。Earlier versions are not tracked here.

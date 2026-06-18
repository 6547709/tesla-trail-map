# Changelog · 变更记录

All notable changes are documented here. 所有重要变更记录在此。
Format follows [Keep a Changelog](https://keepachangelog.com/) and Semantic Versioning.

---

## [v1.6.8] – 2026-06-18

### 🇨🇳 中文

#### 新增（Added）
- **🟣 多车并行播放（headline feature）**：默认 `All cars` 下，`/trips` 现在按 `car_id` 分组，每辆车一个独立 marker + 独立 polyline 链 + 独立 rAF tick，全部**同时**跑。car 1 = 🚗 emoji（macOS 红色），car 2 = 蓝色 🚙 SVG，car ≥ 3 = 🚗 emoji + 彩色环（紫/绿/橙/粉/青/橙，modulo 循环）。每车**独立计时**（基于自己的轨迹长度），不再被全局拉伸。
- **🟣 Trail 颜色按车匹配**：动画结束后，trail 颜色与该车的图标主色一致（car 1 红、car 2 蓝、car ≥ 3 环色），不再全红导致无法对号入座。
- **🟣 Per-car 完成 toast**：一辆车先跑完时，1.5s 短 toast 提示“`{carName} finished`”，而不是 marker 静默钉死等全局结束。

#### 修复（Fixed）
- **🔴 B-FIX（多车重构）leaflet.motion `motionStart` 在 polyline 还没上地图时静默 no-op**：v1.7 重构时把 `polylines[0].motionStart()` 和首次 `requestAnimationFrame` 挪到了 `layerGroup.addTo(map)` **之前**，leaflet.motion 内部 `if (!this._map) return;` 守卫吃掉调用，polyline 不动；rAF tick 此时读 `getLatLngs()` 拿到的是原始完整轨迹，`pts[pts.length-1]` = 末点 → marker 卡死在第一段 drive 的终点不动。**修复**：把 `motionStart()` 和首次 rAF 延迟到 `layerGroup.addTo(map)` 之后，并收集 `perCarStarters[]` 在循环结束后批量 kick off。
- **🔴 B-FIX（错误路径）fetch 失败时不清理旧动画**：`fetchAndDrawData` 的 catch 块只 toast + `setLoading(false)`，旧 `currentAnimation` / `currentPolyline` 仍指向之前成功画的图层组，下次 Show Trail 才被清。中间空档期旧 polylines + markers 残留地图上，像僵尸渲染。**修复**：抽 `teardownAnimation()` helper，catch 中先 teardown 再 toast。
- **🟡 S1 — CSS `transition: transform 200ms ease` 是方向翻转隐患**：rAF 16ms 跑一帧，方向变化可短于 200ms → CSS transition 永远完不成 → 图标卡顿。**修复**：删 transition（翻转是离散视觉不是 tween）。
- **🟡 S2 — `driveTracksByCar` 保留被过滤后为空的 bucket**：所有 drive 都是 singleton 被 `.filter(o => o.track.length >= 2)` 干掉的车的 bucket 仍存在，导致 `carCount` 偏高、toast 说 “across N cars” 但实际跑得少。**修复**：分组后清空数组。
- **🟡 S3 — 1-drive 车的 marker 卡在起点**：单 drive 时 `motion-ended` 立即触发，`activePoly = null`，rAF tick bail，marker 从未移动。**修复**：motion-ended “this car finished” 分支显式把 marker 钉到最后一点。
- **🟡 S4 — 首帧闪烁**：leaflet.motion 的首帧先于我们的 rAF 运行，第二次跑 tick 时 `getLatLngs()` 已推进几段 → marker 闪一下到中段再追上。**修复**：`isFirstTick` 短路，第一帧不读 head。
- **🟡 S5 — 每帧 `getElement().querySelector` 浪费**：wrap div 是一次性的。**修复**：marker addTo 后一次性查 `wrap`，tick 内直接复用。
- **🟡 S6 — per-car 日志缺失**：报 “car 3 太快” 时无法定位。**修复**：每车 carDuration 算完打 console。
- **🟡 S7 — `main.go` bind 与 scan 类型不一致**：`args = append(args, int16(carID))` vs `var carID int`。**修复**：统一 `int`。
- **🟡 S8 — `getSmartSpeed` 在多车模式有偏差**：原按 sum(所有 drive 时长) 推荐速度，2 辆 2h 车 sum=4h 推 5×，但 wall-clock = max=2h 应推 3×。**修复**：按 max(每车总时长) 算阈值。
- **🟡 S9 — 每车完成时静默**：（已计入上面 “新增” 的 per-car toast）。

#### 工程（NIT 清理）
- **🟢 N1**：删 `.car-marker-container` 上的 `will-change: transform`（hint 推的是 inner container，Leaflet 不动它，无效）。
- **🟢 N2**：删 `currentPolyline` 全局死代码（v1.6.5 起就没被赋值过）。
- **🟢 N3**：重构 `CAR_COLOR_PALETTE`（带头部 3 个 dummy null）为 `RING_COLORS` 平铺数组，模运算更可读；car 1/2 主色独立为 `CAR_BRAND_COLORS` 常量。
- **🟢 N5**：删未读的 `_driveCount` / `_carCount` 元数据。
- `Version` / `LatestVersion` → `1.6.8`；UI / README / docker-compose 同步。
- 后端 `getTripsWithPositions` SQL 增 `d.car_id` 到 CTE + SELECT；`IN (SELECT id FROM d)` 改 `JOIN d ON d.id = p.drive_id`（相同 hash-join plan，dbprobe 验证无回退）。**零 DB 写入**。

---

### 🇬🇧 English

#### Added
- **🟣 Multi-car parallel playback (headline)**: with default `All cars` selected, `/trips` is now grouped by `car_id` server-side via the new `car_id` field on each drive, and the frontend spawns one independent marker + polyline chain + rAF tick **per car**, all running concurrently. car 1 = 🚗 emoji (red on macOS), car 2 = blue 🚙 SVG, car ≥ 3 = 🚗 emoji on a coloured ring (purple/emerald/amber/pink/teal/orange, modulo-cycled). Each car times itself off its **own** track length — no global stretching.
- **🟣 Trail colour matches car brand**: when animation ends, each trail is rendered in its car's brand colour (car 1 red, car 2 blue, car ≥ 3 ring colour) so line ↔ icon remain paired after markers stop moving. Previously all trails were Tesla red.
- **🟣 Per-car finished toast**: when an early-finishing car (e.g. car 2's 5 short drives) stops while others keep animating, a 1.5s toast surfaces "`{carName} finished`" instead of leaving the marker silently frozen.

#### Fixed
- **🔴 B-FIX (multi-car) — leaflet.motion `motionStart` no-ops on detached polyline**: the v1.7 refactor moved `polylines[0].motionStart()` and the first `requestAnimationFrame` to BEFORE `layerGroup.addTo(map)`. leaflet.motion's internal `if (!this._map) return;` guard silently swallowed the call; the polyline never animated; the rAF tick then read `getLatLngs()` which returned the original full track, so `pts[pts.length-1]` = last point → marker was stuck at the end of the first drive. **Fix**: defer `motionStart()` and the first rAF until after `layerGroup.addTo(map)`; collect `perCarStarters[]` during the build loop and kick them off in a batch after addTo.
- **🔴 B-FIX (error path) — failed fetch left stale animation on the map**: `fetchAndDrawData`'s catch only toasted + reset loading; the previous `currentAnimation` / `currentPolyline` stayed on the map underneath the error toast, looking like a stale render until the next Show Trail. **Fix**: extract `teardownAnimation()` helper, call it in catch before toasting.
- **🟡 S1 — CSS `transition: transform 200ms ease` fought direction flips**: at 60fps (16ms/frame) the 200ms tween never completed when direction changed faster than that, producing visible stutter. **Fix**: removed the transition (a flip is a discrete visual change, not a tween).
- **🟡 S2 — `driveTracksByCar` kept empty buckets**: cars whose drives were all filtered out as singletons still occupied a bucket, inflating `carCount` and making the "across N cars" toast over-count. **Fix**: prune empty buckets after grouping.
- **🟡 S3 — 1-drive cars froze their marker at the start**: with only one polyline, `motion-ended` fired immediately, `activePoly` became null, rAF tick bailed — marker never moved. **Fix**: in motion-ended's "this car finished" branch, explicitly set marker to the last point of the last drive.
- **🟡 S4 — first-tick flash**: leaflet.motion's internal frame ran before our rAF tick, so `getLatLngs()` already reported a head 1–2 segments in. Reading it teleported the marker start → mid-track → catch up. **Fix**: `isFirstTick` short-circuit — first frame skips the head read and stays at the marker's initial position.
- **🟡 S5 — per-frame `getElement().querySelector` wasted work**: the `.car-marker-container` wrapper is created once with the divIcon. **Fix**: cache `wrap` once after addTo, reuse every frame.
- **🟡 S6 — missing per-car timing log**: when a user reports "car N too fast / too slow", there was no car-level breakdown in the console. **Fix**: per-car `console.log` after `carDuration` is computed.
- **🟡 S7 — `main.go` bind/scan type asymmetry**: `args = append(args, int16(carID))` vs `var carID int`. **Fix**: unify on `int`.
- **🟡 S8 — `getSmartSpeed` over-recommended in multi-car mode**: sum-based thresholds treated 2× 2h cars as 4h (→ 5×) when wall-clock is `max = 2h` (→ 3×). **Fix**: thresholds use `max(per-car totalDurationHours)` when >1 car.
- **🟡 S9 — per-car completion was silent** (counted under "Added" above).

#### Engineering (NIT cleanups)
- **🟢 N1**: removed `.car-marker-container`'s `will-change: transform` (hint was promoting a layer Leaflet doesn't actually transform-animate).
- **🟢 N2**: deleted `currentPolyline` global (unassigned since v1.6.5 — dead code).
- **🟢 N3**: refactored `CAR_COLOR_PALETTE` (3 dummy `null` head entries) into a flat `RING_COLORS` array; car 1/2 brand colours extracted to a separate `CAR_BRAND_COLORS` map. Modulo math is now readable.
- **🟢 N5**: dropped unused `_driveCount` / `_carCount` metadata on the layerGroup.
- `Version` / `LatestVersion` → `1.6.8`; UI / README / docker-compose synced.
- Backend `getTripsWithPositions`: added `d.car_id` to the CTE and outer SELECT; rewrote `IN (SELECT id FROM d)` → `JOIN d ON d.id = p.drive_id` (same hash-join plan, dbprobe-confirmed no regression). **Zero DB writes.**

---

## [v1.6.7] – 2026-06-17

### 🇨🇳 中文

#### 修改（Changed）
- **🟡 B34-A — 动画速度再次加快一倍**：v1.6.6 默认速度仍偏慢，用户再次反馈。`MS_PER_SEG` 从 `15ms` 降到 `7.5ms`（约 133 段/秒 @ 1×）；`MIN/MAX` 上下限同步缩半为 `1.5s / 45s`。默认 24h ~5000 点窗口在 1× 下现在 **~37.5s** 走完（v1.6.6 是 75s，v1.6.5 是 150s）。
- **🟢 B34-B — 速度档位改名为 1×/3×/5×**：旧的 `0.5× / 1× / 3×` 改为 `1× / 3× / 5×`。`0.5×` 档位删除——结合再次翻倍的速度，它实际已无意义。新 1× = 旧 0.5× 位置 + 2× 全局加速，对应“舒适慢放”场景；新 5× 是新增的最快档，适合“快速浏览整天行程”。
- **🟡 B34-C — 智能推荐档位同步上移**：根据新按钮布局，阈值改为 `≥3h → 5×`、`1–3h → 3×`、短而密集 → `1×`（原 `0.5×`）。toast 默认文本同步改为 “Switch to 5x speed?”。

#### 修复（Fixed）
- **🔴 B-F1 — NaN 坐标突破 `typeof` 校验**：原 `typeof pos.latitude === 'number'` 接受 NaN（`typeof NaN === 'number'` 为 true）。如果后端某行 GPS 数据异常产 `NaN`，`L.latLngBounds` / `carMarker.setLatLng` 会拿到 NaN，小车静默飞出地图。**修复**：改用 `Number.isFinite()` 拒绝 NaN / ±Infinity。覆盖 `latlngs` 校验（`map.html:1301-1307`）和 `driveTracks` 校验（`map.html:1359-1363`）两处。
- **🔴 B-F2 — `isValidDateTime` 正则放过非法日期**：原正则 `/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/` 接受 `2024-99-99T99:99`。`new Date(...)` 返回 Invalid Date，`NaN >= NaN` 校验放行，最后 `.toISOString()` 抛 `RangeError`，被 catch 吞掉后用户看到模糊的 "Failed to fetch data: RangeError"。**修复**：在正则通过后再 `new Date()` 一次，`!isNaN(d.getTime())` 才算合法。
- **🔴 B-F3 — Toast 定时器竞争**：原 `setTimeout(..., 3000)` 无取消机制。若用户在 2.9s 再次触发 Show Trail，旧定时器 0.1s 后把新 toast 立即关掉。**修复**：引入模块级 `toastTimer` / `recommendationTimer`，新消息先 `clearTimeout` 旧定时器。
- **🔴 B-F4 — `motion-ended` 监听器在新动画上意外触发**：leaflet.motion 内部 `setTimeout` 在 `map.removeLayer` 后仍可能再 fire 一次，旧闭包里的 `carMarker.setLatLng()` 和 `polylines[nextIdx].motionStart()` 会在孤儿 polyline 上跑，造成内存泄漏与状态错乱。**修复**：模块级 `animGeneration` 计数器，每次 `moveCarMarkerAndPath` 自增并 `const myGen = animGeneration`；旧回调 `myGen !== animGeneration` 直接 `return`，与 B28 rAF 修复同思路。

#### 工程
- `Version` / `LatestVersion` → `1.6.7`；UI / README / docker-compose 同步。
- **🟢 部署清理（仅 ops / 构建端，镜像运行时行为不变）**：
  - 新增 `.dockerignore`：build context 不再带 `.git/`、`*.md`、`.env*` 等冗余或敏感文件，runtime 镜像也不再残留 `.git/` 目录。
  - CI 删掉 `Set up QEMU` 步骤：Dockerfile 走 `GOOS/GOARCH` 交叉编译，QEMU 一直是 dead weight，每跑省 10–20s。
  - Dockerfile 删掉硬编码 `ENV TZ=Asia/Shanghai`：现在 `docker run -e TZ=...` / compose 的 TZ 能真正生效（不再被默认值盖住），`.env.example` 同步注释说明。
  - `docker-compose.yml` 中 `DATABASE_HOST` 从 `:-192.168.66.200`（作者家庭 IP）改为 `:?` 强制必填：忘配时启动直接失败，不再静默连到错误地址。

---

### 🇬🇧 English

#### Changed
- **🟡 B34-A — Animation speed doubled (again)**: v1.6.6 still felt sluggish, per user report. `MS_PER_SEG` lowered from `15ms` to `7.5ms` (≈133 segs/sec at 1×); `MIN/MAX` floor/ceiling halved to `1.5s / 45s`. The default 24h ~5000-point window now plays in **~37.5s** at 1× (was 75s in v1.6.6, 150s in v1.6.5).
- **🟢 B34-B — Speed presets relabelled to 1×/3×/5×**: old `0.5× / 1× / 3×` → `1× / 3× / 5×`. The `0.5×` preset is gone — combined with the new global 2× boost, it's no longer useful. New 1× is the "comfortable slow-mo" position; new 5× is a brand-new top gear for "skim a full day fast".
- **🟡 B34-C — Smart-recommendation thresholds shifted up** to match the new buttons: `≥3h → 5×`, `1–3h → 3×`, short-precise → `1×` (was `0.5×`). Toast default text updated to "Switch to 5x speed?".

#### Fixed
- **🔴 B-F1 — NaN coordinates slipped past `typeof` check**: `typeof NaN === 'number'` is true, so the old `typeof pos.latitude === 'number'` guard accepted NaN. A single NaN row from a bad GPS sample would corrupt `L.latLngBounds` and `carMarker.setLatLng`, sending the car silently off-map. **Fix**: use `Number.isFinite()` to reject NaN / ±Infinity. Applied at both the `latlngs` map (`map.html:1301-1307`) and the `driveTracks` map (`map.html:1359-1363`).
- **🔴 B-F2 — `isValidDateTime` regex accepted impossible dates**: the old `/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/` matched `2024-99-99T99:99`. `new Date(...)` returned Invalid Date, the `NaN >= NaN` comparison let it through, and `.toISOString()` later threw `RangeError` which the catch silently swallowed. **Fix**: after the regex passes, parse with `new Date()` and require `!isNaN(d.getTime())`.
- **🔴 B-F3 — Toast timer race**: the old `setTimeout(..., 3000)` had no cancel. If the user re-fired Show Trail at t=2.9s, the stale 3s timer fired at t=3.0s and wiped the brand-new toast. **Fix**: module-scoped `toastTimer` / `recommendationTimer`; new messages `clearTimeout` the old timer first.
- **🔴 B-F4 — `motion-ended` listener firing on the new animation**: leaflet.motion's internal `setTimeout` could fire one more time after `map.removeLayer`, causing the stale closure to call `carMarker.setLatLng()` and `polylines[nextIdx].motionStart()` on orphaned objects — a memory leak and state corruption (same class of issue as B28). **Fix**: module-scoped `animGeneration` counter, captured per-call as `myGen`; stale callbacks early-return when `myGen !== animGeneration`.

#### Engineering
- `Version` / `LatestVersion` → `1.6.7`; UI / README / docker-compose synced.
- **🟢 Deployment cleanup (ops / build only — no change to image runtime behaviour)**:
  - Added `.dockerignore`: the build context no longer carries `.git/`, `*.md`, `.env*`, or other secret / bulk files, and the runtime image no longer embeds a `.git/` directory.
  - Removed the `Set up QEMU` step from CI: the Dockerfile cross-compiles natively via `GOOS/GOARCH`, so QEMU was dead weight. Saves ~10–20 s per CI run.
  - Removed the hardcoded `ENV TZ=Asia/Shanghai` from the Dockerfile: `docker run -e TZ=...` and compose's `TZ:` now take effect cleanly (no longer shadowed by the baked-in default). `.env.example` updated with a comment showing common zones.
  - `docker-compose.yml` switched `DATABASE_HOST` from `:-192.168.66.200` (the maintainer's home IP) to `:?` (fail-fast required). Forgetting to set it now aborts startup instead of silently connecting to the wrong host.

---

## [v1.6.6] – 2026-06-04

### 🇨🇳 中文

#### 修改（Changed）
- **🟡 B33-A — 动画速度加快一倍**：用户反馈 v1.6.5 默认速度太慢。`MS_PER_SEG` 从 `30ms` 降到 `15ms`（约 67 段/秒 @ 1×）；`MIN/MAX` 上下限同步缩半为 `3s / 90s`。默认 24h ~5000 点窗口现在 ~75 秒走完（之前 150 秒）。
- **🟢 B33-B — 去掉跟随小车的蓝色定位针**：v1.6.5 在 `motion.polyline` 第 4 个 `markerOptions` 参数里传了 `{showMarker:false}` —— 但只要这个参数为 truthy，leaflet.motion 的 `_initializeMarker()` 就会创建一个 `__marker`（默认蓝色 Leaflet pin），并在第一帧 `addTo(map)`。修复：**完全省略第 4 个参数**，让 `markerOptions` 保持 `undefined`，从源头阻止 `__marker` 创建。地图上现在只剩唯一的 `carMarker`（我们自管的红色车图标）。

#### 工程
- `Version` / `LatestVersion` → `1.6.6`；UI / README / docker-compose 同步。

---

### 🇬🇧 English

#### Changed
- **🟡 B33-A — Animation speed doubled**: user reported v1.6.5 felt sluggish at the default speed. `MS_PER_SEG` lowered from `30ms` to `15ms` (≈67 segments/sec at 1×); `MIN/MAX` floor/ceiling halved to `3s / 90s`. The default 24h ~5000-point window now plays in ~75 seconds (was 150 seconds).
- **🟢 B33-B — Killed the stray blue Leaflet pin trailing the car**: v1.6.5 passed `{showMarker:false}` as the 4th `markerOptions` arg to `motion.polyline`. But leaflet.motion's `_initializeMarker()` creates a `__marker` whenever `markerOptions` is truthy — and `_drawMarker()` on the first animation frame `addTo(map)`s it (using the default blue Leaflet pin icon). Fix: **omit the 4th argument entirely**, leaving `markerOptions === undefined`, which prevents `__marker` from ever being constructed. The map now shows exactly one car icon (our self-managed `carMarker`).

#### Engineering
- `Version` / `LatestVersion` → `1.6.6`; UI / README / docker-compose synced.

---

## [v1.6.5] – 2026-06-04

### 🇨🇳 中文

#### 修复（Fixed）
- **🔴 B32 — v1.6.4 修好图标显示后又出现新问题：N 个静态小车，没有动画**：
  - 现象（用户截图）：默认参数 Show Trail 后，地图上 7 辆红色小车静静地分布在 7 个 drive 的起点上，**没有任何动画也没有轨迹绘制**。
  - 根因 1：`L.motion.seq` 把多个 child polyline `addTo(map)` 时，**每条 polyline 的 `__marker` 都被立即渲染**到自己的起点（即使 `auto:false`）。`showMarker:true` + `removeOnEnd:true` 这种组合在 seq 场景下行为不符合直觉。
  - 根因 2：`L.motion.seq` 不会自动推进 child polylines — 必须显式 `motionStart()`。我们之前以为 `auto:true` 会推进 seq 的子项，实际只是 motion-polyline 自身的 auto 字段被吃掉。
  - 修复（v1.6.5）：**完全抛弃 `motion.seq`**，改成自管模式：
    1. 每条 drive 是 `L.motion.polyline` + **`showMarker: false`**（关键！）— 完全不让 motion 自己产生 marker。
    2. 自己加一个 `L.marker(driveTracks[0][0], { icon: carIcon, ...zIndexOffset:1000 })` —— 全程**只有这一个**车图标。
    3. 用 `'motion-ended'` 事件链式触发：第 N 条结束 → `setLatLng` 把车跳到 N+1 的起点 → `motionStart()` N+1。
    4. rAF 循环每帧读 active polyline 的 `getLatLngs()` 末尾点，`carMarker.setLatLng(head)` 同步位置 + 计算 lng delta 决定左右翻转。
    5. 用 `L.layerGroup([...polylines, carMarker])` 一次性 `addTo(map)` / `removeLayer`，清理简洁。
  - 顺手修：事件名 `'motion-ended'`（不是 `motionended`，之前查源码确认）。

#### 工程
- `Version` / `LatestVersion` → `1.6.5`；UI / README / docker-compose 同步。

---

### 🇬🇧 English

#### Fixed
- **🔴 B32 — v1.6.4 fixed the missing-icon bug, but introduced a new one: N static cars, no animation**:
  - Symptom (user screenshot): after Show Trail with default params, the map showed 7 red car icons frozen at the start of each drive. **No animation, no trail being drawn.**
  - Root cause 1: when `L.motion.seq` adds child polylines to the map, **every child's `__marker` is rendered immediately at its starting point**, regardless of `auto:false`. The combination `showMarker:true` + `removeOnEnd:true` doesn't behave intuitively under seq.
  - Root cause 2: `L.motion.seq` does NOT auto-advance its children — they must be explicitly `motionStart()`-ed. We were assuming `auto:true` would chain the children, but the option is consumed by the motion-polyline itself.
  - Fix (v1.6.5): **drop `motion.seq` entirely**, run a manual chain:
    1. Each drive is an `L.motion.polyline` with **`showMarker: false`** (critical!) — no motion-managed markers at all.
    2. Add ONE plain `L.marker(driveTracks[0][0], { icon: carIcon, zIndexOffset: 1000, interactive: false })` — exactly one car visible at any time.
    3. Chain with `'motion-ended'` event: when polyline N finishes, `setLatLng` the car to drive N+1's start, then call `polylines[N+1].motionStart()`. The marker JUMPS — no fake connector polyline.
    4. rAF loop reads the active polyline's animated head via `getLatLngs()`, syncs `carMarker.setLatLng(head)` and computes longitude delta for the left/right flip.
    5. Wrap polylines + marker in an `L.layerGroup` so `map.removeLayer(group)` cleans everything up.
  - Also fixed: event name is `'motion-ended'` (not `motionended` — confirmed from minified source `L.Motion.Event.Ended = "motion-ended"`).

#### Engineering
- `Version` / `LatestVersion` → `1.6.5`; UI / README / docker-compose synced.

---

## [v1.6.4] – 2026-06-04

### 🇨🇳 中文

#### 修复（Fixed）
- **🔴 B31 — 车辆图标在 v1.6.3 之后消失**：v1.6.3 把"按时间戳排序 drives"的逻辑写在了 `moveCarMarkerAndPath()` 函数体里，但函数里引用了 `data` 变量——这个变量只存在于 `fetchAndDrawData()` 作用域。运行时抛 `ReferenceError: data is not defined`，被外层 `try/catch` 静默吞掉，导致整个 `L.motion.seq` 创建路径根本没执行 → 地图上没有任何车标。
  - 修复：把"按时间戳排序"前移到 `fetchAndDrawData()`（`data` 在那里是合法的），让 `moveCarMarkerAndPath` 只接收已经排好序的 `driveTracks` 干净参数。
  - 函数签名更明确，闭包不再泄漏 `data`。
  - 注释里同步说明 v1.6.3 这个 ReferenceError 的原因，避免下次再踩。

#### 工程
- `Version` / `LatestVersion` → `1.6.4`；UI / README / docker-compose 同步。

---

### 🇬🇧 English

#### Fixed
- **🔴 B31 — Car icon disappeared after v1.6.3**: v1.6.3 moved the "sort drives by timestamp" logic into `moveCarMarkerAndPath()`'s body, but that function then referenced `data` — a variable that only exists in `fetchAndDrawData()`'s scope. At runtime it threw `ReferenceError: data is not defined`, was silently swallowed by the surrounding `try/catch`, and the entire `L.motion.seq` setup never ran. Result: the trail painted but no car icon appeared on the map.
  - Fix: hoist the "sort drives by first-point timestamp" logic into `fetchAndDrawData()` (where `data` is valid). `moveCarMarkerAndPath` now receives a clean, already-sorted `driveTracks` parameter — no closure leaks.
  - Comments now document the v1.6.3 ReferenceError so future hands don't re-introduce it.

#### Engineering
- `Version` / `LatestVersion` → `1.6.4`; UI / README / docker-compose synced.

---

## [v1.6.3] – 2026-06-04

### 🇨🇳 中文

#### 修复（Fixed）
- **🔴 B30 — 多个 trip 之间被错画成跨城直线**：旧代码 `data.flatMap(d => d.positions)` 把 N 个 drive 的点合并成一个数组喂给 `L.motion.polyline`，导致每两个 drive 之间用一根直线连起来——视觉上小车像疯狂瞬移；用户截图里那条横跨成都市三个区的红色三角形，就是 drive A 终点到 drive B 起点的"假路径"。
  - 实测确认：默认 24h all-cars = 7 drives，6 处跨 drive 跳跃，每处跨度 7-10 km。
  - 修复：每个 drive 各自一条 `L.motion.polyline`，用 `L.motion.seq` 串接 → drive 之间断开，**不再画虚假直线**。
  - 同时按动画总预算按段数 **比例分配** 每段 duration（长 drive 占更多时间），保持平均速度恒定。
  - 顺手修：drives 在前端按"第一个点的时间戳"重新排序，避免 server 按 drive_id 返回时与真实时序不一致。
  - 每个非末段 polyline `removeOnEnd:true`，避免 7 辆静止小车散在地图上。
- **B29 (carryover)**：动画时长按内容缩放 `clamp(segs × 30ms, 5s, 180s)`，与 B30 协同。

#### 工程
- `Version` / `LatestVersion` → `1.6.3`；UI / README / docker-compose 同步。

---

### 🇬🇧 English

#### Fixed
- **🔴 B30 — Phantom inter-trip lines making the car appear to teleport**: previous code did `data.flatMap(d => d.positions)` and fed the result as one polyline. Adjacent drives end in city A and start in city B with no real path between — but a flat polyline connects them with a straight line, so the car visibly teleported across the map between trips. Confirmed against live data: default 24h all-cars window = 7 drives, 6 inter-drive jumps each 7-10 km wide.
  - Fix: each drive becomes its own `L.motion.polyline`; we sequence them with `L.motion.seq`. The seq advances to the next polyline by jumping the marker, **not** by drawing a connector — gaps between drives are no longer painted.
  - Per-drive duration is allocated proportionally to segment count, preserving roughly constant ground speed across the whole sequence.
  - Drives are now ALSO re-sorted client-side by their first-point timestamp before sequencing — server returns by `drive_id` which can be out of chronological order if trips were ingested late.
  - Non-final polylines use `removeOnEnd: true` so we don't accumulate 7 stationary cars scattered across the city.
- **B29 carryover**: per-content animation duration `clamp(segs × 30ms, 5s, 180s)` works alongside B30.

#### Engineering
- `Version` / `LatestVersion` → `1.6.3`; UI / README / docker-compose synced.

---

## [v1.6.2] – 2026-06-04

### 🇨🇳 中文

#### 修复（Fixed）
- **🔴 B29 — 默认参数下车辆"疯狂瞬移"**：旧代码 `finalDuration = 8000 / finalSpeed` 给整条 polyline 固定 8 秒，与点数无关。默认 24h "all cars" 窗口 = ~5000 个数据点 → 8000ms 走完意味着 **629 段/秒**，肉眼就是疯狂瞬移；smart-speed 自动选 3× 时更夸张（~1900 段/秒）。
  - 修复：动画时长按内容缩放：`baseMs = clamp(segments × 30ms, 5s, 180s)`、`finalDuration = baseMs / speed`。
  - 实测：默认 24h 窗口 5033 点 → **150.9s @ 1×（33 段/秒）**，视觉舒适。
  - Toast 同步显示 `points · 总时长 · 速度倍率`，方便排查。

#### 工程
- `Version` / `LatestVersion` → `1.6.2`；UI 5 处版本字面 + README/docker-compose 镜像 tag 同步。

---

### 🇬🇧 English

#### Fixed
- **🔴 B29 — "Frantic teleporting" on default-window playback**: previous code did `finalDuration = 8000 / finalSpeed` — a fixed 8-second total animation no matter how many points were on the polyline. The default 24h "all cars" window has ~5000 points after server-side downsampling, so 8000ms total = **629 segments per second** of motion. Smart-speed picking 3× compounded it (~1900 seg/s). Looked like the car was teleporting across the map.
  - Fix: duration now scales with content. `baseMs = clamp(segments × 30ms, 5s, 180s)`; `finalDuration = baseMs / speed`.
  - Measured: 24h default window with 5033 points → **150.9s at 1× (33 segs/sec)** — comfortable car-cam feel.
  - Toast now shows `points · total duration · speed` so users can sanity-check.

#### Engineering
- `Version` / `LatestVersion` → `1.6.2`; UI version literals (5 spots) and README / docker-compose image tags bumped.

---

## [v1.6.1] – 2026-06-04

### 🇨🇳 中文

#### 修复（Fixed）
- **B28 — 前端 rAF 泄露**：`moveCarMarkerAndPath()` 内部 `let rafId` 是函数局部变量，每次重新点 *Show Trail* 或切车都新建闭包但未取消上一次。表现：旧 loop 在 stale marker 引用上 tick 1-2 帧（无报错但浪费 CPU + 制造日志噪音）。
  - 修复：`directionRafId` 提升到 script 级；进入函数先 `cancelAnimationFrame` + 置 null；`on('end')` 同步清理。
  - 验证：连续 30 次 Show Trail，DevTools Performance 始终只有一条 rAF loop 在跑。

#### 工程
- `Version` / `LatestVersion` 常量同步升至 `1.6.1`；`/version` 端点自动反映；UI header pill / status panel "Version" 行同步。
- README / docker-compose.yml 中 docker tag 全部升至 `:1.6.1`。

---

### 🇬🇧 English

#### Fixed
- **B28 — Frontend rAF leak**: `moveCarMarkerAndPath()` declared `let rafId` as a function-local variable. Each subsequent *Show Trail* click or vehicle switch spawned a fresh closure and a fresh requestAnimationFrame loop without ever cancelling the previous one. The stale loop ticked 1-2 frames against a dead marker reference before short-circuiting — no error, but wasted CPU and added log noise.
  - Fix: Hoisted `directionRafId` to script scope; cancel + null it on entry to `moveCarMarkerAndPath()`; `on('end')` cancels and nulls again.
  - Verified: 30 consecutive *Show Trail* clicks → DevTools Performance shows exactly one rAF loop alive at any time.

#### Engineering
- `Version` / `LatestVersion` constants bumped to `1.6.1` in lockstep; `/version` reflects automatically; UI header pill and status-panel Version row stay in sync.
- README and docker-compose.yml image tags bumped to `:1.6.1`.

---

## [v1.6] – 2026-06-03

### 🇨🇳 中文

#### 🔒 安全 / 配置（v1.6 patch）
- **移除 DB 连接参数硬编码**：`DATABASE_HOST` / `DATABASE_USER` / `DATABASE_PASS` / `DATABASE_NAME` 不再有默认值，缺任一项服务**立即退出**并打印缺失项列表。杜绝新部署意外连接到旧环境（例如默认 `192.168.10.200` / `secret` 之类的开发凭据残留）。
- 仅保留通用安全默认值：`DATABASE_PORT=5432`、`DATABASE_SSLMODE=disable`。

#### 修复 / 加固（v1.6 后续 patch，同 tag）
- **🔴 B17 高危**：`/trips?max_points=0&car_id=N>0` 触发 PG `SQLSTATE 42P18`（占位符 `$3` 类型推断失败）→ 修复：V1 / V2 分支独立计算 carCond 占位符索引（V1 用 `$3`、V2 用 `$4`）。
- **🟡 B18**：`/cars` 失败时透传 DB error → 改成与 `/trips` 一致的 `{"message":"Failed to fetch vehicle list"}`。
- **🟢 B19**：客户端中断打 ERROR 日志噪声 → 降级为 INFO，并对 `/trips` handler 返回 `499 client closed request`（不再写 body）。
- **🟢 B20**：所有路由加 Cache-Control。`/version` `public,max-age=300`、`/cars` `private,max-age=60`、`/trips` `/health` `no-store`。
- **🟢 B21**：新增 stdlib 实现的 gzip 中间件（无新依赖），`Accept-Encoding: gzip` 时启用，**压缩率 ~88%**（1.6 MB → 186 KB）。`gzip.BestSpeed` + `sync.Pool` writer 复用。
- **🟢 B22**：所有 GET 路由同步注册 HEAD（gin 不自动），让 `curl -I`、K8s 健康探针、uptime 监控都能正常工作。
- **🟢 B23**：新增 graceful shutdown — 监听 SIGTERM/SIGINT，给进行中请求 15 秒收尾，然后 `pool.Close()` 干净退出。Docker/K8s 终止时不再丢请求。
- **🟢 B27**：慢查询阈值 500ms → 1500ms（避开真实库 BRIN-only schema 在 600-1200ms 的常态）。
- HTTP server 加 `ReadHeaderTimeout: 5s` 防 Slowloris。

### 🇬🇧 English

#### 🔒 Security / config (v1.6 patch)
- **Removed hard-coded DB connection defaults**. `DATABASE_HOST`, `DATABASE_USER`, `DATABASE_PASS`, `DATABASE_NAME` are now mandatory — startup aborts immediately listing the missing variable(s). This prevents a fresh deployment from accidentally inheriting prior-environment defaults (e.g. legacy dev creds like `secret` / `192.168.10.200`).
- Only universally-safe defaults remain: `DATABASE_PORT=5432`, `DATABASE_SSLMODE=disable`.

#### Fixed / Hardened (v1.6 patch, same tag)
- **🔴 B17 critical**: `/trips?max_points=0&car_id=N>0` triggered PG `SQLSTATE 42P18` (parameter `$3` type inference failed) → fix: V1 and V2 branches now compute the carCond placeholder index independently (V1 uses `$3`, V2 uses `$4`).
- **🟡 B18**: `/cars` leaked the raw DB error on failure → now returns `{"message":"Failed to fetch vehicle list"}` consistent with `/trips`.
- **🟢 B19**: Client cancellations were spamming ERROR logs → downgraded to INFO; `/trips` handler now returns `499 client closed request` with no body.
- **🟢 B20**: Cache-Control headers everywhere. `/version` `public,max-age=300`, `/cars` `private,max-age=60`, `/trips` and `/health` `no-store`.
- **🟢 B21**: Added a stdlib gzip middleware (no new deps). Activates on `Accept-Encoding: gzip`; **~88% compression** (1.6 MB → 186 KB) on a typical 7-day window. `gzip.BestSpeed` + `sync.Pool` writer reuse.
- **🟢 B22**: Every GET route is mirrored as HEAD (gin doesn't auto-mirror) so `curl -I`, K8s probes, and uptime monitors all return 200.
- **🟢 B23**: Graceful shutdown — listen on SIGTERM/SIGINT, drain in-flight requests for up to 15 seconds, then `pool.Close()` cleanly. Docker/K8s terminations no longer drop requests.
- **🟢 B27**: Slow-query threshold 500ms → 1500ms (real-world BRIN-only schema sits at 600-1200ms baseline).
- HTTP server now sets `ReadHeaderTimeout: 5s` (Slowloris guard).

### 🇨🇳 中文 (v1.6 主版本)

#### 修改（Changed）
- **🚗 全新车辆动画图标**：替换为系统原生 emoji 🚗，跨平台原生渲染
  - macOS / iOS → Apple Color Emoji（圆润红车）
  - Windows → Segoe UI Emoji（扁平风）
  - Linux / Android → Noto Color Emoji（Twemoji 风）
  - 44×44 SVG 容器，包裹 `<text>🚗</text>` + 旋转 -90° 让 emoji 默认朝上；leaflet.motion 仍按轨迹切线旋转
  - 删除上版手绘 SVG 共 113 行代码（净 +23/-118）
- 头部完全去掉之前残缺的 Tesla 内联 SVG logo，标题更克制干净。

#### UI 重设计（视觉系统升级）
- 引入完整 **设计 token 系统**：ink 灰阶（50→900）、语义色、4 级 elevation 阴影、统一 radius / transition tokens。
- 顶栏改为双行：标题 + 副标题 + 右推版本徽章；标题字号 22 → 18px。
- 控件区从 `flex-wrap` 改成 **响应式 12 列 grid**（1100px → 2 列、640px → 1 列）。
- 输入 / 分段控件统一 38px 高度、focus ring、hover tint；车辆数量徽章用 pill。
- 状态浮窗：右上 → 左下角；改为 2 列 grid 对齐 label/value；加 pulsing 状态点；数字使用 tabular-num 字体。
- Toast：圆角胶囊 + slide-in；smart-recommendation 卡片重做。
- Leaflet zoom 与 attribution 自定义统一圆角 + 阴影。

#### 修复 / 加固
- 头部 Tesla SVG path 数据本身损坏，渲染只剩残形 → 直接删除。
- 车辆切换器 `escapeHtml()` 防 XSS。

#### 工程
- map.html 整体净瘦身（v1.5 SVG 字符串体积大）。
- 所有 17 个关键 DOM id 保留，JS 改动仅限 carSvg 常量。
- `Version` / `LatestVersion` 常量同步升至 1.6；`/version` 端点自动反映。

---

### 🇬🇧 English

#### Changed
- **🚗 New car marker** — replaced the hand-drawn Tesla SVG with the system-native 🚗 emoji, rendered natively per platform:
  - macOS / iOS → Apple Color Emoji (rounded red)
  - Windows → Segoe UI Emoji (flat)
  - Linux / Android → Noto Color Emoji (Twemoji-like)
  - 44×44 SVG wraps `<text>🚗</text>` rotated -90° so it points up at rest; leaflet.motion still rotates along path tangent.
  - Net diff: +23 / -118 lines (deleted the v1.5 hand-drawn SVG block).
- Removed the broken inline Tesla logo from the header — the original SVG path data was malformed and only rendered a partial shape.

#### UI redesign
- Introduced a real **design-token system** (ink scale 50→900, semantic colors, 4-tier elevation, unified radius / transition).
- Header → two-row layout: title + subtitle + right-aligned version pill; title 22 → 18px.
- Controls: from wrap-flex to **responsive 12-col grid** (2-col at <1100px, 1-col on phones).
- Inputs and segmented controls share a 38px height, focus ring, hover tint; car-switch buttons get pill counters.
- Floating status panel: top-right → bottom-left, 2-col grid alignment, pulsing status dot, tabular-num figures.
- Toasts: pill-shaped, slide-in; smart-recommendation card rewritten.
- Leaflet zoom + attribution restyled to match the card system.

#### Fixed / Hardened
- Broken Tesla SVG logo removed from header.
- Car switcher names go through `escapeHtml()` (XSS guard).

#### Engineering
- map.html slimmed (v1.5's hand-drawn SVG was sizeable).
- All 17 key DOM IDs preserved; JS unchanged between v1.5 and v1.6 except the carSvg constant.
- `Version` / `LatestVersion` constants bumped to 1.6 in lockstep; `/version` reflects automatically.

---

## [v1.5] – 2026-06-02

### 🇨🇳 中文

#### 新增（Added）
- **多车切换器**：UI 顶部新增 Vehicle 段，按车辆名字（不是 ID）切换；副标题显示行驶次数，hover tooltip 显示市场名称。
- `GET /cars` 路由：返回当前账号下所有车辆，带 `name` / `model` / `marketing_name` / `drive_count`。
- `GET /version` 路由：返回构建版本与 Go 版本，便于部署校验。
  - 同时返回 `latest_version` 与 `is_latest`，前端据此渲染版本徽章（绿色 = 最新 / 红色 = 需更新）。
  - 默认值：`version=1.5`，`latest_version=1.5`，`is_latest=true`。
- `/trips` 新增 `?car_id=N` 参数（缺省 = 全部车辆）。
- `/trips` 新增 `?max_points=N` 服务端抽稀，单趟最多 N 点 + 首末。
- 时间格式更宽松：支持 `YYYY-MM-DD` / `YYYY-MM-DDTHH:MM` / `YYYY-MM-DDTHH:MM:SS` / 带时区。
- 14 个 `DATABASE_*` 环境变量（池大小、生命周期、超时、缓存等）全量可调。
- `/health` 输出 pgxpool 完整指标（acquire_count、empty_acquire、new_conns 等）。
- 双语 README + CHANGELOG。
- **CI/CD**：新增 `.github/workflows/docker-publish.yml`，每次 push `vX.Y` tag 自动构建 `linux/amd64` + `linux/arm64` 多架构镜像并推送到 `ghcr.io/6547709/tesla-trail-map`。Dockerfile 升级为 buildx 多架构（`TARGETOS/TARGETARCH` 交叉编译，无需 QEMU 编译）+ 非 root 运行 + HEALTHCHECK + OCI 标签。

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
  - Also returns `latest_version` and `is_latest`, which the UI uses to render the header pill (green = up to date / red = update available).
  - Defaults: `version=1.5`, `latest_version=1.5`, `is_latest=true`.
- `/trips` accepts `?car_id=N` (omit = all cars).
- `/trips` accepts `?max_points=N` for server-side downsampling (keeps first/last + every ⌈total/N⌉-th point).
- Flexible time-input parser: `YYYY-MM-DD`, `YYYY-MM-DDTHH:MM[:SS]`, with optional timezone.
- 14 `DATABASE_*` env vars covering pool size, lifetimes, timeouts, caches.
- `/health` exposes the full pgxpool stat block.
- Bilingual README + CHANGELOG.
- **CI/CD**: added `.github/workflows/docker-publish.yml`. Every `vX.Y` tag triggers a multi-arch build (`linux/amd64` + `linux/arm64`) pushed to `ghcr.io/6547709/tesla-trail-map`. Dockerfile is now a proper buildx multi-arch one (`TARGETOS/TARGETARCH` cross-compile, no QEMU compile step), runs as non-root, ships a HEALTHCHECK, and stamps OCI labels.

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

# GeeGooAgent Chat TUI（Hermes 全套对标）设计

> 日期：2026-07-12  
> 状态：已批准；Phase A 实施中（feat/chattui-hermes-parity）  
> 参考：[Hermes TUI](https://hermes-agent.nousresearch.com/docs/user-guide/tui)、Hermes `display.details_mode` / `/details`  
> 原则：对齐 Hermes **交互语义与信息架构**；用 Go + Bubble Tea 自研，**不**引入 Node `ui-tui`、**不**照搬 Hermes 源码。

## 1. 目标与非目标

### 目标

用户已确认做 **全套**（相对此前方案 1/2 的摘要折叠）。`geegoo chat` 在 TTY 下提供接近 Hermes TUI 的体验：

1. **Alternate-screen 真 TUI**：可重绘，思考/工具块可 chevron 真折叠/展开  
2. **Details 体系**：全局 `hidden|collapsed|expanded` + 分 section 覆盖 + `/details`  
3. **直播 vs 历史**：当前生成中的 thinking/tools 默认展开；历史块默认折叠（可展开回看全文）  
4. **状态栏 + 忙碌指示**：ready / thinking / running / interrupted；耗时、模型、dry-run、YOLO/approval  
5. **Overlay**：`/help`、`/model`、`/sessions`（切换/新建/关闭 live session）、approval 确认  
6. **输入区**：多行、括号粘贴、slash 补全浮层；鼠标滚动/点击折叠头（可配）  
7. **多 live session**：同进程多会话切换（对齐 Hermes `/sessions` / `Ctrl+X`）  
8. **降级路径**：非 TTY / `GEEGOO_CHAT_PLAIN=1` / `--cli` 继续现有 `chatui` + `go-prompt`

### 非目标

- 不移植 Hermes Node TUI / `HERMES_TUI_GATEWAY_URL` WebSocket 架构  
- 不做 IM Gateway、Desktop Electron、LaTeX 数学渲染（可后置）  
- 不改 Agent Loop / MCP / 压缩算法语义（仅 UI 消费 `EmitProgress`）  
- `agent-runtime` HTTP/SSE 不强制上 TUI（服务端保持日志友好输出）  
- 不一次交付完美皮肤市场；P1 一套默认主题，P3 可换色键即可

## 2. 已确认决策

| 项 | 选择 |
|---|---|
| 方案 | **3：真 TUI（Bubble Tea）全套** |
| 思考默认 | 历史 **collapsed**；当前流式 **expanded**；结束后按 section/全局模式收起 |
| 工具默认 | 同思考（直播 expanded，历史 collapsed） |
| 全局默认 `details_mode` | `collapsed` |
| 技术栈 | Go `charmbracelet/bubbletea` + `bubbles` + 现有 `lipgloss`/`glamour` |
| Legacy | 保留 `internal/cli/chatui` 作 plain/CI/管道兜底 |
| 配置落盘 | `config.json` → `display.*`（与 Hermes `config.yaml` display 语义对齐） |

## 3. 架构

```text
cmd/geegoo chat
    │
    ├─ TTY && !plain && !--cli  →  chattui.App.Run()
    │                                 │
    │                                 ├─ Model (bubbletea)
    │                                 │    ├─ transcript (virtual list)
    │                                 │    ├─ composer (textarea + slash menu)
    │                                 │    ├─ overlays (help/model/sessions/approval)
    │                                 │    └─ status bar
    │                                 │
    │                                 └─ SessionController[]  (1..N live chats)
    │                                        └─ Agent.Run + ProgressSink → tea.Msg
    │
    └─ else                     →  chatrepl + chatui（现有路径）
```

### 3.1 包结构

```text
internal/cli/
├── chatui/          # 保留：plain / --cli
├── chatrepl/        # 保留：legacy REPL；抽出共享 slash 处理到 chatcmd/
├── chatcmd/         # ★ 新增：slash 命令解析与副作用（CLI/TUI 共用）
└── chattui/         # ★ 新增：Bubble Tea 应用
    ├── app.go           # Run、信号、alt-screen
    ├── model.go         # 根 Model
    ├── update.go        # Update 路由
    ├── view.go          # View 拼装
    ├── transcript.go    # 消息行 / section / 虚拟高度
    ├── section.go       # thinking/tools/activity/reply 折叠状态
    ├── composer.go      # 输入 + slash 补全
    ├── overlay_*.go     # help / model / sessions / approval
    ├── status.go        # 状态栏
    ├── mouse.go         # mouse tracking 预设
    ├── progress.go      # EmitProgress → Msg
    ├── theme.go         # 颜色 / light-dark 探测（可选）
    └── session_ctl.go   # live session 生命周期
```

### 3.2 Progress 适配

从 `chatrepl.attachProgress` 抽出接口：

```go
type ProgressSink interface {
    EmitProgress(event string, data map[string]any)
}
```

| event | TUI 行为 |
|---|---|
| `turn_start` | 新 turn 行；重置 live section |
| `stream_delta` | 追加到 live reply 或 live thinking（若带 reasoning 流） |
| `llm_plan` | 写入 thinking section 全文；按 mode 展开/折叠 |
| `llm_tools` / `tool_start` / `tool_done` | tools section 行；直播展开 |
| `reply_start` / 最终 assistant | reply section；abort 中间 typewriter 计划文本 |
| `error` | 内联错误 + 可选 floating alert |
| `context_compressed` / `context_hygiene` | activity（默认 hidden）或状态栏徽章 |

Agent / runtime **不依赖** Bubble Tea；仅发事件。

### 3.3 折叠语义（核心 UX）

每个可折叠块：

```text
▾ 💭 思考 · 12 行 · 1.4s     ← expanded（直播或用户展开）
  | ...正文...

▸ 💭 思考 · 12 行 · 1.4s     ← collapsed（历史默认）
```

规则：

1. **Live block**（当前 turn 正在生成）：强制 UI 层 `expanded`，忽略用户 collapsed——保证可跟读；生成结束后应用 `effectiveMode(section)`  
2. **Historical blocks**：默认 `collapsed`，除非  
   - section 覆盖为 `expanded`，或  
   - 用户对本块显式展开（块级 override，记在内存，不写 config）  
3. **`hidden`**：不渲染该 section（正文仍进会话存储，可 `/details … expanded` 后对后续块生效；对已 hidden 的历史块可用「显示上一次思考」命令重放）  
4. 键盘：焦点在块头时 `Enter`/`Space` 切换；鼠标点击块头切换（mouse on）

`effectiveMode(section) = sections[section] ?? details_mode`

## 4. 配置

`config.json` 新增：

```json
{
  "display": {
    "interface": "tui",
    "details_mode": "collapsed",
    "sections": {
      "thinking": "",
      "tools": "",
      "activity": "hidden"
    },
    "mouse_tracking": "wheel",
    "status_indicator": "emoji",
    "show_reasoning": true
  }
}
```

| 键 | 语义 |
|---|---|
| `interface` | `tui`（默认当 TTY）\| `cli` |
| `details_mode` | `hidden`\|`collapsed`\|`expanded` |
| `sections.*` | 空字符串 = 跟随全局；否则覆盖 |
| `mouse_tracking` | `off`\|`wheel`\|`buttons`\|`all` |
| `status_indicator` | `emoji`\|`unicode`\|`ascii`（不做 kaomoji 也可先 emoji） |
| `show_reasoning` | `false` 时不渲染 thinking（且不写入可视 segment）；模型 `/think` 仍可独立开关 |

CLI 覆盖：

- `geegoo chat --tui` / `--cli`  
- `GEEGOO_CHAT_TUI=1` / `GEEGOO_CHAT_PLAIN=1`

## 5. 命令与快捷键

### 5.1 现有 slash（TUI/CLI 共用 `chatcmd`）

`/help` `/exit` `/quit` `/session` `/tools` `/toolsets` `/trace` `/flow` `/run` `/dry-run` `/model` `/verbose` `/think`

### 5.2 新增（TUI 为主，CLI 可降级打印）

| 命令 | 行为 |
|---|---|
| `/details [hidden\|collapsed\|expanded\|cycle]` | 全局模式 |
| `/details <thinking\|tools\|activity> [mode\|reset]` | 分块 |
| `/details last` | 展开并滚动到上一轮 thinking/tools |
| `/sessions` / `/switch` | live session overlay |
| `/sessions new` | 新建 live session |
| `/mouse [off\|wheel\|buttons\|all\|toggle]` | 鼠标预设并持久化 |
| `/indicator [emoji\|unicode\|ascii]` | 状态指示器 |

`/verbose` 与 `/details` 关系：

- `/verbose off` ≈ tools+thinking 对**新输出**偏 hidden/collapsed（映射到 details，不另搞第三套）  
- 长期以 `/details` 为准；`/verbose` 保留兼容，内部转换成 details 写入

### 5.3 快捷键（TUI）

| 键 | 行为 |
|---|---|
| `Enter` | 发送（单行）/ 补全选中 |
| `Alt+Enter` 或 `Ctrl+J` | 输入换行 |
| `Ctrl+C` | 中断当前 turn；再按或空输入退出确认策略与现网一致 |
| `Esc` | 关闭 overlay；或中断 turn |
| `Ctrl+X` | 打开 sessions switcher |
| `↑/↓` | 历史输入或 overlay 导航 |
| 鼠标滚轮 | 滚动 transcript（wheel 模式） |

## 6. UI 布局

```text
┌─ transcript (可滚动) ─────────────────────────┐
│ 用户 / 助手 / ▸思考 / ▸工具 / 回复 …           │
├─ overlays（居中模态，按需）───────────────────┤
├─ status ──────────────────────────────────────┤
│ ● running · deepseek-chat · ⏱ 12s · SPCX …    │
├─ composer ────────────────────────────────────┤
│ ❯ _                                           │
└───────────────────────────────────────────────┘
```

- Transcript 用虚拟高度估算（折叠行矮、展开行高），保证 live tail 跟随  
- 最终回复用 glamour 渲染 markdown（复用 chatui 主题色）  
- Approval：模态确认 y/n，替代 stdin 行读（TUI）；CLI 保留行读

## 7. 多 Live Session

- 进程内 `[]*SessionController`，每个绑定独立 `ChatSession` + cancel  
- Switcher 列出 title/id/状态；`Enter` 切换，`Ctrl+N` 新建，`Ctrl+D` 关闭（保存后移出 live）  
- 后台 session 可继续跑完 turn（可选 P3：后台跑时状态栏 `▶ N`）；P2 可先做「切换时若 busy 则提示」  
- 持久化仍走现有 SQLite SessionStore；live 仅是 UI 附着

## 8. 交付阶段（全套范围，分 PR 合并）

全套均在范围内；按可合并增量拆分，避免巨型 PR。

### Phase A — 可折叠 TUI 核心

- bubbletea 壳、alt-screen、transcript、thinking/tools 真折叠  
- ProgressSink 接入、流式 reply  
- `/details` + config `display`  
- `--cli` / plain 兜底  
- 单 session

### Phase B — 状态栏 + Overlay + 输入

- status bar、busy indicator  
- `/help` `/model` overlay  
- composer 多行 + slash 浮层  
- approval 模态  
- `/verbose` 映射到 details

### Phase C — 鼠标 + 多 session + 打磨

- mouse presets  
- `/sessions` live switcher + `Ctrl+X`  
- 虚拟高度跟随、历史折叠在流式时不抖动  
- light/dark 简易探测（可选）  
- 文档：`README` + `config.example.json` + roadmap 勾选

每 Phase 需：`go test ./internal/cli/chattui/...` + 手工 TTY 检查清单。

## 9. 测试策略

- **单测**：section `effectiveMode`、折叠高度估算、Progress 事件 → 消息归并、`/details` 解析、config 读写  
- **Model 测**：tea.Msg 序列（无真实 TTY）  
- **手工**：TTY 下折叠点击/键盘、流式历史不展开、overlay、双 session 切换、`--cli` 回归  
- **不测**：全终端矩阵像素级一致

## 10. 风险与缓解

| 风险 | 缓解 |
|---|---|
| Windows 终端 mouse/alt-screen 差异 | 默认 `mouse=wheel`；失败自动降级 off；保留 `--cli` |
| go-prompt 与 bubbletea 抢终端 | TUI 路径不再进 go-prompt；legacy 独占 |
| 长思考虚拟列表卡顿 | 截断展示上限（如 200 行）+ 折叠高度缓存 |
| 范围膨胀 | 严格按 Phase A→B→C 合并；C 未完成前不宣称「全套已交付」 |

## 11. 成功标准

1. TTY 默认进入 TUI；思考/工具可折叠，历史默认折叠，直播可跟读  
2. `/details` 与 config 持久化行为与 Hermes 文档语义一致  
3. `/model`、`/sessions`、approval 为 overlay，而非打断式纯文本流  
4. 鼠标滚轮可滚动；点击块头可折叠（mouse 开启时）  
5. 同进程 ≥2 live session 可切换  
6. `--cli` / plain 下旧行为不回归  

## 12. Spec 自检

- [x] 无 TBD/占位实现细节冒充已决（阶段边界已写清）  
- [x] 与「方案 2 摘要折叠」无矛盾（已明确否决，走真 TUI）  
- [x] 非目标排除 Node TUI / IM  
- [x] Agent 核心与 UI 解耦经 ProgressSink  
- [x] 全套 = A+B+C，非仅 MVP  

---

**请审阅本文。** 确认后进入 `writing-plans` 拆实施计划与任务清单，再按 Phase A→B→C 实现。

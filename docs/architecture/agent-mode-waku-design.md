# Agent 模式（Waku Dashboard 全量复刻）设计 SSOT

> 运营系统 **双模式**：**平台模式**（现有 Monday 运营）+ **Agent 模式**（Waku 三栏 Dashboard）。

## 1. 目标

| 模式 | 用户 | 布局 | 功能 |
|------|------|------|------|
| **平台模式** | 运营/运维 | 现有左侧栏 + 顶栏 | 用户/策略/模板/服务器… |
| **Agent 模式** | 运营 + Agent 调试 | Waku：`Nav \| Main \| Chat Dock` | 与 waku `ops/static` 功能对齐 |

成功标准：Agent 模式下，熟悉 Waku 的用户无需重新学习即可操作 Overview / Gateway / Loop / Memory / Tools / Database / Ops / Compare / Settings，且右侧 Chat Dock 与架构图 live 动画联动。

## 2. Waku → GeeGoo 映射

| Waku Nav | GeeGoo 数据源 | 备注 |
|----------|---------------|------|
| Overview | `/v1/metrics/overview` + `/v1/doctor` + 架构图 SSE | 5s 轮询 + `sessionEventsStream` 动画 |
| Gateway | `/v1/sessions` + session messages | 会话收件箱 |
| Loop | `/v1/sessions/{id}/trace` + live events | 与 Gateway 联动 |
| Memory | `/v1/memory/*` + subtabs | Semantic/Episodic/Procedural 映射 chunks + skills |
| Tools | `/v1/tools` | Registry + MCP 状态 |
| Database | `/v1/dashboard/db` + `/v1/dashboard/query` | 只读 SQL（PostgreSQL） |
| Ops | `/v1/doctor` + trace 汇总 | LLM Ops 面板 |
| Compare | `/v1/dashboard/compare` | 模型对比（二期后端） |
| Settings | runtime config + catalog model | 模型切换 |

## 3. Flutter 模块结构

```
trading_operation/lib/modules/agent_mode/
  theme/waku_theme.dart
  controllers/agent_mode_controller.dart   # 路由、5s tick、nav 计数
  shell/agent_shell.dart                   # 三栏 + resizer
  shell/agent_nav.dart
  shell/agent_dock.dart                    # 复用 geegoo_agent_chat
  views/overview_view.dart
  views/gateway_view.dart
  views/loop_view.dart
  views/memory_view.dart
  views/tools_view.dart
  views/database_view.dart
  views/ops_view.dart
  views/compare_view.dart
  views/settings_view.dart
  widgets/waku_arch_diagram.dart           # 从 diagram.js 移植
```

## 4. 模式切换

- 顶栏右侧：`平台 | Agent` `SegmentedButton`
- `SharedPreferences` 键 `app_mode` = `platform` | `agent`
- 平台模式隐藏 FAB；Agent 模式用右侧 Dock 替代 FAB

## 5. 后端增量（GeeGooAgent）

| API | 用途 |
|-----|------|
| `GET /v1/dashboard/data` | 聚合 JSON（对标 waku `/api/data`） |
| `POST /v1/dashboard/query` | 只读 SQL |
| `GET /v1/dashboard/sessions/{id}/messages` | Gateway 消息列表 |
| `POST /v1/dashboard/voice` | 语音转写（可选，对标 waku Whisper） |

## 6. 分期交付

| 阶段 | 内容 |
|------|------|
| **A** | 双模式壳 + Nav + Dock + 9 视图接线 | ✅ |
| **B** | `GET /v1/dashboard/data` + SQL + 架构图动画 + hash 路由 | ✅ |
| **C** | Compare SSE + Memory 子 Tab + Ops；Voice 待 Whisper | 🟡 |

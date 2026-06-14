# 工程化文档

实现 GeeGoo Agent 的**执行依据**（架构蓝图见 [../architecture/README.md](../architecture/README.md)）。

| 文档 | 用途 |
|------|------|
| [requirements.md](./requirements.md) | 范围、技术栈、分期交付、验收标准、质量门禁 |
| [coding-standards.md](./coding-standards.md) | **编码规范**（命名、分层、错误处理、Client/Tool 模板） |
| [testing-standards.md](./testing-standards.md) | **测试规范**（每 Step 必交付用例、mock、覆盖率、验收命令） |
| [cursor-workflow.md](./cursor-workflow.md) | 如何用 Cursor Agent **分步骤、高效率**完成开发 |

**阅读顺序**：requirements → coding-standards + testing-standards → cursor-workflow → 按 Step 开工。

**铁律**：每 Step 未完成 [testing-standards.md §5](./testing-standards.md#5-分-step-测试交付物强制) 对应用例，不得进入下一步。

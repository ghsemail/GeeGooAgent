#!/usr/bin/env python3
import os
import subprocess

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
FILES = [
    "catalog.go",
    "catalog_test.go",
    "chat_meta.go",
    "detect.go",
    "detect_test.go",
    "eval_test.go",
    "multi_strategy.go",
    "runner.go",
    "runner_test.go",
    "runner_card_test.go",
    "store.go",
    "types.go",
]

for name in FILES:
    proc = subprocess.run(
        ["git", "show", f"HEAD:internal/taskflow/{name}"],
        cwd=ROOT,
        capture_output=True,
    )
    if proc.returncode != 0:
        continue
    text = proc.stdout.decode("utf-8")
    text = text.replace("package taskflow", "package chat", 1)
    text = text.replace("TemplateMultiStrategyCompare", "skills.SkillMultiStrategyCompare")
    text = text.replace("taskflow_started", "workflow_started")
    text = text.replace("taskflow_completed", "workflow_completed")
    text = text.replace("taskflow_card", "workflow_card")
    text = text.replace("taskflow_step_skipped", "workflow_step_skipped")
    text = text.replace("由 taskflow 串行执行", "由 workflow 串行执行")
    text = text.replace('"template": flow.Template', '"skill": flow.Template')
    if name == "multi_strategy.go" and "internal/skills" not in text:
        text = text.replace(
            '"github.com/ghsemail/GeeGooAgent/internal/sessiontask"',
            '"github.com/ghsemail/GeeGooAgent/internal/sessiontask"\n\t"github.com/ghsemail/GeeGooAgent/internal/skills"',
        )
    if name == "types.go":
        text = text.replace('\tTemplateMultiStrategyCompare = "multi_strategy_compare"\n\n', "")
    out = os.path.join(ROOT, "internal", "workflow", "chat", name)
    with open(out, "w", encoding="utf-8", newline="\n") as f:
        f.write(text)
    print("wrote", name)

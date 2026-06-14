"""Skill pack loader — reads manifest.yaml from skills/{name}/."""

from __future__ import annotations

from pathlib import Path
from typing import Any

import yaml
from pydantic import BaseModel, Field

from geegoo_agent.exceptions import ConfigError


class SkillManifest(BaseModel):
    name: str
    version: str
    description: str = ""
    phase: int = 1
    mode: str = "scheduled"
    tools: list[str]
    llm_tasks: list[str] = Field(default_factory=list)
    rules: list[str] = Field(default_factory=list)
    bundled: list[str] = Field(default_factory=list)
    workflow: dict[str, Any] = Field(default_factory=dict)
    indices: list[dict[str, str]] = Field(default_factory=list)

    @property
    def registered_tool_count(self) -> int:
        return len(self.tools)


class SkillLoader:
    def __init__(self, project_root: Path | None = None) -> None:
        self._root = project_root or Path(__file__).resolve().parents[3]

    @property
    def skills_dir(self) -> Path:
        return self._root / "skills"

    def manifest_path(self, skill_name: str) -> Path:
        return self.skills_dir / skill_name / "manifest.yaml"

    def load(self, skill_name: str) -> SkillManifest:
        path = self.manifest_path(skill_name)
        if not path.exists():
            raise ConfigError(f"skill manifest not found: {path}")
        try:
            data = yaml.safe_load(path.read_text(encoding="utf-8"))
        except yaml.YAMLError as exc:
            raise ConfigError(f"invalid YAML in manifest: {path}") from exc
        try:
            return SkillManifest.model_validate(data)
        except Exception as exc:
            raise ConfigError(f"invalid manifest fields for {skill_name}: {exc}") from exc

    def validate_asset_paths(self, skill_name: str) -> list[str]:
        """Return missing relative paths for rules, bundled scripts, and skill docs."""
        manifest = self.load(skill_name)
        missing: list[str] = []
        for rel in manifest.rules + manifest.bundled:
            if not (self._root / rel).exists():
                missing.append(rel)
        skill_dir = self.skills_dir / skill_name
        for doc in ("workflow.md", "template.md", "SKILL.md", "manifest.yaml"):
            if not (skill_dir / doc).exists():
                missing.append(f"skills/{skill_name}/{doc}")
        return missing

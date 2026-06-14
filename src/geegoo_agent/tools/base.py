"""Base tool abstraction."""

from __future__ import annotations

from abc import ABC, abstractmethod
from typing import Any

from pydantic import BaseModel

from geegoo_agent.llm.types import ToolSchema
from geegoo_agent.tools.types import ToolCategory, ToolContext, ToolResult


class BaseTool(ABC):
    name: str
    description: str
    category: ToolCategory
    input_model: type[BaseModel]

    def to_schema(self) -> ToolSchema:
        schema = self.input_model.model_json_schema()
        return ToolSchema(
            name=self.name,
            description=self.description,
            parameters=schema,
        )

    def validate_params(self, arguments: dict[str, Any]) -> BaseModel:
        return self.input_model.model_validate(arguments)

    @abstractmethod
    def run(self, params: BaseModel, ctx: ToolContext) -> ToolResult: ...

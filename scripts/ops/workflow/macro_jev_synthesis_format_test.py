#!/usr/bin/env python3
from __future__ import annotations

import unittest

from macro_jev_synthesis_format import format_pool_jev_reason


class TestFormatPoolJevReason(unittest.TestCase):
    def test_prefers_chinese_section_after_english_preamble(self) -> None:
        raw = (
            "Let me synthesize a macro JEV verdict.\n\n"
            "## 整体结论\n\n"
            "- **JEV 方向**：short\n"
            "- **理由**：上证压制权重。"
        )
        out = format_pool_jev_reason(raw)
        self.assertIn("整体结论", out)
        self.assertNotIn("Let me synthesize", out)

    def test_keeps_chinese_only(self) -> None:
        raw = "## 整体结论\n\nA股池偏空。"
        self.assertEqual(format_pool_jev_reason(raw), raw)


if __name__ == "__main__":
    unittest.main()
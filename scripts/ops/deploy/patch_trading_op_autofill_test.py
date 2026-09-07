#!/usr/bin/env python3
"""Tests for trading_operation autofill bridge deploy helper."""
from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path

MODULE = Path(__file__).resolve().parents[1] / "deploy" / "patch_trading_op_autofill.py"
spec = importlib.util.spec_from_file_location("patch_trading_op_autofill", MODULE)
mod = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(mod)


class PatchIndexHtmlTest(unittest.TestCase):
    def test_injects_after_clarify_overlay(self) -> None:
        html = '<script src="clarify-overlay.js"></script>\n<script src="flutter_bootstrap.js"></script>'
        out = mod.patch_index_html(html)
        self.assertIn('autofill-bridge.js', out)
        self.assertLess(out.index("clarify-overlay"), out.index("autofill-bridge"))
        self.assertLess(out.index("autofill-bridge"), out.index("flutter_bootstrap"))

    def test_idempotent(self) -> None:
        html = '<script src="autofill-bridge.js"></script>'
        self.assertEqual(mod.patch_index_html(html), html)


if __name__ == "__main__":
    unittest.main()

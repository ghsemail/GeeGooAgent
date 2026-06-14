"""Business mapping helpers."""

from __future__ import annotations

_ATTITUDE_TO_RESULT = {
    "bullish": "long",
    "bearish": "short",
    "neutral": "neutral",
}


def attitude_to_result(attitude: str) -> str:
    return _ATTITUDE_TO_RESULT.get(attitude.lower(), "neutral")


def is_a_share(code: str) -> bool:
    """True for China A-share tickers (SH/SZ suffix)."""
    upper = str(code).strip().upper()
    return upper.endswith(".SH") or upper.endswith(".SZ")


A_SHARE_CAPITAL_SKIP_REASON = "A股标的跳过（当前 OpenD 无 A 股资金流向权限）"


def format_capital_distribution(data: dict) -> str:
    """Format capital distribution for reports (unit: 亿)."""

    def net(in_key: str, out_key: str) -> float:
        return (float(data.get(in_key, 0)) - float(data.get(out_key, 0))) / 1e8

    def fmt_line(label: str, in_key: str, out_key: str) -> str:
        in_val = float(data.get(in_key, 0)) / 1e8
        out_val = float(data.get(out_key, 0)) / 1e8
        net_val = in_val - out_val
        return (
            f"{label}净流入：{net_val:+.1f}亿"
            f"（滞留：{in_val:+.1f}亿 / 撤离：{-out_val:+.1f}亿）"
        )

    lines = [
        fmt_line("超大单", "capital_in_super", "capital_out_super"),
        fmt_line("大单", "capital_in_big", "capital_out_big"),
        fmt_line("中单", "capital_in_mid", "capital_out_mid"),
        fmt_line("小单", "capital_in_small", "capital_out_small"),
    ]
    update_time = data.get("update_time", "")
    if update_time:
        lines.append(f"更新时间：{update_time}")
    return "\n".join(lines)

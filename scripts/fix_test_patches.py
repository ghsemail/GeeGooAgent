from pathlib import Path

root = Path(__file__).resolve().parents[1] / "tests"
for p in root.rglob("*.py"):
    t = p.read_text(encoding="utf-8")
    n = t.replace('"geegoo.', '"geegoo_agent.').replace("'geegoo.", "'geegoo_agent.")
    if n != t:
        p.write_text(n, encoding="utf-8", newline="\n")
        print(p)

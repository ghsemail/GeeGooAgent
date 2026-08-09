#!/usr/bin/env python3
"""Reorganize GeeGooAgent scripts/ into ops subfolders and remove local junk."""
from __future__ import annotations

import shutil
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPTS = ROOT / "scripts"
GIT = r"C:\Program Files\Git\bin\git.exe"

KEEP_AT_ROOT = {
    "install.sh",
    "install-go.sh",
    "ensure-go.sh",
    "reorganize_scripts.py",
    "README.md",
}
KEEP_DIRS = {"hooks"}


def category(name: str) -> str:
    if name.startswith(("deploy_", "fast_deploy_", "patch_")):
        return "ops/deploy"
    if name.startswith("verify_"):
        return "ops/verify"
    if name.startswith(("probe_", "diagnose_", "diag_", "ssh_probe_", "audit_")):
        return "ops/probe"
    if name.startswith(
        ("run_", "clear_", "delete_and_", "rerun_", "push_postmarket", "ssh_run_")
    ):
        return "ops/workflow"
    if name.startswith(
        (
            "fix_",
            "sync_",
            "recover_",
            "migrate_",
            "ensure_",
            "restart_",
            "refresh_",
            "rewrite_",
            "finalize_",
            "configure_",
            "stop_",
            "start_",
            "cancel_",
            "setup_",
        )
    ):
        return "ops/fix"
    if name.endswith(".go") or name.startswith(("test_", "local_test_", "sse_chat_", "bench_")):
        return "dev"
    return "ops/misc"


def is_tracked(rel: str) -> bool:
    return (
        subprocess.run(
            [GIT, "-C", str(ROOT), "ls-files", "--error-unmatch", rel],
            capture_output=True,
        ).returncode
        == 0
    )


def git_mv(src: Path, dst: Path) -> None:
    dst.parent.mkdir(parents=True, exist_ok=True)
    rel_src = str(src.relative_to(ROOT))
    rel_dst = str(dst.relative_to(ROOT))
    if is_tracked(rel_src):
        subprocess.run([GIT, "-C", str(ROOT), "mv", rel_src, rel_dst], check=True)
    else:
        shutil.move(str(src), str(dst))


def main() -> None:
    # Remove local-only junk (gitignored _* and probe outputs).
    removed = 0
    for path in SCRIPTS.glob("_*"):
        if path.is_file():
            path.unlink()
            removed += 1
    for pattern in (
        "probe_*_result.*",
        "probe_*_out.txt",
        "probe_*_session.json",
        "probe_*.txt",
        "run_*_out.txt",
        "rerun_*_out.txt",
    ):
        for path in SCRIPTS.glob(pattern):
            if path.is_file() and not path.name.startswith("_"):
                # keep if tracked? only delete untracked outputs at root
                rel = path.relative_to(ROOT)
                tracked = (
                    subprocess.run(
                        [GIT, "-C", str(ROOT), "ls-files", "--error-unmatch", str(rel)],
                        capture_output=True,
                    ).returncode
                    == 0
                )
                if not tracked:
                    path.unlink()
                    removed += 1
    print(f"removed local junk files: {removed}")

    moves: list[tuple[Path, Path]] = []
    for path in sorted(SCRIPTS.iterdir()):
        if path.name in KEEP_AT_ROOT or path.name in KEEP_DIRS:
            continue
        if path.is_dir():
            if path.name in KEEP_DIRS:
                continue
            # unexpected top-level dir besides hooks
            if path.name.startswith("ops") or path.name == "dev":
                continue
            continue
        if not path.is_file():
            continue
        if path.name == "reorganize_scripts.py":
            continue
        dest_dir = SCRIPTS / category(path.name)
        target = dest_dir / path.name
        if target == path:
            continue
        if target.exists():
            raise SystemExit(f"target exists: {target}")
        moves.append((path, target))

    for src, dst in moves:
        git_mv(src, dst)
    print(f"git mv count: {len(moves)}")


if __name__ == "__main__":
    main()

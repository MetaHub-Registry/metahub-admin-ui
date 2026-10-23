#!/usr/bin/env python3
"""Fail when non-test source mentions a host outside the allowlist.

This is a review gate, not a network sandbox: it makes a new outbound host a
deliberate, reviewed change (for example, no telemetry endpoint slips in with
a dependency-free edit).
"""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
ALLOWLIST = Path(__file__).resolve().parents[1] / "outbound-allowlist.txt"
SOURCE_SUFFIXES = {".go", ".ts", ".tsx", ".js", ".jsx"}
SKIP_DIRS = {"node_modules", "dist", "site", "vendor", ".git", ".venv"}
TEST_SUFFIXES = (".test.ts", ".test.tsx", ".spec.ts", ".spec.tsx", "_test.go")
URL = re.compile(r"https?://([A-Za-z0-9._:-]+)")
HOSTNAME = re.compile(r"^([A-Za-z0-9-]+\.)+[A-Za-z0-9-]+$")


def entries():
    for line in ALLOWLIST.read_text(encoding="utf-8").splitlines():
        line = line.split("#", 1)[0].strip().lower()
        if line:
            yield line


def allowed(host, allow):
    for entry in allow:
        if entry.startswith("*."):
            if host.endswith(entry[1:]):
                return True
        elif host == entry:
            return True
    return False


def main():
    allow = list(entries())
    failures = []
    for path in sorted(ROOT.rglob("*")):
        if not path.is_file() or path.suffix not in SOURCE_SUFFIXES:
            continue
        if any(part in SKIP_DIRS for part in path.relative_to(ROOT).parts):
            continue
        if path.name.endswith(TEST_SUFFIXES):
            continue
        try:
            text = path.read_text(encoding="utf-8", errors="ignore")
        except OSError:
            continue
        for match in URL.finditer(text):
            host = match.group(1).split(":")[0].rstrip(".")
            if not HOSTNAME.match(host):
                continue
            if not allowed(host.lower(), allow):
                failures.append((path.relative_to(ROOT), host))

    if not failures:
        print("outbound host check: OK")
        return 0

    for path, host in failures:
        print(f"::error file={path}::outbound host '{host}' is not on the allowlist")
    print("Add the host to .github/outbound-allowlist.txt after review.")
    return 1


if __name__ == "__main__":
    sys.exit(main())

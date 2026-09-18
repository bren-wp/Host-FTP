#!/usr/bin/env python3
"""Fail CI if production UI source contains demo/development-facing copy."""

from __future__ import annotations

import ast
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SOURCE_DIRS = [ROOT / "internal" / "desktop", ROOT / "internal" / "platform", ROOT / "cmd"]

BANNED = {
    "example.com": "reserved sample host",
    "runneradmin": "CI account name",
    "john doe": "sample persona",
    "pro plan": "sample subscription label",
    "ghost ftp premium": "non-production upsell copy",
    "production server": "sample site name",
    "media server": "sample site name",
    "client a": "sample client name",
    "dev.example": "development sample host",
    "staging.example": "staging sample host",
    "lorem ipsum": "placeholder copy",
    "demo credentials": "demo-only copy",
    "placeholder text": "placeholder copy",
}

STRING_RE = re.compile(r'"(?:\\.|[^"\\])*"|\x60(?:\\.|[^\x60\\])*\x60', re.S)


def decoded_string(token: str) -> str:
    if token.startswith("`"):
        return token[1:-1]
    try:
        value = ast.literal_eval(token)
    except Exception:
        return token
    return value if isinstance(value, str) else ""


def main() -> int:
    problems: list[str] = []
    for directory in SOURCE_DIRS:
        if not directory.exists():
            continue
        for path in directory.rglob("*.go"):
            if path.name.endswith("_test.go"):
                continue
            text = path.read_text(encoding="utf-8")
            for match in STRING_RE.finditer(text):
                value = decoded_string(match.group(0))
                normalized = value.casefold()
                for banned, reason in BANNED.items():
                    if banned in normalized:
                        line = text.count("\n", 0, match.start()) + 1
                        problems.append(
                            f"{path.relative_to(ROOT)}:{line}: {reason}: {banned!r}"
                        )
    if problems:
        print("PRODUCT_COPY_AUDIT=FAIL")
        for problem in problems:
            print(problem)
        return 1
    print("PRODUCT_COPY_AUDIT=PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

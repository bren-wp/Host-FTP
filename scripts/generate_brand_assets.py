#!/usr/bin/env python3
"""Generate and validate deterministic Ghost FTP desktop brand assets.

The canonical Ghost FTP mark follows the approved dark reference UI: a compact
ghost silhouette carrying two transfer arrows, rendered with a cyan -> electric
blue -> violet gradient on a transparent/dark rounded-square application tile.
The renderer is dependency-free so Windows builds can reproduce PNG/ICO assets
without network access or bundled font/image tooling.
"""

from __future__ import annotations

import argparse
import base64
import binascii
import math
from pathlib import Path
import struct
import sys
import zlib

ROOT = Path(__file__).resolve().parents[1]
ICON_PNG = ROOT / "build" / "icon.png"
ICON_ICO = ROOT / "build" / "icon.ico"
BRANDMARK_PNG = ROOT / "build" / "brandmark.png"
BRANDMARK_ICO = ROOT / "build" / "brandmark.ico"

PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"
ICO_SIGNATURE = b"\x00\x00\x01\x00"
CANVAS = 256
SUPERSAMPLE = 4

SOURCE_MARK_SIZE = 96
SOURCE_MARK_RGBA = zlib.decompress(base64.b64decode(
    "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"
))
if len(SOURCE_MARK_RGBA) != SOURCE_MARK_SIZE * SOURCE_MARK_SIZE * 4:
    raise RuntimeError("embedded Ghost FTP brand mark is corrupt")

TRANSPARENT = (0, 0, 0, 0)
CHARCOAL = (8, 14, 20, 255)
CYAN = (0, 229, 255, 255)
BLUE = (59, 130, 246, 255)
VIOLET = (139, 92, 246, 255)
MIST = (229, 231, 235, 255)


def _mix(a: tuple[int, int, int, int], b: tuple[int, int, int, int], t: float) -> tuple[int, int, int, int]:
    t = min(1.0, max(0.0, t))
    return tuple(round(a[i] * (1.0 - t) + b[i] * t) for i in range(4))


def _gradient(x: float, y: float) -> tuple[int, int, int, int]:
    t = min(1.0, max(0.0, (0.62 * x + 0.38 * y - 42.0) / 205.0))
    if t < 0.52:
        return _mix(CYAN, BLUE, t / 0.52)
    return _mix(BLUE, VIOLET, (t - 0.52) / 0.48)


def _inside_rounded_rect(x: float, y: float, left: float, top: float, right: float, bottom: float, radius: float) -> bool:
    if x < left or x >= right or y < top or y >= bottom:
        return False
    cx = min(max(x, left + radius), right - radius)
    cy = min(max(y, top + radius), bottom - radius)
    dx = x - cx
    dy = y - cy
    return dx * dx + dy * dy <= radius * radius


def _inside_ellipse(x: float, y: float, cx: float, cy: float, rx: float, ry: float) -> bool:
    if rx <= 0 or ry <= 0:
        return False
    dx = (x - cx) / rx
    dy = (y - cy) / ry
    return dx * dx + dy * dy <= 1.0


def _inside_rotated_ellipse(x: float, y: float, cx: float, cy: float, rx: float, ry: float, angle: float) -> bool:
    s, c = math.sin(angle), math.cos(angle)
    dx, dy = x - cx, y - cy
    px = dx * c + dy * s
    py = -dx * s + dy * c
    return (px / rx) ** 2 + (py / ry) ** 2 <= 1.0


def _inside_triangle(x: float, y: float, a: tuple[float, float], b: tuple[float, float], c: tuple[float, float]) -> bool:
    def sign(p1, p2, p3):
        return (p1[0] - p3[0]) * (p2[1] - p3[1]) - (p2[0] - p3[0]) * (p1[1] - p3[1])
    p = (x, y)
    d1, d2, d3 = sign(p, a, b), sign(p, b, c), sign(p, c, a)
    neg = d1 < 0 or d2 < 0 or d3 < 0
    pos = d1 > 0 or d2 > 0 or d3 > 0
    return not (neg and pos)


def _inside_arrow(x: float, y: float, y0: float, length: float, thickness: float) -> bool:
    left = 54.0
    body_right = left + length - 28.0
    tip = left + length
    if left <= x <= body_right and y0 - thickness / 2 <= y <= y0 + thickness / 2:
        return True
    return _inside_triangle(
        x,
        y,
        (body_right - 2.0, y0 - thickness * 1.35),
        (tip, y0),
        (body_right - 2.0, y0 + thickness * 1.35),
    )


def _inside_ghost(x: float, y: float) -> bool:
    # Sleek forward-leaning ghost: rounded crown, tapered lower body and a
    # streaming tail that visually merges with the transfer direction.
    head = _inside_rotated_ellipse(x, y, 143.0, 101.0, 54.0, 52.0, -0.18)
    shoulder = _inside_rotated_ellipse(x, y, 136.0, 132.0, 69.0, 52.0, -0.18)
    body = _inside_triangle(x, y, (83.0, 118.0), (193.0, 94.0), (171.0, 197.0))
    tail = _inside_triangle(x, y, (91.0, 128.0), (171.0, 197.0), (61.0, 176.0))
    return head or shoulder or body or tail


def _source_mark_pixel(x: float, y: float) -> tuple[int, int, int, int]:
    sx = min(SOURCE_MARK_SIZE - 1.0, max(0.0, x * (SOURCE_MARK_SIZE - 1.0) / (CANVAS - 1.0)))
    sy = min(SOURCE_MARK_SIZE - 1.0, max(0.0, y * (SOURCE_MARK_SIZE - 1.0) / (CANVAS - 1.0)))
    x0, y0 = int(math.floor(sx)), int(math.floor(sy))
    x1, y1 = min(SOURCE_MARK_SIZE - 1, x0 + 1), min(SOURCE_MARK_SIZE - 1, y0 + 1)
    fx, fy = sx - x0, sy - y0
    weights = ((x0, y0, (1.0 - fx) * (1.0 - fy)), (x1, y0, fx * (1.0 - fy)),
               (x0, y1, (1.0 - fx) * fy), (x1, y1, fx * fy))
    alpha = 0.0
    premul = [0.0, 0.0, 0.0]
    for px, py, weight in weights:
        offset = (py * SOURCE_MARK_SIZE + px) * 4
        a = SOURCE_MARK_RGBA[offset + 3] / 255.0
        alpha += a * weight
        for channel in range(3):
            premul[channel] += SOURCE_MARK_RGBA[offset + channel] * a * weight
    if alpha <= 1e-6:
        return TRANSPARENT
    return (
        round(premul[0] / alpha),
        round(premul[1] / alpha),
        round(premul[2] / alpha),
        round(alpha * 255.0),
    )


def _composite(foreground: tuple[int, int, int, int], background: tuple[int, int, int, int]) -> tuple[int, int, int, int]:
    fa = foreground[3] / 255.0
    ba = background[3] / 255.0
    out_a = fa + ba * (1.0 - fa)
    if out_a <= 1e-6:
        return TRANSPARENT
    out = []
    for channel in range(3):
        value = (foreground[channel] * fa + background[channel] * ba * (1.0 - fa)) / out_a
        out.append(round(value))
    return (out[0], out[1], out[2], round(out_a * 255.0))


def _sample_reference_pixel(x: float, y: float) -> tuple[int, int, int, int]:
    if not _inside_rounded_rect(x, y, 8.0, 8.0, 248.0, 248.0, 42.0):
        return TRANSPARENT
    inner = _inside_rounded_rect(x, y, 13.0, 13.0, 243.0, 243.0, 38.0)
    if inner:
        tile = (8, 14, 20, 88)
    else:
        rim = _mix(BLUE, VIOLET, 0.42)
        tile = (rim[0], rim[1], rim[2], 184)
    return _composite(_source_mark_pixel(x, y), tile)

def _sample_brandmark_pixel(x: float, y: float) -> tuple[int, int, int, int]:
    """Approved Ghost FTP mark extracted from the supplied brand reference."""
    return _source_mark_pixel(x, y)

def _render_rgba(size: int, sampler=_sample_reference_pixel) -> bytes:
    scale = CANVAS / float(size)
    ss = SUPERSAMPLE
    out = bytearray(size * size * 4)
    for py in range(size):
        for px in range(size):
            accum = [0, 0, 0, 0]
            for sy in range(ss):
                for sx in range(ss):
                    x = (px + (sx + 0.5) / ss) * scale
                    y = (py + (sy + 0.5) / ss) * scale
                    sample = sampler(x, y)
                    for i, value in enumerate(sample):
                        accum[i] += value
            base = (py * size + px) * 4
            samples = ss * ss
            for i in range(4):
                out[base + i] = round(accum[i] / samples)
    return bytes(out)


def _png_chunk(kind: bytes, payload: bytes) -> bytes:
    return (
        struct.pack(">I", len(payload))
        + kind
        + payload
        + struct.pack(">I", binascii.crc32(kind + payload) & 0xFFFFFFFF)
    )


def _png_bytes(size: int, sampler=_sample_reference_pixel) -> bytes:
    rgba = _render_rgba(size, sampler)
    scanlines = bytearray()
    stride = size * 4
    for row in range(size):
        scanlines.append(0)
        start = row * stride
        scanlines.extend(rgba[start : start + stride])
    ihdr = struct.pack(">IIBBBBB", size, size, 8, 6, 0, 0, 0)
    return (
        PNG_SIGNATURE
        + _png_chunk(b"IHDR", ihdr)
        + _png_chunk(b"IDAT", zlib.compress(bytes(scanlines), 9))
        + _png_chunk(b"IEND", b"")
    )


def _ico_bytes(sampler=_sample_reference_pixel) -> bytes:
    sizes = (16, 24, 32, 48, 64, 96, 128, 256)
    images = [_png_bytes(size, sampler) for size in sizes]
    header = ICO_SIGNATURE + struct.pack("<H", len(images))
    directory = bytearray()
    offset = 6 + len(images) * 16
    for size, image in zip(sizes, images):
        width = 0 if size == 256 else size
        height = 0 if size == 256 else size
        directory.extend(
            struct.pack("<BBBBHHII", width, height, 0, 0, 1, 32, len(image), offset)
        )
        offset += len(image)
    return header + bytes(directory) + b"".join(images)


def materialize() -> None:
    ICON_PNG.parent.mkdir(parents=True, exist_ok=True)
    ICON_PNG.write_bytes(_png_bytes(256))
    ICON_ICO.write_bytes(_ico_bytes())
    BRANDMARK_PNG.write_bytes(_png_bytes(256, _sample_brandmark_pixel))
    BRANDMARK_ICO.write_bytes(_ico_bytes(_sample_brandmark_pixel))


def require_file(path: Path, minimum_size: int = 1) -> bytes:
    if not path.is_file():
        raise ValueError(f"missing brand asset: {path.relative_to(ROOT)}")
    data = path.read_bytes()
    if len(data) < minimum_size:
        raise ValueError(f"brand asset is unexpectedly small: {path.relative_to(ROOT)}")
    return data


def validate() -> None:
    png = require_file(ICON_PNG, 1024)
    if not png.startswith(PNG_SIGNATURE):
        raise ValueError("build/icon.png is not a valid PNG asset")
    ico = require_file(ICON_ICO, 1024)
    if not ico.startswith(ICO_SIGNATURE):
        raise ValueError("build/icon.ico is not a valid Windows icon asset")
    brandmark_png = require_file(BRANDMARK_PNG, 1024)
    if not brandmark_png.startswith(PNG_SIGNATURE):
        raise ValueError("build/brandmark.png is not a valid PNG asset")
    brandmark_ico = require_file(BRANDMARK_ICO, 1024)
    if not brandmark_ico.startswith(ICO_SIGNATURE):
        raise ValueError("build/brandmark.ico is not a valid Windows icon asset")
    if (ROOT / "GhostFTP WEB").exists():
        raise ValueError("retired Web/PWA application surface is present")


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate or validate Ghost FTP desktop brand assets")
    parser.add_argument("--materialize", action="store_true", help="write deterministic cyan/blue/violet PNG/ICO assets before validation")
    parser.add_argument("--check", action="store_true", help="validate the current assets without rewriting them")
    args = parser.parse_args()

    try:
        if args.materialize:
            materialize()
        validate()
    except (OSError, UnicodeError, ValueError) as exc:
        print(f"BRAND_ASSET_AUDIT=FAILED: {exc}", file=sys.stderr)
        return 1

    print("BRAND_ASSET_AUDIT=PASS")
    print("PUBLIC_BRAND=Ghost FTP")
    print("CANONICAL_LOGO=GHOST_TRANSFER_CYAN_BLUE_VIOLET")
    print("ACTIVE_BRAND_ASSETS=WINDOWS,LINUX,MACOS,ANDROID")
    print("RETIRED_WEB_PWA_ASSETS=BLOCKED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

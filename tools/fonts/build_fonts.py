#!/usr/bin/env python3
"""Rebuild the two self-hosted variable WOFF2 fonts of web/static/fonts (plan 10.6).

The build never runs this script; it documents how the committed files were made.
Requirements: pip install fonttools brotli

Usage:
    python tools/fonts/build_fonts.py [--hanken PATH] [--geist PATH] [--out DIR]

Without --hanken/--geist the upstream variable TTFs are downloaded from the
google/fonts repository. The size and SHA-256 of the output depend on the
upstream source revision and the fontTools version; when they differ from the
committed files, update plan 2.2 T12 and web/assets_test.go together.
"""

from __future__ import annotations

import argparse
import hashlib
import subprocess
import sys
import tempfile
import urllib.request
from pathlib import Path

GOOGLE_FONTS_RAW = "https://raw.githubusercontent.com/google/fonts/main/"

# Basic Latin, Latin-1, Latin Extended-A (o/u with double acute), dashes, quotes,
# arrows and a few math signs.
UNICODES = ",".join(
    [
        "U+0020-007E",
        "U+00A0-00FF",
        "U+0100-017F",
        "U+2010-2015",
        "U+2018-201E",
        "U+2020-2022",
        "U+2026",
        "U+2030",
        "U+2032-2033",
        "U+2039-203A",
        "U+20AC",
        "U+2122",
        "U+2190-2193",
        "U+2212",
        "U+00D7",
        "U+2260",
        "U+2264-2265",
        "U+2713",
    ]
)

FONTS = [
    {
        "key": "hanken",
        "source": "ofl/hankengrotesk/HankenGrotesk[wght].ttf",
        "output": "HankenGrotesk-Variable.woff2",
        "features": "kern,liga,calt,ccmp,locl,mark,mkmk,tnum,lnum,case,ss01",
    },
    {
        "key": "geist",
        "source": "ofl/geistmono/GeistMono[wght].ttf",
        "output": "GeistMono-Variable.woff2",
        "features": "kern,liga,calt,ccmp,locl,mark,mkmk,tnum,lnum,case,zero,ss01",
    },
]

# Hungarian letters that must survive subsetting: o/u with double acute, both cases.
REQUIRED_CODEPOINTS = (0x0150, 0x0151, 0x0170, 0x0171)


def download(relative: str, target_dir: Path) -> Path:
    """Downloads one upstream TTF from the google/fonts repository."""
    url = GOOGLE_FONTS_RAW + relative.replace("[", "%5B").replace("]", "%5D")
    target = target_dir / Path(relative).name
    print(f"download {url}")
    with urllib.request.urlopen(url, timeout=60) as response:
        target.write_bytes(response.read())
    return target


def subset(source: Path, output: Path, features: str) -> None:
    """Subsets the variable TTF into a WOFF2 with the full wght axis kept."""
    subprocess.run(
        [
            sys.executable,
            "-m",
            "fontTools.subset",
            str(source),
            f"--unicodes={UNICODES}",
            "--flavor=woff2",
            "--no-hinting",
            "--desubroutinize",
            f"--layout-features={features}",
            f"--output-file={output}",
        ],
        check=True,
    )


def verify(output: Path) -> None:
    """Checks the acceptance rules: variable wght 100-900 and the required glyphs."""
    from fontTools.ttLib import TTFont

    font = TTFont(str(output))
    if "fvar" not in font:
        raise SystemExit(f"{output.name}: not a variable font")
    axes = {axis.axisTag: (axis.minValue, axis.maxValue) for axis in font["fvar"].axes}
    if axes.get("wght") != (100, 900):
        raise SystemExit(f"{output.name}: wght axis is {axes.get('wght')}, want (100, 900)")
    cmap = font.getBestCmap()
    missing = [f"U+{cp:04X}" for cp in REQUIRED_CODEPOINTS if cp not in cmap]
    if missing:
        raise SystemExit(f"{output.name}: missing glyphs {', '.join(missing)}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--hanken", type=Path, help="local HankenGrotesk[wght].ttf")
    parser.add_argument("--geist", type=Path, help="local GeistMono[wght].ttf")
    parser.add_argument(
        "--out",
        type=Path,
        default=Path(__file__).resolve().parents[2] / "web" / "static" / "fonts",
        help="output directory (default: web/static/fonts)",
    )
    args = parser.parse_args()
    args.out.mkdir(parents=True, exist_ok=True)

    with tempfile.TemporaryDirectory(prefix="kinex-mark-fonts-") as tmp:
        for spec in FONTS:
            source = getattr(args, spec["key"]) or download(spec["source"], Path(tmp))
            output = args.out / spec["output"]
            subset(source, output, spec["features"])
            verify(output)
            data = output.read_bytes()
            digest = hashlib.sha256(data).hexdigest()
            print(f"{output.name}: {len(data)} bytes, sha256 {digest}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

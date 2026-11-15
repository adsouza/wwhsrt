"""Reproduce the offline font (not needed to build/play).

Usage: python3 tools/prepare-skeleton-assets.py /path/to/NotoSansSC[wght].ttf
Requires fonttools==4.54.1. Fetch the source from the URL in assets/fonts/README.md.
"""

import hashlib
from pathlib import Path
import sys
from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont
from fontTools import subset

root = Path(__file__).resolve().parents[1]
source = Path(sys.argv[1]).read_bytes()
expected = "a3041811a78c361b1de50f953c805e0244951c21c5bd412f7232ef0d899af0da"
if hashlib.sha256(source).hexdigest() != expected:
    raise SystemExit("Font source hash differs; review provenance before regenerating")

font = instantiateVariableFont(TTFont(sys.argv[1]), {"wght": 450}, inplace=True)
text = "".join((root / p).read_text() for p in ["pkg/ui/en.json", "pkg/ui/zh-Hans.json"])
for a in range(0xA1, 0xF8):
    for b in range(0xA1, 0xFF):
        try:
            text += bytes([a, b]).decode("gb2312")
        except UnicodeDecodeError:
            pass
opts = subset.Options()
opts.name_IDs = ["*"]
opts.name_legacy = True
sub = subset.Subsetter(options=opts)
sub.populate(text=text + "".join(chr(i) for i in range(32, 384)) + "−→")
sub.subset(font)
font.save(root / "assets/fonts/rail-ui.ttf")

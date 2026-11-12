# Offline UI font

`rail-ui.ttf` is a static weight-450 subset of [Noto Sans SC](https://github.com/google/fonts/tree/main/ofl/notosanssc), distributed under the [SIL Open Font License](OFL.txt). Both the native map/HUD and browser controls bundle it; no font CDN is needed.

The subset includes Latin characters, GB2312's Simplified Chinese repertoire, and every character in the two current interface catalogs. Player names remain Unicode strings; characters outside this font's repertoire may display a missing-glyph box in the native client.

Source: `https://raw.githubusercontent.com/google/fonts/main/ofl/notosanssc/NotoSansSC%5Bwght%5D.ttf`. The exact source SHA-256 is pinned in [the preparation script](../../tools/prepare-skeleton-assets.py). Regenerate using Python with `fonttools==4.54.1`; source changes require an explicit provenance review. The generated font is checked in so building does not need Python or a font download.

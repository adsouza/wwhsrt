# First-business development scenario

See [the scenario and financial assumptions](../../docs/validation/FIRST_BUSINESS.md#scenario-and-financial-assumptions) for the 12-city roster, four corridor markets, construction alternatives, pricing and sources. The original nine metros retain their supplied coordinates. Tsuruga, Shin-Hakodate-Hokuto and Sapporo use approximate display reference points; polylines are illustrative and cannot price engineering by screen distance.

`business.json` freezes the authored engineering coverage, runtime inputs, outside-option assumptions and alignment sketches. `scenario.json` freezes city identity. `camera.json` retains the East Asia landscape/portrait anchors from the [camera policy](../../data/prototypes/map-region-presets.v1.json). The separate camera and Natural Earth display outline do not enter financial distance calculations.

`land.json` derives exterior rings from the repository public-domain Natural Earth 110m snapshot using `tools/prepare-skeleton-assets.py`. The source outline is too coarse to qualify construction. These scenario estimates support the development loop; production engineering qualification remains required.

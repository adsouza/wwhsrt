# Launch construction inputs

`launch-prospects.v1.json` authors the 71 pairs in
[`LAUNCH_CORRIDORS.md`](../../docs/LAUNCH_CORRIDORS.md). The two original Japanese
projects retain their existing alternatives; the compiler produces 69 additional
strategic game alignments in `compiled-prospects.v1.json`.

`route-land.v1.json.gz` pins Natural Earth 1:10m country and lake geometry. Its
embedded `source_urls`, source hashes and public-domain license identify the
inputs. The land mask subtracts lakes; country geometry apportions construction
work to the regional rates in `../costs/construction_costs.v1.json`. It is a
build-only cartographic screen, not a terrain elevation or engineering survey.
Named crossing envelopes and authored terrain shares remain game assumptions.

From the repository root, with the dependencies in
`tools/requirements-public-network.txt` installed:

```sh
python3 tools/prepare-launch-prospects.py
python3 tools/import-public-network.py
python3 tools/check-launch-prospects.py
```

Run these sequentially under the memory limits in [`TESTING.md`](../../TESTING.md).
Both preparation and import support `--check` for reproducibility without writing
runtime artifacts. Preparation reads the immutable v7 catalog in
`assets/scenario/public-network-v3.json.gz` for endpoint coordinates, so a clean
checkout does not depend on a previously generated current catalog. The optional
geometry cache under `build/prospect-catalog` is created automatically; cache
entries include the authoring, endpoint, mask and compiler identities and are
validated again on use. Neither preparation nor import accesses company saves.

The importer combines these projects with the repaired public network and
generates future through-service templates. The independent checker admits paths
against geographic, Equal Earth and actually drawn basemap geometry, checks
crossing budgets and country work totals, and rejects deliberate ocean-shortcut
and unpriced-crossing fixtures. See the
[current implementation and verification report](../../docs/validation/LAUNCH_PROSPECTS.md)
for infrastructure classifications, compatibility and qualification limits.

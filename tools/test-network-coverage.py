"""Regressions against the real compiled catalog, including deliberately broken copies."""

import copy, json, pathlib, unittest
from network_coverage import checks, require, require_map

ROOT = pathlib.Path(__file__).resolve().parents[1]


class Coverage(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.catalog = json.loads((ROOT / "assets/scenario/public-network.json").read_bytes())
        cls.manifest = json.loads((ROOT / "data/imports/operating-networks.v2.json").read_bytes())

    def test_all_operating_networks(self):
        self.assertEqual(len(require(self.catalog, self.manifest)), 14)

    def test_nec_mid_corridor_cut_fails_even_with_legacy_express_aliases(self):
        c = copy.deepcopy(self.catalog)
        # Sever the real Philadelphia/Wilmington approach while keeping historical
        # express aliases. Missing physical resources must invalidate those aliases.
        cut = set()
        for co in c["corridors"]:
            if not co["existing"] or co.get("shared_sections"):
                continue
            p = co["alignments"][0]["path"]
            if min(v["lat"] for v in p) < 39.95 < max(v["lat"] for v in p) and any(-77 < v["lon"] < -74 for v in p):
                cut.add(co["id"])
        self.assertTrue(cut)
        c["corridors"] = [co for co in c["corridors"] if co["id"] not in cut]
        self.assertFalse(next(r for r in checks(c, self.manifest) if r["id"] == "nec")["connected"])

    def test_tashkent_isolation_fails(self):
        c = copy.deepcopy(self.catalog)
        c["corridors"] = [co for co in c["corridors"] if "tashkent" not in (co["from_city"], co["to_city"])]
        self.assertFalse(next(r for r in checks(c, self.manifest) if r["id"] == "uzbek-main")["connected"])

    def test_ordered_service_cannot_skip_or_reorder_stop(self):
        for mutation in ("skip", "reverse", "section"):
            c = copy.deepcopy(self.catalog)
            r = next(r for r in c["network"]["routes"] if r["id"] == "network_nec_all_stops")
            if mutation == "skip":
                r["stops"].remove("wilmington")
            elif mutation == "reverse":
                r["cities"][2:4] = reversed(r["cities"][2:4])
            else:
                r["sections"].pop()
            self.assertFalse(next(r for r in checks(c, self.manifest) if r["id"] == "nec")["connected"])

    def test_required_map_stations_and_paths(self):
        m = json.loads((ROOT / "assets/scenario/playable-rail-map.json").read_bytes())
        require_map(self.catalog, m, self.manifest)
        m["station_ids"].remove(self.manifest["hubs"][0]["station_id"])
        with self.assertRaises(AssertionError):
            require_map(self.catalog, m, self.manifest)

    def test_taipei_and_tainan_cannot_be_skipped(self):
        for city in ["taipei", "tainan"]:
            c = copy.deepcopy(self.catalog)
            c["corridors"] = [co for co in c["corridors"] if city not in (co["from_city"], co["to_city"])]
            self.assertFalse(next(r for r in checks(c, self.manifest) if r["id"] == "taiwan-hsr")["connected"])

    def test_new_bridge_station_and_junction_cuts_fail(self):
        for city, network in [("turin", "italian-hsr"), ("figueres", "catalan-hsr"), ("maibara", "tokaido-junction")]:
            c = copy.deepcopy(self.catalog)
            c["corridors"] = [co for co in c["corridors"] if city not in (co["from_city"], co["to_city"])]
            self.assertFalse(next(r for r in checks(c, self.manifest) if r["id"] == network)["connected"])

    def test_island_networks_remain_separate(self):
        adj = {}
        for co in self.catalog["corridors"]:
            if co["existing"]:
                a, b = co["from_city"], co["to_city"]
                adj.setdefault(a, set()).add(b)
                adj.setdefault(b, set()).add(a)
        for start, end in [("haikou", "guangzhou"), ("taichung", "shanghai"), ("taipei", "shanghai")]:
            self.assertIn(start, {c["id"] for c in self.catalog["cities"]})
            seen = {start}
            q = [start]
            for n in q:
                for other in adj.get(n, set()) - seen:
                    seen.add(other)
                    q.append(other)
            self.assertNotIn(end, seen)


if __name__ == "__main__":
    unittest.main()

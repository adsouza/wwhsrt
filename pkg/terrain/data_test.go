package terrain

import (
	"testing"
	"testing/fstest"

	"wwhsrt/assets"
)

// manifestOnly holds the two JSON files and no tile, as a view that has drawn nothing needs.
func manifestOnly(t *testing.T) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{}
	for _, name := range []string{"terrain/terrain.json", "terrain/features.json"} {
		b, e := assets.Terrain.ReadFile(name)
		if e != nil {
			t.Fatal(e)
		}
		files[name] = &fstest.MapFile{Data: append([]byte{}, b...)}
	}
	return files
}

func TestManifestLoadsWithoutDecodingATile(t *testing.T) {
	s, e := Load(manifestOnly(t))
	if e != nil {
		t.Fatal(e)
	}
	if len(s.Tiles) < 100 || len(s.BasemapSHA) != 64 {
		t.Fatalf("%d tiles, basemap %q", len(s.Tiles), s.BasemapSHA)
	}
	if tile, ok := s.Tiles[Address{8, 224, 101}]; !ok || tile.Kind != Coast || len(tile.SHA) != 64 || len(tile.MaskSHA) != 64 {
		t.Fatalf("the Lake Biwa tile is %+v", tile)
	}
	if len(s.Regions) != 2 || s.Regions[0].ID != "hokkaido" || len(s.Regions[0].Contours) == 0 {
		t.Fatal("Japan's contours are missing")
	}
	if len(s.Features) < 6 {
		t.Fatal("context missing")
	}
	for _, feature := range s.Features {
		if feature.Key == "mt_yotei" {
			if feature.Lat < 42.8 || feature.Lat > 42.9 || feature.Lon < 140.7 || feature.Lon > 140.9 {
				t.Fatal("Yotei label outside its massif")
			}
		}
	}
}

func TestBadManifestsAreRefused(t *testing.T) {
	for name, text := range map[string]string{
		"an old schema":         `{"schema_version":1,"regions":[]}`,
		"a tile off the grid":   `{"schema_version":2,"basemap_sha256":"x","tiles":[[2,4,0,"sea"]],"regions":[]}`,
		"a level out of range":  `{"schema_version":2,"basemap_sha256":"x","tiles":[[9,0,0,"sea"]],"regions":[]}`,
		"an unknown kind":       `{"schema_version":2,"basemap_sha256":"x","tiles":[[2,0,0,"swamp"]],"regions":[]}`,
		"land without a hash":   `{"schema_version":2,"basemap_sha256":"x","tiles":[[2,0,0,"land"]],"regions":[]}`,
		"coast without a mask":  `{"schema_version":2,"basemap_sha256":"x","tiles":[[2,0,0,"coast","aa"]],"regions":[]}`,
		"a tile listed twice":   `{"schema_version":2,"basemap_sha256":"x","tiles":[[2,0,0,"sea"],[2,0,0,"sea"]],"regions":[]}`,
		"a contour off the map": `{"schema_version":2,"basemap_sha256":"x","tiles":[],"regions":[{"id":"a","contours":[{"metres":500,"points":[[95,0],[0,0]]}]}]}`,
		"not JSON":              `broken`,
	} {
		files := manifestOnly(t)
		files["terrain/terrain.json"].Data = []byte(text)
		if _, e := Load(files); e == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

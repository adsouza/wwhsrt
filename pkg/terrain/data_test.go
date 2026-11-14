package terrain

import (
	"math"
	"testing"
	"testing/fstest"
	"wwhsrt/assets"
)

func TestGeoreferencedTerrainLoadsAndProjects(t *testing.T) {
	d, e := Load(assets.Terrain)
	if e != nil {
		t.Fatal(e)
	}
	if len(d.Features) < 6 || len(d.Lakes) != 1 {
		t.Fatal("context missing")
	}
	r := d.Regions[0]
	lat, lon := GeographicPixel(r, 0, 0)
	if math.Abs(lon-139.21875) > 1e-8 || math.Abs(lat-44.087585) > 1e-5 {
		t.Fatal("tile origin misplaced", lat, lon)
	}
	lat2, lon2 := GeographicPixel(r, float64(r.Width), float64(r.Height))
	if lat2 >= lat || lon2 <= lon {
		t.Fatal("terrain axes reversed")
	}
	for _, feature := range d.Features {
		if feature.Key == "mt_yotei" {
			if feature.Lat < 42.8 || feature.Lat > 42.9 || feature.Lon < 140.7 || feature.Lon > 140.9 {
				t.Fatal("Yotei label outside its massif")
			}
		}
	}
}
func TestBadTerrainCannotLoadAsFlatLand(t *testing.T) {
	files := fstest.MapFS{}
	for _, name := range []string{
		"terrain/terrain.json",
		"terrain/features.json",
		"terrain/hokkaido.png",
		"terrain/kansai.png",
	} {
		b, e := assets.Terrain.ReadFile(name)
		if e != nil {
			t.Fatal(e)
		}
		files[name] = &fstest.MapFile{Data: append([]byte{}, b...)}
	}
	files["terrain/hokkaido.png"].Data = []byte("broken")
	if _, e := Load(files); e == nil {
		t.Fatal("corrupt terrain accepted")
	}
}

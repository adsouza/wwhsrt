package terrain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
	"testing/fstest"

	"wwhsrt/assets"
)

func hash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// coastTile is a set with one coast tile whose west half is land, and the files that hold it.
func coastTile(t *testing.T) (*Set, fstest.MapFS, Address) {
	t.Helper()
	colour := image.NewRGBA(image.Rect(0, 0, 256, 256))
	mask := image.NewGray(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			colour.SetRGBA(x, y, color.RGBA{200, 100, 50, 255})
			if x < 128 {
				mask.SetGray(x, y, color.Gray{255})
			}
		}
	}
	var jpg, msk bytes.Buffer
	if err := jpeg.Encode(&jpg, colour, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&msk, mask); err != nil {
		t.Fatal(err)
	}
	a := Address{8, 1, 2}
	files := fstest.MapFS{
		Path(a) + ".jpg": &fstest.MapFile{Data: jpg.Bytes()},
		Path(a) + ".png": &fstest.MapFile{Data: msk.Bytes()},
	}
	s := &Set{Tiles: map[Address]Tile{a: {Kind: Coast, SHA: hash(jpg.Bytes()), MaskSHA: hash(msk.Bytes())}}}
	return s, files, a
}

func near(a, b uint8) bool { return int(a)-int(b) <= 4 && int(b)-int(a) <= 4 }

func TestDecodeAppliesTheMaskAndLandOpacity(t *testing.T) {
	s, files, a := coastTile(t)
	img, err := s.Decode(files, a)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds() != image.Rect(0, 0, 256, 256) {
		t.Fatal("wrong size", img.Bounds())
	}
	// Land is 225/255 opaque and its colour is premultiplied: 200, 100, 50 become 176, 88, 44.
	land := img.RGBAAt(20, 100)
	if land.A != 225 || !near(land.R, 176) || !near(land.G, 88) || !near(land.B, 44) {
		t.Errorf("land pixel is %+v", land)
	}
	if sea := img.RGBAAt(200, 100); sea != (color.RGBA{}) {
		t.Errorf("sea pixel is %+v, want fully clear", sea)
	}
	// An all-land tile has no mask and every pixel is land.
	s.Tiles[a] = Tile{Kind: Land, SHA: s.Tiles[a].SHA}
	if img, err = s.Decode(files, a); err != nil || img.RGBAAt(200, 100).A != 225 {
		t.Errorf("all-land tile: %v, %+v", err, img)
	}
}

func TestDecodeRejectsATileThatFailsItsHash(t *testing.T) {
	for _, suffix := range []string{".jpg", ".png"} {
		s, files, a := coastTile(t)
		data := files[Path(a)+suffix].Data
		data[len(data)/2] ^= 1
		if _, err := s.Decode(files, a); err == nil {
			t.Errorf("a tile whose %s was altered was accepted", suffix)
		}
	}
	s, files, a := coastTile(t)
	delete(files, Path(a)+".png")
	if _, err := s.Decode(files, a); err == nil {
		t.Error("a coast tile with no mask file was accepted")
	}
}

func TestDecodeRefusesSeaAndUnknownTiles(t *testing.T) {
	s, files, a := coastTile(t)
	if _, err := s.Decode(files, Address{8, 9, 9}); err == nil {
		t.Error("a tile that is not in the set was decoded")
	}
	s.Tiles[a] = Tile{Kind: Sea}
	if _, err := s.Decode(files, a); err == nil {
		t.Error("a sea tile was decoded")
	}
}

func TestPath(t *testing.T) {
	if got := Path(Address{8, 227, 93}); got != "terrain/tiles/8/227/93" {
		t.Fatal(got)
	}
}

// Relief is cut to the drawn basemap: Lake Biwa is clear and the shore beside it is land.
func TestACommittedCoastTileLeavesTheLakeClear(t *testing.T) {
	s, err := Load(assets.Terrain)
	if err != nil {
		t.Fatal(err)
	}
	a := Address{8, 224, 101}
	if s.Tiles[a].Kind != Coast {
		t.Fatalf("the tile holding Lake Biwa is %v, want a coast tile", s.Tiles[a].Kind)
	}
	img, err := s.Decode(assets.Terrain, a)
	if err != nil {
		t.Fatal(err)
	}
	if lake := img.RGBAAt(200, 35); lake.A != 0 { // 136.10 E, 35.30 N: the middle of the lake
		t.Errorf("Lake Biwa is drawn as land: %+v", lake)
	}
	if shore := img.RGBAAt(250, 35); shore.A != 225 { // 136.37 E: the plain east of it
		t.Errorf("the plain beside the lake is %+v, want land", shore)
	}
}

func BenchmarkDecode(b *testing.B) {
	s, err := Load(assets.Terrain)
	if err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		if _, err := s.Decode(assets.Terrain, Address{8, 224, 101}); err != nil {
			b.Fatal(err)
		}
	}
}

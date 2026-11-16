// Package terrain loads display context independently of the railway economy.
package terrain

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

// Kind says what a tile holds.
type Kind uint8

const (
	// Land tiles are land in every pixel and have only a colour image.
	Land Kind = iota
	// Coast tiles are part land and carry a one-bit mask beside their colour image.
	Coast
	// Sea tiles have no land and no files. They are listed so that a line over open water is known to
	// be covered, and so that nothing coarser is drawn in their place.
	Sea
)

func (k Kind) String() string { return [...]string{"land", "coast", "sea"}[k] }

// Tile is one listed tile: its kind and the SHA-256 of its files.
type Tile struct {
	Kind         Kind
	SHA, MaskSHA string
}

type Contour struct {
	Metres int          `json:"metres"`
	Points [][2]float64 `json:"points"` // latitude, longitude
}

// Region is one of the two areas of Japan that keep contour lines.
type Region struct {
	ID       string    `json:"id"`
	Contours []Contour `json:"contours"`
}
type Feature struct {
	Key    string  `json:"key"`
	Kind   string  `json:"kind"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Region string  `json:"region"`
}

// Set is the relief the game ships: which tiles exist, and the hand-made context for Japan.
type Set struct {
	Tiles map[Address]Tile
	// BasemapSHA is the SHA-256 of the land outline the tiles were cut to.
	BasemapSHA string
	// Complete is false for a sample that covers only some lines.
	Complete bool
	Regions  []Region
	Features []Feature
}

// Load reads the manifest and the features. It reads and decodes no tile.
func Load(files fs.FS) (*Set, error) {
	b, e := fs.ReadFile(files, "terrain/terrain.json")
	if e != nil {
		return nil, e
	}
	var manifest struct {
		Schema   int      `json:"schema_version"`
		Basemap  string   `json:"basemap_sha256"`
		Complete bool     `json:"complete"`
		Tiles    [][]any  `json:"tiles"`
		Regions  []Region `json:"regions"`
	}
	if e = json.Unmarshal(b, &manifest); e != nil {
		return nil, e
	}
	if manifest.Schema != 2 || manifest.Basemap == "" {
		return nil, fmt.Errorf("invalid terrain manifest")
	}
	s := &Set{
		Tiles:      make(map[Address]Tile, len(manifest.Tiles)),
		BasemapSHA: manifest.Basemap,
		Complete:   manifest.Complete,
		Regions:    manifest.Regions,
	}
	for _, entry := range manifest.Tiles {
		a, tile, e := listed(entry)
		if e != nil {
			return nil, e
		}
		if s.has(a) {
			return nil, fmt.Errorf("terrain tile %v is listed twice", a)
		}
		s.Tiles[a] = tile
	}
	for _, r := range s.Regions {
		for _, line := range r.Contours {
			if len(line.Points) < 2 {
				return nil, fmt.Errorf("invalid contour")
			}
			for _, p := range line.Points {
				if !valid(p[0], p[1]) {
					return nil, fmt.Errorf("invalid contour coordinate")
				}
			}
		}
	}
	b, e = fs.ReadFile(files, "terrain/features.json")
	if e != nil {
		return nil, e
	}
	var features struct {
		Features []Feature `json:"features"`
	}
	if e = json.Unmarshal(b, &features); e != nil {
		return nil, e
	}
	s.Features = features.Features
	for _, f := range s.Features {
		if !valid(f.Lat, f.Lon) {
			return nil, fmt.Errorf("invalid terrain feature")
		}
	}
	return s, nil
}

// listed reads one manifest entry: [level, column, row, kind], then the colour image's hash unless the
// tile is sea, then the mask's hash if it is a coast tile.
func listed(entry []any) (Address, Tile, error) {
	bad := func() (Address, Tile, error) { return Address{}, Tile{}, fmt.Errorf("invalid terrain tile %v", entry) }
	if len(entry) < 4 {
		return bad()
	}
	var n [3]int
	for i := range n {
		f, ok := entry[i].(float64)
		if !ok || f != float64(int(f)) {
			return bad()
		}
		n[i] = int(f)
	}
	a := Address{n[0], n[1], n[2]}
	if a.Z < MinLevel || a.Z > MaxLevel || a.X < 0 || a.Y < 0 || a.X >= 1<<a.Z || a.Y >= 1<<a.Z {
		return bad()
	}
	text := make([]string, len(entry)-3)
	for i := range text {
		var ok bool
		if text[i], ok = entry[i+3].(string); !ok || text[i] == "" {
			return bad()
		}
	}
	switch {
	case text[0] == "sea" && len(text) == 1:
		return a, Tile{Kind: Sea}, nil
	case text[0] == "land" && len(text) == 2:
		return a, Tile{Kind: Land, SHA: text[1]}, nil
	case text[0] == "coast" && len(text) == 3:
		return a, Tile{Kind: Coast, SHA: text[1], MaskSHA: text[2]}, nil
	}
	return bad()
}

func valid(lat, lon float64) bool { return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 }

// Package terrain loads display context independently of the railway economy.
package terrain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"io/fs"
	"math"
	"wwhsrt/pkg/mapview"
)

type Contour struct {
	Metres int          `json:"metres"`
	Points [][2]float64 `json:"points"`
}
type Region struct {
	ID       string      `json:"id"`
	File     string      `json:"file"`
	SHA      string      `json:"sha256"`
	Zoom     int         `json:"zoom"`
	X0       int         `json:"x0"`
	Y0       int         `json:"y0"`
	TilesX   int         `json:"tiles_x"`
	TilesY   int         `json:"tiles_y"`
	Width    int         `json:"width"`
	Height   int         `json:"height"`
	Contours []Contour   `json:"contours"`
	Image    image.Image `json:"-"`
}
type Feature struct {
	Key    string  `json:"key"`
	Kind   string  `json:"kind"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Region string  `json:"region"`
}
type Lake struct {
	Key    string       `json:"key"`
	Points [][2]float64 `json:"points"`
}
type Data struct {
	Regions  []Region `json:"regions"`
	Features []Feature
	Lakes    []Lake
}

// GeographicPixel converts a source pixel position to geographic coordinates.
// The renderer then applies Equal Earth; source tile projection is never a camera.
func GeographicPixel(r Region, x, y float64) (lat, lon float64) {
	n := math.Exp2(float64(r.Zoom))
	lon = (float64(r.X0)+x/256)/n*360 - 180
	lat = math.Atan(math.Sinh(math.Pi*(1-2*(float64(r.Y0)+y/256)/n))) * 180 / math.Pi
	return
}
func Load(files fs.FS) (Data, error) {
	var d Data
	b, e := fs.ReadFile(files, "terrain/terrain.json")
	if e != nil {
		return d, e
	}
	var manifest struct {
		Schema  int      `json:"schema_version"`
		Regions []Region `json:"regions"`
	}
	if e = json.Unmarshal(b, &manifest); e != nil {
		return d, e
	}
	if manifest.Schema != 1 || len(manifest.Regions) != 2 {
		return d, fmt.Errorf("invalid terrain manifest")
	}
	d.Regions = manifest.Regions
	for i := range d.Regions {
		r := &d.Regions[i]
		if r.Zoom != 8 || r.TilesX < 1 || r.TilesX > 4 || r.TilesY < 1 || r.TilesY > 4 || r.Width != r.TilesX*256 ||
			r.Height != r.TilesY*256 {
			return d, fmt.Errorf("invalid terrain dimensions")
		}
		b, e = fs.ReadFile(files, "terrain/"+r.File)
		if e != nil {
			return d, e
		}
		sum := sha256.Sum256(b)
		if hex.EncodeToString(sum[:]) != r.SHA {
			return d, fmt.Errorf("terrain image checksum mismatch")
		}
		r.Image, _, e = image.Decode(bytes.NewReader(b))
		if e != nil {
			return d, e
		}
		if r.Image.Bounds().Dx() != r.Width || r.Image.Bounds().Dy() != r.Height {
			return d, fmt.Errorf("terrain image size mismatch")
		}
		for _, line := range r.Contours {
			if len(line.Points) < 2 {
				return d, fmt.Errorf("invalid contour")
			}
			for _, p := range line.Points {
				if !valid(p[0], p[1]) {
					return d, fmt.Errorf("invalid contour coordinate")
				}
			}
		}
	}
	b, e = fs.ReadFile(files, "terrain/features.json")
	if e != nil {
		return d, e
	}
	var features struct {
		Features []Feature `json:"features"`
		Lakes    []Lake    `json:"lakes"`
	}
	if e = json.Unmarshal(b, &features); e != nil {
		return d, e
	}
	d.Features = features.Features
	d.Lakes = features.Lakes
	for _, f := range d.Features {
		if !valid(f.Lat, f.Lon) {
			return d, fmt.Errorf("invalid terrain feature")
		}
	}
	return d, nil
}
func valid(lat, lon float64) bool { return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 }
func ProjectedPixel(r Region, x, y float64) mapview.Point {
	lat, lon := GeographicPixel(r, x, y)
	return mapview.Project(lat, lon)
}

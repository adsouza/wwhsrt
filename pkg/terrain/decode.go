package terrain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io/fs"
)

// landAlpha is how opaque relief is over the plain land colour, out of 255.
const landAlpha = 225

// Path is where a tile's files are, without their suffix: ".jpg" for colour, ".png" for a coast tile's mask.
func Path(a Address) string { return fmt.Sprintf("terrain/tiles/%d/%d/%d", a.Z, a.X, a.Y) }

// Decode reads one tile, checks each file against the manifest's hash, and returns it ready to draw:
// premultiplied, with land at landAlpha and everything else fully clear.
func (s *Set) Decode(files fs.FS, a Address) (*image.RGBA, error) {
	tile, ok := s.Tiles[a]
	if !ok || tile.Kind == Sea {
		return nil, fmt.Errorf("terrain tile %v has no image", a)
	}
	read := func(suffix, sum string) ([]byte, error) {
		b, e := fs.ReadFile(files, Path(a)+suffix)
		if e != nil {
			return nil, e
		}
		if got := sha256.Sum256(b); hex.EncodeToString(got[:]) != sum {
			return nil, fmt.Errorf("terrain tile %v%s checksum mismatch", a, suffix)
		}
		return b, nil
	}
	b, e := read(".jpg", tile.SHA)
	if e != nil {
		return nil, e
	}
	colour, e := jpeg.Decode(bytes.NewReader(b))
	if e != nil {
		return nil, e
	}
	size := image.Rect(0, 0, TileSize, TileSize)
	if colour.Bounds() != size {
		return nil, fmt.Errorf("terrain tile %v size mismatch", a)
	}
	out := image.NewRGBA(size)
	draw.Draw(out, size, colour, image.Point{}, draw.Src)
	var mask *image.Gray
	if tile.Kind == Coast {
		if b, e = read(".png", tile.MaskSHA); e != nil {
			return nil, e
		}
		decoded, e := png.Decode(bytes.NewReader(b))
		if e != nil {
			return nil, e
		}
		if decoded.Bounds() != size {
			return nil, fmt.Errorf("terrain tile %v mask size mismatch", a)
		}
		if mask, ok = decoded.(*image.Gray); !ok {
			mask = image.NewGray(size)
			draw.Draw(mask, size, decoded, image.Point{}, draw.Src)
		}
	}
	for i, p := 0, out.Pix; i < len(p); i += 4 {
		if mask != nil && mask.Pix[i/4] < 128 {
			p[i], p[i+1], p[i+2], p[i+3] = 0, 0, 0, 0
			continue
		}
		p[i] = uint8(uint32(p[i]) * landAlpha / 255)
		p[i+1] = uint8(uint32(p[i+1]) * landAlpha / 255)
		p[i+2] = uint8(uint32(p[i+2]) * landAlpha / 255)
		p[i+3] = landAlpha
	}
	return out, nil
}

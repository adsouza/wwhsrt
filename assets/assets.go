// Package assets bundles all runtime data for offline use.
package assets

import "embed"

//go:embed scenario/scenario.json
var Scenario []byte

//go:embed terrain/land.json
var Land []byte

//go:embed scenario/camera.json
var Camera []byte

//go:embed scenario/business.json
var Business []byte

//go:embed fonts/rail-ui.ttf
var Font []byte

// Terrain contains presentation-only relief. It never enters campaign hashes.
//
//go:embed terrain/*.json terrain/*.png terrain/ATTRIBUTION.md
var Terrain embed.FS

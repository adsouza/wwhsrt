// Package assets bundles all runtime data for offline use.
package assets

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"sync"
)

//go:embed scenario/scenario.json
var Scenario []byte

//go:embed scenario/camera.json
var Camera []byte

//go:embed scenario/business.json
var Business []byte

//go:embed scenario/network.json
var Network []byte

//go:embed scenario/public-network.json
var PublicNetwork []byte

// PublicNetworkSHA256 is the lowercase hex digest of PublicNetwork, computed
// once for every package that pins data to the catalog.
var PublicNetworkSHA256 = sync.OnceValue(func() string {
	sum := sha256.Sum256(PublicNetwork)
	return hex.EncodeToString(sum[:])
})

//go:embed scenario/public-network-v1.json
var PublicNetworkV1 []byte

//go:embed scenario/public-network-v2.json.gz
var PublicNetworkV2Compressed []byte

//go:embed scenario/public-network-v3.json.gz
var PublicNetworkV3Compressed []byte

//go:embed scenario/public-network-v4.json.gz
var PublicNetworkV4Compressed []byte

//go:embed scenario/public-network-v5.json.gz
var PublicNetworkV5Compressed []byte

// RailDisplayPaths corrects legacy drawing/animation without changing save inputs.
//
//go:embed scenario/rail-display-paths.json
var RailDisplayPaths []byte

// PlayableRailMap excludes source fragments unused by the operating catalog.
//
//go:embed scenario/playable-rail-map.json
var PlayableRailMap []byte

//go:embed scenario/regions.json
var Regions []byte

//go:embed fonts/rail-ui.ttf
var Font []byte

// Terrain contains presentation-only relief. It never enters campaign hashes.
//
//go:embed terrain/*.json terrain/tiles terrain/ATTRIBUTION.md
var Terrain embed.FS

//go:embed scenario/economy.json
var Economy []byte

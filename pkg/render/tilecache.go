package render

import (
	"image"
	"log"
	"sync"

	"wwhsrt/pkg/terrain"
)

const (
	// tileCap is the most relief tiles kept on the GPU beyond what a scene needs: 288 tiles of 256x256
	// are 75 MB. A 3737x2029 map shows about 170 at most.
	tileCap = 288
	// tileFloor is the least the cache keeps, however small the view.
	tileFloor = 64
	// uploadsPerFrame bounds the work a frame does for tiles that have just been decoded.
	uploadsPerFrame = 8
)

// texture is a tile on the GPU. Go's collector does not see GPU memory, so the cache frees each one itself.
type texture interface{ Deallocate() }

type heldTile struct {
	texture texture
	scene   uint64 // the last scene that drew it
}

type decodedTile struct {
	address terrain.Address
	image   *image.RGBA
}

// tileCache loads relief tiles as a view asks for them and keeps a number that follows the view.
//
// One goroutine decodes, and frame takes a few decoded tiles in at a time, so a native pan never waits
// on decoding. (In the browser there is one thread: decoding runs between frames instead.) The decoded
// image is dropped once it is on the GPU.
//
// The map draws a scene once and then shows it from a cached image until something changes, so "in
// use" means used by the scene on screen, not drawn this frame. Those tiles are never freed: the next
// redraw would find them gone and show plain land while they loaded again. Beyond them the cache keeps
// up to twice what the scene used, between tileFloor and its limit, and frees the tiles whose last
// scene is oldest. Memory therefore follows the window, not how far the player has panned.
//
// ready, get, want, beginScene, frame, len and close belong to the drawing goroutine.
type tileCache struct {
	limit  int
	decode func(terrain.Address) (*image.RGBA, error)
	upload func(*image.RGBA) texture

	held  map[terrain.Address]*heldTile
	scene uint64 // counts scenes drawn
	used  int    // tiles the current scene has drawn

	mu      sync.Mutex
	queue   []terrain.Address        // wanted and not yet started, coarsest first
	pending map[terrain.Address]bool // being decoded, or decoded and waiting for a frame
	failed  map[terrain.Address]bool
	wake    chan struct{}
	stop    chan struct{}
	stopped bool
	results chan decodedTile
	worker  sync.WaitGroup
}

func newTileCache(limit int, decode func(terrain.Address) (*image.RGBA, error), upload func(*image.RGBA) texture) *tileCache {
	c := &tileCache{
		limit:   limit,
		decode:  decode,
		upload:  upload,
		held:    map[terrain.Address]*heldTile{},
		pending: map[terrain.Address]bool{},
		failed:  map[terrain.Address]bool{},
		wake:    make(chan struct{}, 1),
		stop:    make(chan struct{}),
		results: make(chan decodedTile, uploadsPerFrame),
	}
	c.worker.Add(1)
	go c.load()
	return c
}

// load decodes wanted tiles one at a time until the cache is closed.
func (c *tileCache) load() {
	defer c.worker.Done()
	for {
		c.mu.Lock()
		var next terrain.Address
		found := len(c.queue) > 0
		if found {
			next, c.queue = c.queue[0], c.queue[1:]
			c.pending[next] = true
		}
		c.mu.Unlock()
		if !found {
			select {
			case <-c.wake:
				continue
			case <-c.stop:
				return
			}
		}
		img, err := c.decode(next)
		if err != nil {
			log.Printf("terrain tile %v could not be loaded: %v", next, err)
			c.mu.Lock()
			delete(c.pending, next)
			c.failed[next] = true
			c.mu.Unlock()
			continue
		}
		select {
		case c.results <- decodedTile{next, img}:
		case <-c.stop:
			return
		}
	}
}

func (c *tileCache) ready(a terrain.Address) bool { return c.held[a] != nil }

// beginScene starts a redraw of the scene. Tiles got from here on are the ones on screen.
func (c *tileCache) beginScene() {
	c.scene++
	c.used = 0
}

// get returns a ready tile and notes that the scene being drawn uses it; nil if it is not ready.
func (c *tileCache) get(a terrain.Address) texture {
	tile := c.held[a]
	if tile == nil {
		return nil
	}
	if tile.scene != c.scene {
		tile.scene = c.scene
		c.used++
	}
	return tile.texture
}

// cap is how many tiles to keep: twice what the scene on screen used, within the floor and the limit.
func (c *tileCache) cap() int {
	return min(c.limit, max(min(tileFloor, c.limit), 2*c.used))
}

// want replaces the list of tiles to load, coarsest first. Tiles asked for earlier and not yet started
// are dropped: the view has moved on.
func (c *tileCache) want(tiles []terrain.Address) {
	c.mu.Lock()
	c.queue = c.queue[:0]
	for _, a := range tiles {
		if c.held[a] == nil && !c.pending[a] && !c.failed[a] {
			c.queue = append(c.queue, a)
		}
	}
	waiting := len(c.queue) > 0
	c.mu.Unlock()
	if waiting {
		select {
		case c.wake <- struct{}{}:
		default:
		}
	}
}

// frame takes in up to uploadsPerFrame decoded tiles, frees what is over the cap and not on screen, and
// reports whether anything arrived. Call it once per frame, whether or not the scene is redrawn.
func (c *tileCache) frame() (arrived bool) {
	for range uploadsPerFrame {
		select {
		case tile := <-c.results:
			// A new tile belongs to the scene on screen until the redraw its arrival causes; otherwise
			// a full cache would free it at once and ask for it again.
			c.held[tile.address] = &heldTile{c.upload(tile.image), c.scene}
			c.mu.Lock()
			delete(c.pending, tile.address)
			c.mu.Unlock()
			arrived = true
			continue
		default:
		}
		break
	}
	for keep := c.cap(); len(c.held) > keep; {
		var oldest terrain.Address
		found := false
		for a, tile := range c.held {
			if tile.scene < c.scene && (!found || tile.scene < c.held[oldest].scene) {
				oldest, found = a, true
			}
		}
		if !found {
			break
		}
		c.held[oldest].texture.Deallocate()
		delete(c.held, oldest)
	}
	return arrived
}

func (c *tileCache) len() int { return len(c.held) }

// close stops the loader and frees every tile.
func (c *tileCache) close() {
	if !c.stopped {
		c.stopped = true
		close(c.stop)
		c.worker.Wait()
	}
	for a, tile := range c.held {
		tile.texture.Deallocate()
		delete(c.held, a)
	}
}

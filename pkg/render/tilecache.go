package render

import (
	"image"
	"log"
	"sync"

	"wwhsrt/pkg/terrain"
)

const (
	// tileCap is how many relief tiles are kept on the GPU: 288 tiles of 256x256 are 75 MB. A 3737x2029
	// map shows about 170 at most, so a pan has room before anything in view is dropped.
	tileCap = 288
	// uploadsPerFrame bounds the work a frame does for tiles that have just been decoded.
	uploadsPerFrame = 8
)

// texture is a tile on the GPU. Go's collector does not see GPU memory, so the cache frees each one itself.
type texture interface{ Deallocate() }

type heldTile struct {
	texture texture
	drawn   uint64 // the frame it was last drawn in
}

type decodedTile struct {
	address terrain.Address
	image   *image.RGBA
}

// tileCache loads relief tiles as a view asks for them and keeps a fixed number.
//
// One goroutine decodes, so a pan never waits on it; frame takes a few decoded tiles in at a time. The
// decoded image is dropped once it is on the GPU. When more tiles are held than the limit, the ones
// drawn longest ago are freed, but never one drawn since the last frame: the view comes first, and a
// tile freed while in use would only be asked for again.
//
// ready, get, want, frame, len and close belong to the drawing goroutine.
type tileCache struct {
	limit  int
	decode func(terrain.Address) (*image.RGBA, error)
	upload func(*image.RGBA) texture

	held  map[terrain.Address]*heldTile
	stamp uint64

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
		stamp:   1,
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

// get returns a ready tile and notes that it is in use this frame; nil if it is not ready.
func (c *tileCache) get(a terrain.Address) texture {
	tile := c.held[a]
	if tile == nil {
		return nil
	}
	tile.drawn = c.stamp
	return tile.texture
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

// frame takes in up to uploadsPerFrame decoded tiles, frees what is over the limit and unused, and
// reports whether anything arrived. Call it once per drawn frame.
func (c *tileCache) frame() (arrived bool) {
	for range uploadsPerFrame {
		select {
		case tile := <-c.results:
			// A new tile counts as in use, or a full cache would free it at once and ask for it again.
			c.held[tile.address] = &heldTile{c.upload(tile.image), c.stamp}
			c.mu.Lock()
			delete(c.pending, tile.address)
			c.mu.Unlock()
			arrived = true
			continue
		default:
		}
		break
	}
	for len(c.held) > c.limit {
		var oldest terrain.Address
		found := false
		for a, tile := range c.held {
			if tile.drawn < c.stamp && (!found || tile.drawn < c.held[oldest].drawn) {
				oldest, found = a, true
			}
		}
		if !found {
			break
		}
		c.held[oldest].texture.Deallocate()
		delete(c.held, oldest)
	}
	c.stamp++
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

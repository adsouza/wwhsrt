package render

import (
	"errors"
	"image"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"wwhsrt/pkg/terrain"
)

// fakeTexture stands in for a GPU image and records being freed.
type fakeTexture struct{ freed *atomic.Int32 }

func (f fakeTexture) Deallocate() { f.freed.Add(1) }

// cacheUnderTest is a tile cache whose decoding and uploading are counted instead of done.
type cacheUnderTest struct {
	*tileCache
	freed   atomic.Int32
	mu      sync.Mutex
	decoded map[terrain.Address]int
	// gate, when set, holds every decode until it is closed.
	gate chan struct{}
	fail map[terrain.Address]bool
}

func newCacheUnderTest(t *testing.T, limit int) *cacheUnderTest {
	c := &cacheUnderTest{decoded: map[terrain.Address]int{}, fail: map[terrain.Address]bool{}}
	c.tileCache = newTileCache(limit, func(a terrain.Address) (*image.RGBA, error) {
		c.mu.Lock()
		c.decoded[a]++
		gate, fail := c.gate, c.fail[a]
		c.mu.Unlock()
		if gate != nil {
			<-gate
		}
		if fail {
			return nil, errors.New("broken tile")
		}
		return image.NewRGBA(image.Rect(0, 0, 1, 1)), nil
	}, func(*image.RGBA) texture { return fakeTexture{&c.freed} })
	t.Cleanup(c.close)
	return c
}

func (c *cacheUnderTest) decodes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	total := 0
	for _, n := range c.decoded {
		total += n
	}
	return total
}

func tiles(row, count int) []terrain.Address {
	out := make([]terrain.Address, count)
	for i := range out {
		out[i] = terrain.Address{Z: 8, X: i, Y: row}
	}
	return out
}

// until calls step, which usually runs a frame, until done reports true or two seconds pass.
func until(t *testing.T, what string, step func(), done func() bool) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); !done(); {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting until %s", what)
		}
		step()
		time.Sleep(time.Millisecond)
	}
}

func (c *cacheUnderTest) readyCount(tiles []terrain.Address) int {
	n := 0
	for _, a := range tiles {
		if c.ready(a) {
			n++
		}
	}
	return n
}

// scene draws a scene that uses the given tiles, as the map does on a redraw.
func (c *cacheUnderTest) scene(tiles ...terrain.Address) {
	c.beginScene()
	for _, a := range tiles {
		c.get(a)
	}
}

func TestCacheHoldsNoMoreThanItsLimitAndFreesTheRest(t *testing.T) {
	c := newCacheUnderTest(t, 4)
	c.want(tiles(0, 10))
	// Each scene uses none of them: the view has moved on.
	until(t, "ten tiles are decoded", func() { c.frame(); c.scene() }, func() bool { return c.decodes() == 10 })
	until(t, "the cache is back within its limit", func() { c.frame(); c.scene() }, func() bool {
		return c.len() <= 4 && int(c.freed.Load())+c.len() == 10
	})
	if c.len() > 4 || c.freed.Load() != int32(10-c.len()) {
		t.Fatalf("%d tiles held and %d freed after loading 10 with a limit of 4", c.len(), c.freed.Load())
	}
}

// The scene is drawn once and then shown from a cached image for many frames. The tiles it used must
// stay, or the next redraw would find them gone and show plain land while they load again.
func TestCacheKeepsWhatTheSceneOnScreenUsed(t *testing.T) {
	c := newCacheUnderTest(t, 2)
	onScreen := tiles(0, 5) // more than the limit: a very large window
	c.want(onScreen)
	until(t, "five tiles are ready", func() { c.frame() }, func() bool { return c.readyCount(onScreen) == 5 })
	c.scene(onScreen...)
	for range 50 { // frames with no redraw
		c.frame()
	}
	if c.readyCount(onScreen) != 5 || c.freed.Load() != 0 {
		t.Fatalf("%d of 5 tiles on screen are still held, %d freed, after frames with no redraw", c.readyCount(onScreen), c.freed.Load())
	}
	// Tiles that arrive for somewhere else do not push out what is on screen either.
	c.want(tiles(1, 3))
	until(t, "three more are decoded", func() { c.frame() }, func() bool { return c.decodes() == 8 })
	for range 5 {
		c.frame()
	}
	if c.readyCount(onScreen) != 5 {
		t.Fatal("a tile on screen was freed to make room for one that is not")
	}
	// Once a scene no longer uses them, the limit applies again.
	c.scene(onScreen[:2]...)
	c.frame()
	if c.len() != 2 || c.readyCount(onScreen[:2]) != 2 {
		t.Fatalf("%d tiles held after the view shrank to two, with a limit of 2", c.len())
	}
}

// Memory follows the window: a small view keeps a small cache however far the player has panned.
func TestCacheLimitFollowsTheView(t *testing.T) {
	c := newCacheUnderTest(t, 200)
	seen := tiles(0, 150)
	c.want(seen)
	until(t, "150 tiles are ready", func() { c.frame() }, func() bool { return c.readyCount(seen) == 150 })
	c.scene(seen[:10]...) // a phone-sized view
	c.frame()
	if c.len() != tileFloor {
		t.Fatalf("a view of 10 tiles keeps %d, want the floor of %d", c.len(), tileFloor)
	}
	if c.readyCount(seen[:10]) != 10 {
		t.Fatal("the view's own tiles were freed")
	}
	more := tiles(1, 150)
	c.want(more)
	until(t, "150 more are decoded", func() { c.frame(); c.scene(more[:90]...) }, func() bool { return c.readyCount(more[:90]) == 90 })
	c.scene(more[:90]...)
	c.frame()
	if c.len() > 180 || c.len() < 90 {
		t.Fatalf("a view of 90 tiles keeps %d, want at most twice the view", c.len())
	}
	c.scene(more[:150]...) // twice this view would pass the cache's own limit
	c.frame()
	if c.len() > 200 {
		t.Fatalf("%d tiles held past the limit of 200 by a view of 150", c.len())
	}
}

func TestCacheUploadsAFewTilesAFrame(t *testing.T) {
	c := newCacheUnderTest(t, 100)
	all := tiles(0, 40)
	c.want(all)
	// The loader stops once the hand-over is full: that many decoded and waiting, and one more in hand.
	until(t, "the loader is waiting for a frame", func() {}, func() bool { return c.decodes() == uploadsPerFrame+1 })
	time.Sleep(5 * time.Millisecond)
	if c.decodes() != uploadsPerFrame+1 {
		t.Fatalf("the loader ran ahead of the frames: %d tiles decoded", c.decodes())
	}
	if !c.frame() {
		t.Fatal("a frame that took tiles in did not say so")
	}
	if n := c.readyCount(all); n == 0 || n > uploadsPerFrame {
		t.Fatalf("one frame took in %d tiles, want 1 to %d", n, uploadsPerFrame)
	}
	until(t, "all forty are ready", func() { c.frame() }, func() bool { return c.readyCount(all) == 40 })
	if c.frame() {
		t.Fatal("a frame with nothing new claimed an arrival")
	}
}

func TestCacheDropsWhatIsNoLongerWanted(t *testing.T) {
	c := newCacheUnderTest(t, 100)
	gate := make(chan struct{})
	c.mu.Lock()
	c.gate = gate
	c.mu.Unlock()
	first, second := tiles(0, 5), tiles(1, 3)
	c.want(first)
	until(t, "the first tile is being decoded", func() {}, func() bool { return c.decodes() == 1 })
	c.want(second) // the view has moved on
	close(gate)
	until(t, "the second view is ready", func() { c.frame() }, func() bool { return c.readyCount(second) == 3 })
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, a := range first[1:] {
		if c.decoded[a] != 0 {
			t.Errorf("%+v was decoded after the view stopped wanting it", a)
		}
	}
}

func TestCacheDoesNotLoadATileTwice(t *testing.T) {
	c := newCacheUnderTest(t, 100)
	all := tiles(0, 6)
	for range 50 { // the view asks again every frame
		c.want(all)
		c.frame()
		time.Sleep(200 * time.Microsecond)
	}
	until(t, "all six are ready", func() { c.want(all); c.frame() }, func() bool { return c.readyCount(all) == 6 })
	if c.decodes() != 6 {
		t.Fatalf("six tiles took %d decodes", c.decodes())
	}
}

func TestLoaderReportsAFailedTileOnceAndKeepsGoing(t *testing.T) {
	c := newCacheUnderTest(t, 100)
	all := tiles(0, 4)
	bad := all[1]
	c.fail[bad] = true
	good := []terrain.Address{all[0], all[2], all[3]}
	until(t, "the good tiles are ready", func() { c.want(all); c.frame() }, func() bool { return c.readyCount(good) == 3 })
	for range 20 {
		c.want(all)
		c.frame()
		time.Sleep(200 * time.Microsecond)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ready(bad) || c.decoded[bad] != 1 {
		t.Fatalf("the broken tile is ready=%v after %d attempts, want one attempt", c.ready(bad), c.decoded[bad])
	}
}

func TestCloseFreesEverything(t *testing.T) {
	c := newCacheUnderTest(t, 100)
	all := tiles(0, 5)
	c.want(all)
	until(t, "five tiles are ready", func() { c.frame() }, func() bool { return c.readyCount(all) == 5 })
	c.close()
	if c.len() != 0 || c.freed.Load() != 5 {
		t.Fatalf("after close %d tiles are held and %d were freed", c.len(), c.freed.Load())
	}
	c.close() // closing twice is harmless
}

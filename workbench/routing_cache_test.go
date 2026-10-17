package workbench

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func TestRouteCacheConcurrentMissPublicationKeepsOneEntry(t *testing.T) {
	var cache wireRouteCache
	key := sha256.Sum256([]byte("same graph and wire"))
	path := []Point{{X: 10}, {X: 20}}
	start := make(chan struct{})
	var workers sync.WaitGroup
	// These represent computations that all missed before the first finished.
	// A barrier makes the publication invariant independent of routing speed.
	for range 64 {
		workers.Go(func() {
			<-start
			cache.retain(key, path)
		})
	}
	close(start)
	workers.Wait()
	if len(cache.paths) != 1 || len(cache.order) != 1 {
		t.Fatalf("one route consumed multiple cache slots: paths=%d order=%d", len(cache.paths), len(cache.order))
	}
}

func TestRouteCacheLateDuplicateDoesNotEvictAnotherRoute(t *testing.T) {
	var cache wireRouteCache
	keys := [][32]byte{}
	for i := range 513 {
		keys = append(keys, sha256.Sum256([]byte(fmt.Sprint(i))))
	}
	for _, key := range keys[:512] {
		cache.retain(key, []Point{{X: 10}, {X: 20}})
	}
	// The last route's second calculation finishes after other paths filled
	// the cache. It must not consume a slot or prematurely evict the first.
	cache.retain(keys[511], []Point{{X: 10}, {X: 20}})
	if _, ok := cache.paths[keys[0]]; !ok || len(cache.paths) != 512 || len(cache.order) != 512 {
		t.Fatal("late duplicate evicted an unrelated cached route")
	}
	cache.retain(keys[512], []Point{{X: 30}, {X: 40}})
	if _, ok := cache.paths[keys[0]]; ok {
		t.Fatal("a new route did not evict the oldest entry")
	}
	seen := map[[32]byte]bool{}
	for _, key := range cache.order {
		if seen[key] {
			t.Fatal("FIFO retained a duplicate key")
		}
		seen[key] = true
		if _, ok := cache.paths[key]; !ok {
			t.Fatal("FIFO retained a key absent from the cache")
		}
	}
	if len(cache.paths) != 512 || len(cache.order) != 512 {
		t.Fatal("cache capacity changed")
	}
}

func TestWirePathCallerCannotChangeRetainedRoute(t *testing.T) {
	g := ExampleGraph()
	wire := g.Wires[0]
	want := computeWireRoute(g, wire)
	for range 2 {
		got := WirePath(g, wire)
		if !reflect.DeepEqual(got, want) || len(got) < 2 {
			t.Fatal("cache changed the deterministic route")
		}
		got[0].X += 1000
	}
	if !reflect.DeepEqual(WirePath(g, wire), want) {
		t.Fatal("caller changed another renderer/exporter's retained route")
	}
}

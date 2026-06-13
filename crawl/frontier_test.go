package crawl

import (
	"sync"
	"testing"
)

func TestFrontierDedup(t *testing.T) {
	f := newFrontier(false, nil)
	if !f.Add("https://example.com/a", 0) {
		t.Fatal("first Add should report a new URL")
	}
	if f.Add("https://example.com/a", 0) {
		t.Fatal("duplicate Add should report not-new")
	}
	if got := f.Seen(); got != 1 {
		t.Errorf("Seen() = %d, want 1", got)
	}
	if got := f.Len(); got != 1 {
		t.Errorf("Len() = %d, want 1", got)
	}
}

func TestFrontierDedupByKey(t *testing.T) {
	// A key function that ignores the query collapses ?a and ?b onto one.
	key := func(s string) string {
		if i := indexByte(s, '?'); i >= 0 {
			return s[:i]
		}
		return s
	}
	f := newFrontier(false, key)
	if !f.Add("https://example.com/p?a=1", 0) {
		t.Fatal("first Add should be new")
	}
	if f.Add("https://example.com/p?b=2", 0) {
		t.Fatal("same key should dedup")
	}
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func TestFrontierBFSOrder(t *testing.T) {
	f := newFrontier(false, nil)
	f.Add("a", 0)
	f.Add("b", 0)
	f.Add("c", 0)
	got := drainURLs(f)
	want := []string{"a", "b", "c"}
	if !equalStrings(got, want) {
		t.Errorf("BFS order = %v, want %v", got, want)
	}
}

func TestFrontierDFSOrder(t *testing.T) {
	f := newFrontier(true, nil)
	f.Add("a", 0)
	f.Add("b", 0)
	f.Add("c", 0)
	got := drainURLs(f)
	want := []string{"c", "b", "a"}
	if !equalStrings(got, want) {
		t.Errorf("DFS order = %v, want %v", got, want)
	}
}

// drainURLs pops every item, marking each Done immediately, until the frontier
// self-closes. It is the single-worker drain pattern.
func drainURLs(f *frontier) []string {
	var out []string
	for {
		it, ok := f.Next()
		if !ok {
			return out
		}
		out = append(out, it.URL)
		f.Done()
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestFrontierAutoClose is the defining behavior: a worker that discovers more
// work while processing keeps the frontier open, and once the last item is Done
// with nothing queued the frontier closes and Next unblocks every waiter.
func TestFrontierAutoClose(t *testing.T) {
	f := newFrontier(false, nil)
	f.Add("seed", 0)

	var mu sync.Mutex
	var seen []string
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for {
				it, ok := f.Next()
				if !ok {
					return
				}
				// The seed discovers two children; nothing else branches.
				if it.URL == "seed" {
					f.Add("child-1", 1)
					f.Add("child-2", 1)
				}
				mu.Lock()
				seen = append(seen, it.URL)
				mu.Unlock()
				f.Done()
			}
		})
	}
	wg.Wait()

	if len(seen) != 3 {
		t.Fatalf("processed %d items, want 3 (seed + 2 children)", len(seen))
	}
	if got := f.Seen(); got != 3 {
		t.Errorf("Seen() = %d, want 3", got)
	}
}

func TestFrontierCloseStopsAdd(t *testing.T) {
	f := newFrontier(false, nil)
	f.Close()
	if f.Add("https://example.com/a", 0) {
		t.Error("Add after Close should report not-new")
	}
	if _, ok := f.Next(); ok {
		t.Error("Next after Close should report not-ok")
	}
}

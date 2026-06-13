package crawl

import "sync"

// item is one URL waiting to be crawled, with its link depth from the seed.
type item struct {
	URL   string
	Depth int
}

// frontier is the crawl queue: an in-memory, deduplicated set of URLs to fetch,
// guarded by one mutex and a condition variable. A single host's URL space fits
// comfortably in a map, so no bloom filter is needed.
//
// Its defining behavior is self-closing. Next blocks while the queue is empty
// but work is still in flight; when the queue is empty and nothing is pending,
// the frontier closes itself and wakes every waiter, so workers terminate
// without an external shutdown signal.
type frontier struct {
	mu      sync.Mutex
	cond    *sync.Cond
	items   []item
	seen    map[string]bool
	pending int // handed out via Next but not yet Done
	closed  bool
	dfs     bool // pop newest instead of oldest
	keyFn   func(string) string
}

// newFrontier builds an empty frontier. keyFn maps a URL to its dedup key
// (identity if nil); dfs selects depth-first popping when true.
func newFrontier(dfs bool, keyFn func(string) string) *frontier {
	if keyFn == nil {
		keyFn = func(s string) string { return s }
	}
	f := &frontier{seen: map[string]bool{}, dfs: dfs, keyFn: keyFn}
	f.cond = sync.NewCond(&f.mu)
	return f
}

// Add enqueues a URL at the given depth if its key has not been seen. It
// returns true when the URL was newly added.
func (f *frontier) Add(url string, depth int) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return false
	}
	key := f.keyFn(url)
	if f.seen[key] {
		return false
	}
	f.seen[key] = true
	f.items = append(f.items, item{URL: url, Depth: depth})
	f.cond.Signal()
	return true
}

// Next returns the next URL to crawl, blocking until one is available. ok is
// false once the frontier has drained and closed.
func (f *frontier) Next() (item, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for {
		if len(f.items) > 0 {
			var it item
			if f.dfs {
				it = f.items[len(f.items)-1]
				f.items = f.items[:len(f.items)-1]
			} else {
				it = f.items[0]
				f.items = f.items[1:]
			}
			f.pending++
			return it, true
		}
		if f.closed || f.pending == 0 {
			// Nothing queued and nothing in flight: the crawl is done.
			f.closed = true
			f.cond.Broadcast()
			return item{}, false
		}
		f.cond.Wait()
	}
}

// Done reports that the item from a Next call has finished processing. When it
// is the last in-flight item and the queue is empty, the frontier closes.
func (f *frontier) Done() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pending--
	if f.pending <= 0 && len(f.items) == 0 {
		f.closed = true
		f.cond.Broadcast()
	}
}

// Close stops the frontier early (for example when a page cap is reached),
// waking every waiter. Items already handed out still finish.
func (f *frontier) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	f.cond.Broadcast()
}

// Len returns the number of queued items.
func (f *frontier) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.items)
}

// Seen returns the number of distinct URLs ever added.
func (f *frontier) Seen() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.seen)
}

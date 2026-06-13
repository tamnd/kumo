package kumo

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// DefaultDataRoot is the root of the family URI tree, $HOME/data, where every
// kumo page is written as pages/<host>/<path>. It falls back to a temp dir when
// the home directory cannot be resolved.
func DefaultDataRoot() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "data")
	}
	return filepath.Join(home, "data")
}

// hostOf returns the lower-cased host of a URL, or "" when it cannot be parsed.
func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// Store writes page records into the URI tree rooted at a data directory. Each
// page becomes a Markdown file with a JSON front-matter block carrying every
// structured field and the @id/@type/@fetched envelope.
type Store struct {
	root string
}

// NewStore returns a Store rooted at the given data directory.
func NewStore(root string) *Store {
	return &Store{root: root}
}

// Write persists a page to the tree. Pages without a body (errors, redirects,
// non-HTML, and conditional-GET 304s) carry no content to store, so they are
// streamed to the caller but not written, which keeps a re-crawl free of churn.
func (s *Store) Write(p *Page) error {
	if p.Content == "" {
		return nil
	}
	dest := TreePath(s.root, p.Host, p.URL, "md")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	doc, err := s.render(p)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, doc, 0o644)
}

// render builds the Markdown document: a JSON front-matter block fenced by
// ---json ... --- followed by the page body. The body is the Content field, so
// it is cleared before marshalling the front-matter to avoid repeating the
// whole page text inside the metadata block.
func (s *Store) render(p *Page) ([]byte, error) {
	front := *p
	front.Content = ""
	meta, err := json.MarshalIndent(&front, "", "  ")
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("---json\n")
	b.Write(meta)
	b.WriteString("\n---\n\n")
	b.WriteString(p.Content)
	if !strings.HasSuffix(p.Content, "\n") {
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}

// List walks the stored tree for a host and returns each page record, read from
// the JSON front-matter and the Markdown body of every .md file. The body is
// restored onto the record so a listed page round-trips what was written.
func (s *Store) List(host string) ([]*Page, error) {
	dir := filepath.Join(s.root, "pages", host)
	var pages []*Page
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries rather than abort the listing
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if p := parseDoc(raw); p != nil {
			pages = append(pages, p)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return pages, nil
}

// parseDoc splits a stored Markdown document into its JSON front-matter and body
// and reconstructs the Page. It returns nil when the front-matter is absent or
// invalid.
func parseDoc(raw []byte) *Page {
	const openFence, closeFence = "---json\n", "\n---\n"
	if !bytes.HasPrefix(raw, []byte(openFence)) {
		return nil
	}
	meta, body, found := bytes.Cut(raw[len(openFence):], []byte(closeFence))
	if !found {
		return nil
	}
	var p Page
	if err := json.Unmarshal(meta, &p); err != nil {
		return nil
	}
	p.Content = strings.TrimLeft(string(body), "\n")
	return &p
}

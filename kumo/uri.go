package kumo

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// MintURI turns a page's absolute URL into its pages:// URI, the family scheme
// from the URI spec. The authority is the host and the path is the URL path
// (with any query preserved), so https://example.com/docs/x becomes
// pages://example.com/docs/x.
func MintURI(host, rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "pages://" + host + "/"
	}
	p := strings.TrimPrefix(u.Path, "/")
	uri := "pages://" + host + "/" + p
	if u.RawQuery != "" {
		uri += "?" + u.RawQuery
	}
	return uri
}

// TreePath returns the file path, under root, where a page's record is written.
// It mirrors the URL path: the host root and any directory-style path become an
// index file, every other path becomes "<path>.<ext>", and a query string is
// folded into the filename so distinct query pages do not collide.
func TreePath(root, host, rawURL, ext string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return filepath.Join(root, "pages", host, "index."+ext)
	}
	p := strings.Trim(u.Path, "/")
	segs := splitClean(p)

	base := filepath.Join(root, "pages", host)
	if len(segs) == 0 {
		return filepath.Join(base, withQuery("index", u.RawQuery)+"."+ext)
	}
	if strings.HasSuffix(u.Path, "/") {
		// Directory-style URL: store its own page as index inside the directory.
		segs = append(segs, "index")
	}
	last := segs[len(segs)-1]
	segs[len(segs)-1] = withQuery(last, u.RawQuery)
	full := append([]string{base}, segs...)
	return filepath.Join(full...) + "." + ext
}

// withQuery appends a short, filesystem-safe digest of a query string to a file
// stem so /search?q=a and /search?q=b land in different files.
func withQuery(stem, rawQuery string) string {
	stem = safeSegment(stem)
	if rawQuery == "" {
		return stem
	}
	return stem + "@" + safeSegment(rawQuery)
}

// splitClean splits a URL path into safe segments, dropping empties.
func splitClean(p string) []string {
	if p == "" {
		return nil
	}
	parts := strings.Split(path.Clean(p), "/")
	out := make([]string, 0, len(parts))
	for _, s := range parts {
		if s == "" || s == "." {
			continue
		}
		out = append(out, safeSegment(s))
	}
	return out
}

// safeSegment replaces path separators and other awkward characters so a URL
// component is a valid single filename.
func safeSegment(s string) string {
	repl := func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '&', '=', ' ':
			return '_'
		}
		return r
	}
	return strings.Map(repl, s)
}

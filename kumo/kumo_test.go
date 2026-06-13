package kumo

import (
	"path/filepath"
	"testing"
)

func TestMintURI(t *testing.T) {
	cases := []struct {
		host, url, want string
	}{
		{"example.com", "https://example.com/", "pages://example.com/"},
		{"example.com", "https://example.com/docs/x", "pages://example.com/docs/x"},
		{"example.com", "https://example.com/s?q=go", "pages://example.com/s?q=go"},
	}
	for _, c := range cases {
		if got := MintURI(c.host, c.url); got != c.want {
			t.Errorf("MintURI(%q, %q) = %q, want %q", c.host, c.url, got, c.want)
		}
	}
}

func TestTreePath(t *testing.T) {
	root := "/data"
	cases := []struct {
		url, want string
	}{
		{"https://example.com/", filepath.Join(root, "pages/example.com/index.md")},
		{"https://example.com/docs/x", filepath.Join(root, "pages/example.com/docs/x.md")},
		{"https://example.com/docs/", filepath.Join(root, "pages/example.com/docs/index.md")},
	}
	for _, c := range cases {
		if got := TreePath(root, "example.com", c.url, "md"); got != c.want {
			t.Errorf("TreePath(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

func TestTreePathQueryDoesNotCollide(t *testing.T) {
	a := TreePath("/data", "example.com", "https://example.com/s?q=a", "md")
	b := TreePath("/data", "example.com", "https://example.com/s?q=b", "md")
	if a == b {
		t.Fatalf("query pages collided: both %q", a)
	}
}

func TestSameHost(t *testing.T) {
	if !sameHost("www.example.com", "example.com") {
		t.Error("www. prefix should match")
	}
	if !sameHost("docs.example.com", "example.com") {
		t.Error("subdomain should match")
	}
	if sameHost("evil.com", "example.com") {
		t.Error("unrelated host should not match")
	}
}

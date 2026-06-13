package crawl

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"adds root path", "https://example.com", "https://example.com/", true},
		{"lowercases scheme and host", "HTTPS://Example.COM/Path", "https://example.com/Path", true},
		{"drops default https port", "https://example.com:443/a", "https://example.com/a", true},
		{"drops default http port", "http://example.com:80/a", "http://example.com/a", true},
		{"keeps non-default port", "https://example.com:8443/a", "https://example.com:8443/a", true},
		{"drops fragment", "https://example.com/a#section", "https://example.com/a", true},
		{"strips utm params", "https://example.com/a?utm_source=x&id=7", "https://example.com/a?id=7", true},
		{"strips tracking params", "https://example.com/a?gclid=x&fbclid=y", "https://example.com/a", true},
		{"sorts query keys", "https://example.com/a?b=2&a=1", "https://example.com/a?a=1&b=2", true},
		{"rejects relative", "/just/a/path", "", false},
		{"rejects mailto", "mailto:nobody@example.com", "", false},
		{"rejects empty", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := Normalize(c.in)
			if ok != c.ok {
				t.Fatalf("Normalize(%q) ok = %v, want %v", c.in, ok, c.ok)
			}
			if got != c.want {
				t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestScopeInScope(t *testing.T) {
	base := Scope{Host: "example.com"}
	cases := []struct {
		name  string
		scope Scope
		url   string
		want  bool
	}{
		{"same host", base, "https://example.com/a", true},
		{"www variant", base, "https://www.example.com/a", true},
		{"other host", base, "https://other.com/a", false},
		{"subdomain off by default", base, "https://docs.example.com/a", false},
		{"subdomain when enabled", Scope{Host: "example.com", IncludeSubs: true}, "https://docs.example.com/a", true},
		{"prefix match", Scope{Host: "example.com", Prefix: "/docs"}, "https://example.com/docs/x", true},
		{"prefix miss", Scope{Host: "example.com", Prefix: "/docs"}, "https://example.com/blog/x", false},
		{"search excluded by default", base, "https://example.com/search/x", false},
		{"search allowed when enabled", Scope{Host: "example.com", IncludeSearch: true}, "https://example.com/search/x", true},
		{"exclude path", Scope{Host: "example.com", Exclude: []string{"/private"}}, "https://example.com/private/x", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.scope.InScope(c.url); got != c.want {
				t.Errorf("InScope(%q) = %v, want %v", c.url, got, c.want)
			}
		})
	}
}

func TestScopeDedupKeyCollapsesLocale(t *testing.T) {
	s := Scope{Host: "example.com", CollapseLocale: true}
	en, _ := Normalize("https://example.com/en/guide")
	fr, _ := Normalize("https://example.com/fr/guide")
	if a, b := s.DedupKey(en), s.DedupKey(fr); a != b {
		t.Errorf("locale variants got distinct keys: %q vs %q", a, b)
	}

	// Without collapsing, the two are distinct.
	s.CollapseLocale = false
	if a, b := s.DedupKey(en), s.DedupKey(fr); a == b {
		t.Errorf("expected distinct keys without CollapseLocale, both = %q", a)
	}
}

func TestScopeDedupKeyKeepsNonLocaleSegment(t *testing.T) {
	s := Scope{Host: "example.com", CollapseLocale: true}
	// "docs" is not a locale tag, so the path must survive intact.
	u, _ := Normalize("https://example.com/docs/guide")
	if got := s.DedupKey(u); got != u {
		t.Errorf("DedupKey rewrote a non-locale path: %q -> %q", u, got)
	}
}

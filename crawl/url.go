package crawl

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// trackingParams are query parameters that identify a campaign or a click, not
// a distinct page. Stripping them collapses the many tagged copies of one URL
// into a single canonical form so the frontier does not crawl the same content
// dozens of times.
var trackingParams = map[string]bool{
	"gclid": true, "dclid": true, "fbclid": true, "msclkid": true,
	"yclid": true, "twclid": true, "igshid": true, "mc_cid": true,
	"mc_eid": true, "ref": true, "ref_src": true, "ref_url": true,
	"spm": true, "scm": true, "_hsenc": true, "_hsmi": true,
	"vero_id": true, "vero_conv": true, "oly_anon_id": true,
	"oly_enc_id": true, "wickedid": true, "s_cid": true,
}

// localePrefix matches a leading path segment that is a BCP-47 language tag,
// optionally with a script and region (en, en-US, zh-Hans-CN). Stripping it for
// the dedup key collapses the localized copies of a page onto one.
var localePrefix = regexp.MustCompile(`(?i)^/[a-z]{2}(-[a-z]{2,4}){0,2}(/|$)`)

// searchPath matches the obvious site-search routes that explode into endless
// query permutations and rarely carry indexable content.
var searchPath = regexp.MustCompile(`(?i)(^|/)(search|find)(/|$)`)

// Normalize canonicalizes a URL for fetching and storage: it lowercases the
// scheme and host, drops the fragment, drops a default port, strips tracking
// parameters, and sorts the remaining query so two equivalent URLs share one
// string. It returns ok=false for anything that is not an absolute http(s) URL.
func Normalize(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}
	if u.Host == "" {
		return "", false
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Host = stripDefaultPort(u.Host, u.Scheme)
	u.Fragment = ""
	u.RawQuery = cleanQuery(u.Query())
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String(), true
}

// stripDefaultPort removes :80 from an http host and :443 from an https host.
func stripDefaultPort(host, scheme string) string {
	switch {
	case scheme == "http" && strings.HasSuffix(host, ":80"):
		return strings.TrimSuffix(host, ":80")
	case scheme == "https" && strings.HasSuffix(host, ":443"):
		return strings.TrimSuffix(host, ":443")
	}
	return host
}

// cleanQuery drops tracking parameters and utm_* parameters and returns the
// remaining query encoded with sorted keys for a stable canonical form.
func cleanQuery(q url.Values) string {
	for k := range q {
		if trackingParams[strings.ToLower(k)] || strings.HasPrefix(strings.ToLower(k), "utm_") {
			delete(q, k)
		}
	}
	if len(q) == 0 {
		return ""
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		vs := q[k]
		sort.Strings(vs)
		for _, v := range vs {
			if b.Len() > 0 {
				b.WriteByte('&')
			}
			b.WriteString(url.QueryEscape(k))
			b.WriteByte('=')
			b.WriteString(url.QueryEscape(v))
		}
	}
	return b.String()
}

// Scope decides which URLs a crawl is allowed to enqueue. It is built once from
// the crawl options and consulted for every discovered link.
type Scope struct {
	Host           string   // the bare host being crawled (no scheme, no port)
	IncludeSubs    bool     // allow *.Host as well as Host
	Prefix         string   // restrict to URLs whose path starts with this
	Exclude        []string // skip URLs whose path contains any of these
	IncludeSearch  bool     // allow site-search routes
	CollapseLocale bool     // treat localized copies of a page as one
}

// InScope reports whether a normalized URL is eligible to be crawled.
func (s Scope) InScope(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if !s.sameHost(strings.ToLower(u.Hostname())) {
		return false
	}
	if s.Prefix != "" && !strings.HasPrefix(u.Path, s.Prefix) {
		return false
	}
	if !s.IncludeSearch && searchPath.MatchString(u.Path) {
		return false
	}
	for _, ex := range s.Exclude {
		if ex != "" && strings.Contains(u.Path, ex) {
			return false
		}
	}
	return true
}

// sameHost matches the target host against the crawl host, accounting for an
// optional leading www. and, when enabled, any subdomain.
func (s Scope) sameHost(host string) bool {
	want := strings.TrimPrefix(s.Host, "www.")
	host = strings.TrimPrefix(host, "www.")
	if host == want {
		return true
	}
	if s.IncludeSubs && strings.HasSuffix(host, "."+want) {
		return true
	}
	return false
}

// DedupKey returns the key the frontier uses to decide whether a URL has been
// seen. It is the normalized URL, with the leading locale segment removed when
// CollapseLocale is set, so a site that serves the same page under /en/, /fr/,
// and /ja/ is crawled once.
func (s Scope) DedupKey(normalized string) string {
	if !s.CollapseLocale {
		return normalized
	}
	u, err := url.Parse(normalized)
	if err != nil {
		return normalized
	}
	if loc := localePrefix.FindString(u.Path); loc != "" {
		// Keep the trailing slash the match may have consumed.
		rest := u.Path[len(loc):]
		if strings.HasSuffix(loc, "/") {
			rest = "/" + rest
		}
		if rest == "" {
			rest = "/"
		}
		u.Path = rest
	}
	return u.String()
}

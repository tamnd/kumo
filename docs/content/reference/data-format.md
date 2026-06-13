---
title: "Data format"
description: "The URI tree kumo writes: where files land, the front-matter schema, and how to consume it."
weight: 25
---

A crawl writes one file per page into a tree under the data directory (`$HOME/data`
by default, or wherever `--out` points). The tree is the output: it is meant to
be read, grepped, diffed, and committed, not just produced.

## Where a page lands

Each page becomes a Markdown file at `pages/<host>/<path>.md`. The path mirrors
the URL, with two rules:

- A directory URL (one ending in `/`, including the host root) is written as
  `index.md` in that directory. `https://example.com/` becomes
  `pages/example.com/index.md`; `https://example.com/docs/` becomes
  `pages/example.com/docs/index.md`.
- A URL with a query string folds the query into the filename so two pages that
  differ only by query do not collide.

```
$HOME/data/
  pages/
    quotes.toscrape.com/
      index.md
      login.md
      author/
        Albert-Einstein.md
      tag/
        life/
          page/
            1/
              index.md
```

The same layout is addressable as a `pages://<host>/<path>` URI, which is how
on-host links are recorded inside each file (see below).

## The file format

Every file is a JSON front-matter block fenced by `---json` and `---`, followed
by the page content as Markdown:

```markdown
---json
{
  "@id": "pages://quotes.toscrape.com/author/Albert-Einstein",
  "@type": "pages/page",
  "@fetched": "2026-06-14T02:34:03Z",
  "url": "https://quotes.toscrape.com/author/Albert-Einstein",
  "host": "quotes.toscrape.com",
  "status": 200,
  "title": "Quotes to Scrape",
  "description": "Born: March 14, 1879 in Ulm, Germany",
  "lang": "en",
  "author": "Albert Einstein",
  "links": [
    { "uri": "pages://quotes.toscrape.com/", "anchor": "Quotes to Scrape" }
  ],
  "hash": "…",
  "source": "extracted"
}
---

The main content of the page, as Markdown.
```

The front-matter carries the structured record; the converted Markdown is the
body. The body is not repeated inside the front-matter, so the file reads
cleanly and the content lives in exactly one place.

## Front-matter fields

| Field | Meaning |
|---|---|
| `@id` | The page URI, `pages://<host>/<path>` |
| `@type` | `pages/page`, or `pages/unchanged` for a conditional-GET 304 |
| `@fetched` | When the page was fetched (RFC 3339) |
| `url` | The absolute source URL |
| `host` | The crawled host |
| `status` | HTTP status code |
| `title`, `description` | Page title and meta description |
| `lang` | Document language |
| `canonical` | The page's declared canonical URL, if any |
| `author`, `site_name` | From metadata and OpenGraph |
| `published`, `modified` | Article dates, if declared |
| `og` | OpenGraph properties, keyed by their full `og:` name |
| `jsonld` | Raw JSON-LD blocks found on the page |
| `links` | Outbound links: on-host as `pages://` URIs, off-host as absolute URLs |
| `hash` | SHA-256 of the body, for change detection |
| `source` | How the body was produced: `extracted` or `raw-md` |

Pages with no body (an error, a redirect, a non-HTML response, or a 304) are not
written, so re-crawling does not churn the tree with empty files.

## Consuming the tree

Because the format is plain files, the usual tools work. Pull a field out of
every page with a front-matter aware query, or read the records back through
kumo itself:

```bash
# Every page already crawled for a host, as JSON, offline:
kumo pages quotes.toscrape.com -o jsonl | jq -r '.url + "\t" + .title'

# Find pages that link somewhere specific:
kumo pages quotes.toscrape.com -o jsonl \
  | jq -r 'select(.links[]?.uri == "pages://quotes.toscrape.com/login") | .url'
```

The `pages` command reads the tree without touching the network, so it stays
useful long after a crawl. Since the tree is just files, it also versions
cleanly: commit `$HOME/data` and each crawl becomes a diff you can review.

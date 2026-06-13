# kumo

Crawl a whole host into structured data.

`kumo` is a single pure-Go binary. Point it at a host and it fetches every page,
converts each one to clean Markdown and JSON, and writes the result as a
navigable URI tree under your data directory. It honors robots.txt, paces itself
politely, and identifies itself honestly. No API key, nothing to run alongside
it.

## Install

```bash
go install github.com/tamnd/kumo/cmd/kumo@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/kumo/releases), or run
the container image:

```bash
docker run --rm ghcr.io/tamnd/kumo:latest --help
```

## Usage

Crawl a host into the data tree:

```bash
kumo scrape example.com
kumo scrape example.com --max-pages 500 --scope-prefix /docs
kumo scrape example.com --include-subdomains -o jsonl
kumo scrape example.com --dry-run            # list the seed frontier, fetch nothing
```

Each page lands as `pages/<host>/<path>.md`: a JSON front-matter block carrying
the title, description, canonical URL, language, dates, OpenGraph, JSON-LD, and
outbound links, followed by the main content as Markdown.

Work with a single page, or read back what you have already crawled:

```bash
kumo page https://example.com/docs/intro              # fetch and structure one URL
kumo page https://example.com/docs/intro -o jsonl | jq -r .content  # the Markdown body
kumo links https://example.com/docs/intro             # outbound links, as URIs
kumo sitemap example.com                        # the crawlable frontier, no fetch
kumo pages example.com                          # pages already on disk, offline
```

Every command renders in the format you ask for with `-o` (table, json, jsonl,
csv, tsv, url), respects `--limit`, and is exposed over the `serve` and `mcp`
surfaces.

## How a crawl is scoped

A crawl is bound to one host. It seeds from the host root, robots.txt, and the
sitemaps, then follows in-scope links breadth-first (or depth-first with
`--traversal dfs`). robots.txt and its crawl-delay are honored unless you pass
`--no-robots`. Bound the crawl with:

- `--max-pages N` and `--max-depth N`
- `--scope-prefix /docs` to stay under a path
- `--exclude-path /private` (repeatable) to skip paths
- `--include-subdomains` to widen to `*.host`
- `--include-search` to crawl site-search routes (off by default)
- `--raw-md` to probe for a `<page>.md` Markdown sibling on docs sites

## Development

```
cmd/kumo/   thin main, wires the kit app
cli/        the command tree: globals, ops, version
kumo/       the library: engine, page record, URI tree, store
crawl/      the crawl engine: frontier, fetcher, robots, sitemap, scope
extract/    HTML to structured record and Markdown
docs/       tago documentation site
```

```bash
make build      # ./bin/kumo
make test       # go test ./...
make vet        # go vet ./...
```

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the
archives, Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a
cosign signature:

```bash
git tag v0.1.0
git push --tags
```

The Homebrew and Scoop steps self-disable until their tokens exist, so the first
release works with no extra secrets.

## License

Apache-2.0. See [LICENSE](LICENSE).

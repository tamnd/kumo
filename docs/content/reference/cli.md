---
title: "CLI"
description: "Every command and subcommand, with the flags that matter."
weight: 10
---

```
kumo <command> [arguments] [flags]
```

Run `kumo <command> --help` for the full flag list on any command.

## Commands

| Command | What it does |
|---|---|
| `scrape` | Crawl a whole host into structured page records (aliases: `domain`, `site`, `crawl`) |
| `page` | Fetch and structure a single page |
| `links` | List the outbound links of a page, as URIs |
| `sitemap` | Discover the crawlable URL frontier without fetching pages |
| `pages` | List the pages already stored on disk for a host |
| `version` | Print the version and exit |

## scrape

Crawl every page of a host, convert each to a structured record, and write the
result into the data tree as `pages://<host>/<path>`.

```bash
kumo scrape example.com
kumo scrape example.com --max-pages 500 --scope-prefix /docs
kumo scrape example.com --include-subdomains -o jsonl
kumo scrape example.com --dry-run
```

| Flag | Meaning |
|---|---|
| `--max-pages N` | Stop after N pages (0 = unlimited) |
| `--max-depth N` | Link-depth cap from the seed (0 = unlimited) |
| `--traversal bfs\|dfs` | Frontier order (default `bfs`) |
| `--include-subdomains` | Also crawl subdomains of the host |
| `--scope-prefix /path` | Restrict to URLs whose path starts with this |
| `--exclude-path /path` | Skip URLs whose path contains this (repeatable) |
| `--include-search` | Crawl site-search routes too |
| `--no-robots` | Do not fetch or honor robots.txt |
| `--no-sitemap` | Do not seed the frontier from sitemaps |
| `--raw-md` | Probe for a `<page>.md` Markdown sibling on docs sites |

## page

Fetch one URL and emit it as a structured record. This is the pointwise form of
`scrape` and writes no tree.

```bash
kumo page https://example.com/docs/intro
kumo page https://example.com/docs/intro -o jsonl | jq -r .content   # the Markdown body
```

## links

List the outbound links of a single page, resolved to absolute URLs (on-host
links as `pages://` URIs).

```bash
kumo links https://example.com/docs/intro
```

## sitemap

Discover the URLs a crawl would start from, by reading robots.txt and the host's
sitemaps. No page content is fetched.

```bash
kumo sitemap example.com
kumo sitemap example.com -o url | head
```

## pages

Read the data tree for a host and emit the page records already crawled. This
makes no network request.

```bash
kumo pages example.com
kumo pages example.com -o jsonl
```

## Global flags

These apply to every command:

| Flag | Meaning |
|---|---|
| `-o, --output` | Output format: `table`, `json`, `jsonl`, `csv`, `tsv`, `url` |
| `-n, --limit` | Cap the number of records emitted |
| `-j, --workers` | Concurrent fetchers (default 16) |
| `--out` | Data tree root (default `$HOME/data`) |
| `--rate` | Minimum gap between requests |
| `--retries` | Retry attempts per request |
| `--timeout` | Per-request timeout |
| `--dry-run` | Plan the work, fetch nothing |
| `-v, --verbose` / `--quiet` | Adjust logging |

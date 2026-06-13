---
title: "Introduction"
description: "What kumo is and how it is put together."
weight: 10
---

Crawl a whole host into structured data.

kumo is a single binary. Point it at a host and it fetches every page, converts
each one to clean Markdown and JSON, and writes the result as a navigable URI
tree under your data directory. There is nothing to sign up for and nothing to
run alongside it.

## How it is built

- An **extract pipeline** (`extract`) turns each HTML response into a structured
  record: title, description, canonical, language, dates, OpenGraph, JSON-LD,
  outbound links, and the main content as Markdown.
- A **crawl engine** (`crawl`) walks a host: a self-closing frontier, a polite
  paced fetcher, robots.txt and sitemap handling, and the scope rules that keep
  a crawl on one host.
- A **library package** (`kumo`) is the glue: it builds a crawl, mints a URI for
  each page, and writes the URI tree store.
- A **command tree** (`cli`) wraps the library in subcommands with shared output
  formats and flags, and one **`cmd/kumo`** entry point ties it together.

## Scope and manners

kumo reads only what a host already serves publicly. It honors robots.txt and
its crawl-delay by default, paces its requests, and sends an honest
User-Agent. A crawl is bound to one host and you bound it further with page,
depth, and path limits.

Next: [install it](/getting-started/installation/), then take the
[quick start](/getting-started/quick-start/).

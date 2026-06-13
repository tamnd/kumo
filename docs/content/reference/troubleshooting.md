---
title: "Troubleshooting"
description: "The handful of things that trip people up, and how to fix each one."
weight: 40
---

Most of these come down to network reality or how a host serves its pages, not a
bug.

## Requests start failing or returning 429

A public host rate-limits like any other. kumo already paces requests and
retries the transient failures, but a hard limit still means backing off. Raise
the delay between requests with `--rate` (for example `--rate 2s`), lower the
worker count with `-j`, and retry later. A burst of 429 or 5xx responses is the
host asking you to slow down, not a defect.

## A crawl stops earlier than you expected

A crawl is bound to one host and to your limits. If it ends with fewer pages
than you thought, check that robots.txt does not disallow the paths (run with
`--no-robots` to confirm), that `--max-pages`, `--max-depth`, and
`--scope-prefix` are not cutting it short, and that the links you want are
on-host. Subdomains need `--include-subdomains`, and site-search routes need
`--include-search`.

## A page comes back with no content

kumo extracts the main readable content and drops site chrome. A page that is
mostly navigation, a redirect, a non-HTML response, or one that only renders
with JavaScript can yield an empty body, and an empty body is not written to the
tree. For documentation sites that publish a Markdown source, `--raw-md` prefers
the site's own `<page>.md`.

## The binary is not on your PATH

`go install` puts the binary in `$(go env GOPATH)/bin` (usually `~/go/bin`), and
a release archive leaves it wherever you unpacked it. If your shell cannot find
`kumo`, add that directory to your `PATH`. See
[installation](/getting-started/installation/).

## Seeing what kumo actually did

When something behaves unexpectedly, `-v` adds per-request detail so you can see
the URLs it hit and the responses it got. That is usually enough to tell a rate
limit apart from a genuinely empty result.

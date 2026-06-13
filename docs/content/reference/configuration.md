---
title: "Configuration"
description: "Environment variables and global flags."
weight: 20
---

kumo needs almost no configuration: a host on the command line is enough. The
global flags below tune a crawl, and a few environment variables move the
directories it writes to.

## Global flags

These apply to every command:

| Flag | Meaning | Default |
|---|---|---|
| `-o, --output` | Output format: `table`, `json`, `jsonl`, `csv`, `tsv`, `url` | auto |
| `-n, --limit` | Cap the number of records emitted | 0 (no cap) |
| `-j, --workers` | Concurrent fetchers | 16 |
| `--out` | Data tree root | `$HOME/data` |
| `--rate` | Minimum gap between requests | 1s |
| `--retries` | Retry attempts per request | 3 |
| `--timeout` | Per-request timeout | 30s |
| `--dry-run` | Plan the work, fetch nothing | off |
| `-v, --verbose` | Per-request logging | off |
| `--quiet` | Suppress progress output | off |
| `--color` | Colorize output | auto |
| `--help` | Help for any command | |
| `--version` | Print the version | |

## Environment variables

kumo reads a few `KUMO_`-prefixed variables, falling back to the XDG base
directories:

| Variable | Meaning |
|---|---|
| `KUMO_DATA_DIR` | Override the data tree root (otherwise `--out`, then `$HOME/data`) |
| `KUMO_CONFIG_DIR` | Override the config directory |

`XDG_DATA_HOME` and `XDG_CONFIG_HOME` are honored when the `KUMO_` variant is
unset.

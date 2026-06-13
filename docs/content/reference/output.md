---
title: "Output formats"
description: "The output contract every command shares: formats, fields, and templates."
weight: 30
---

Every command renders through one formatter, so the same flags work everywhere.
Pick a format with `-o`, or let kumo choose: a table when writing to a terminal,
JSONL when piped.

## Formats

```bash
kumo <command> -o table   # aligned columns for reading
kumo <command> -o jsonl   # one JSON object per line, for piping
kumo <command> -o json    # a single JSON array
kumo <command> -o csv     # spreadsheet friendly
kumo <command> -o tsv     # tab-separated
kumo <command> -o url     # just the URL column
kumo <command> -o raw     # the underlying bytes, unformatted
```

| Format | Best for |
|---|---|
| `table` | Reading on a terminal |
| `jsonl` | Piping into another tool, one object at a time |
| `json` | Loading a whole result as an array |
| `csv` / `tsv` | Spreadsheets and quick column math |
| `url` | Feeding URLs into other commands |
| `raw` | The unformatted bytes (response bodies, file contents) |

## Narrowing columns

Keep only the fields you want:

```bash
kumo <command> --fields id,title,url
```

`--no-header` drops the header row in `table` and `csv` output, which helps when
a downstream tool expects bare rows.

## Templating rows

For full control over each line, apply a Go text/template. Fields are the JSON
keys, capitalised:

```bash
kumo <command> --template '{{.URL}} {{.Title}}'
```

## Why auto-detection helps

Because the default adapts to the destination, the same command reads well by
hand and parses cleanly in a pipe:

```bash
kumo <command>            # a table, because this is a terminal
kumo <command> | wc -l    # JSONL, because this is a pipe
```

You only reach for `-o` when you want something other than that default.

// Command kumo crawls a whole host into structured data: it fetches every page,
// converts each to clean Markdown and JSON, and writes the result as a
// navigable URI tree under the data directory.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/kumo/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// kit builds the command tree from the operation registry, exposes the serve
	// and mcp surfaces, and maps the typed error taxonomy to exit codes.
	os.Exit(kit.Run(ctx, cli.NewApp()))
}

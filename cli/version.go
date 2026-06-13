package cli

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/tamnd/any-cli/kit"
)

// versionCmd holds the --short flag for the version command.
type versionCmd struct {
	short bool
}

// newVersionCmd builds the version escape-hatch command. Version reporting does
// not fit the emit-records shape, so it is a plain kit.Command rather than an
// operation.
func newVersionCmd() kit.Command {
	v := &versionCmd{}
	return kit.Command{
		Use:   "version",
		Short: "Print version information",
		Flags: v.flags,
		Run:   v.run,
	}
}

func (v *versionCmd) flags(f *kit.FlagSet) {
	f.BoolVar(&v.short, "short", false, "print just the version number")
}

func (v *versionCmd) run(_ context.Context, _ []string) error {
	if v.short {
		_, err := fmt.Fprintln(os.Stdout, Version)
		return err
	}
	_, err := fmt.Fprintf(os.Stdout, "kumo %s (commit %s, built %s, %s/%s, %s)\n",
		Version, Commit, Date, runtime.GOOS, runtime.GOARCH, runtime.Version())
	return err
}

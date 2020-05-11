package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	ff "github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"

	_ "github.com/jtolds/jam/backends/fs"
	_ "github.com/jtolds/jam/backends/s3"
	_ "github.com/jtolds/jam/backends/storj"
)

var (
	sysFlags      = flag.NewFlagSet("", flag.ExitOnError)
	sysFlagConfig = sysFlags.String("config",
		filepath.Join(homeDir(), ".jam", "jam.conf"),
		"path to config file")

	cmdRoot = &ffcli.Command{
		ShortHelp:  "jam preserves your data",
		ShortUsage: fmt.Sprintf("%s [opts] <subcommand> [opts]", os.Args[0]),
		Subcommands: []*ffcli.Command{
			cmdLs,
			cmdMount,
			cmdRename,
			cmdRevertTo,
			cmdRm,
			cmdSnaps,
			cmdStore,
			cmdUnsnap,
			cmdUtils,
		},
		FlagSet: sysFlags,
		Options: []ff.Option{
			ff.WithAllowMissingConfigFile(true),
			ff.WithConfigFileParser(ff.PlainParser),
			ff.WithConfigFileFlag("config"),
		},
		Exec: help,
	}
)

func main() {
	err := cmdRoot.ParseAndRun(context.Background(), os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)
		os.Exit(1)
	}
}

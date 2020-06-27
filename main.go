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

	"github.com/jtolds/jam/utils"

	_ "github.com/jtolds/jam/backends/fs"
	_ "github.com/jtolds/jam/backends/s3"
	_ "github.com/jtolds/jam/backends/sftp"
	_ "github.com/jtolds/jam/backends/storj"
)

var (
	sysFlags      = flag.NewFlagSet("", flag.ExitOnError)
	sysFlagConfig = sysFlags.String("config",
		filepath.Join(homeDir(), ".jam", "jam.conf"),
		"path to config file")
	sysFlagLogLevel = sysFlags.String("log.level", "normal",
		"default log level. can be:\n\tdebug, normal, urgent, or none")

	cmdRoot = &ffcli.Command{
		ShortHelp:  "jam preserves your data",
		ShortUsage: fmt.Sprintf("%s [opts] <subcommand> [opts]", os.Args[0]),
		Subcommands: []*ffcli.Command{
			cmdIntegrity,
			cmdKeys,
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
	err := func() error {
		err := cmdRoot.Parse(os.Args[1:])
		if err != nil {
			return err
		}
		logLevel, err := utils.ParseLogLevel(*sysFlagLogLevel)
		if err != nil {
			return err
		}
		return cmdRoot.Run(
			utils.ContextWithLogger(context.Background(),
				utils.StandardLogger(logLevel)))
	}()
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)
		os.Exit(1)
	}
}

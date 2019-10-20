package cmd

import (
	"github.com/predakanga/bencode_gen/internal"
	"github.com/predakanga/bencode_gen/pkg"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var (
	pkgNames []string
	clobber  bool
	dryRun   bool
	verbose  bool
	rootCmd  = &cobra.Command{
		Use:                   "bencode_gen [flags] [typename]...\n\nTypename may include '*' to find all tagged structs",
		Short:                 "Go code generator for writing bencoded data",
		Version:               pkg.VersionString,
		Run:                   run,
		DisableFlagsInUseLine: true,
	}
)

func init() {
	flags := rootCmd.Flags()
	flags.StringSliceVarP(&pkgNames, "pkg", "p", []string{"./..."}, "package(s) to search")
	flags.BoolVarP(&clobber, "force", "f", false, "overwrite files")
	flags.BoolVarP(&dryRun, "dry-run", "n", false, "don't write any files")
	flags.BoolVarP(&verbose, "verbose", "v", false, "verbose logging")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	var outMode internal.OutputMode
	if dryRun {
		outMode = internal.DryRun
	} else if clobber {
		outMode = internal.Overwrite
	}

	internal.DoGenerate(pkgNames, args, outMode)
}

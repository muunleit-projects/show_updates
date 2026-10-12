/*
upgrades_cli refreshes Homebrew and prints the outdated packages, one per line.

Usage:

	upgrades_cli [-json] [-exit-code] [-timeout 10m]

Exit status: 0 on success, 1 on error, and with -exit-code 2 if updates are
available (handy in scripts: `upgrades_cli -exit-code >/dev/null || ...`).
*/
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

const (
	exitOK      = 0
	exitError   = 1
	exitUpdates = 2
	exitUsage   = 64 // EX_USAGE from sysexits.h
)

type outdatedFunc func(ctx context.Context, opts ...checkupdates.Option) ([]checkupdates.Package, error)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr, checkupdates.Outdated)

	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, outdated outdatedFunc) int {
	flags := flag.NewFlagSet("upgrades_cli", flag.ContinueOnError)
	flags.SetOutput(stderr)

	asJSON := flags.Bool("json", false, "print the packages as JSON")
	exitCode := flags.Bool("exit-code", false, "exit with status 2 if updates are available")
	timeout := flags.Duration("timeout", 10*time.Minute, "timeout per brew command")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}

		return exitUsage
	}

	if flags.NArg() > 0 {
		_, _ = fmt.Fprintf(stderr, "unexpected arguments: %v\n", flags.Args())
		flags.Usage()

		return exitUsage
	}

	pkgs, err := outdated(ctx, checkupdates.WithCommandTimeout(*timeout))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error checking for updates: %v\n", err)
		return exitError
	}

	if err := printPackages(stdout, stderr, pkgs, *asJSON); err != nil {
		_, _ = fmt.Fprintf(stderr, "error writing output: %v\n", err)
		return exitError
	}

	if *exitCode && len(pkgs) > 0 {
		return exitUpdates
	}

	return exitOK
}

// printPackages writes one package per line, or a JSON array. The "up to
// date" note goes to stderr so stdout stays empty and scriptable.
func printPackages(stdout, stderr io.Writer, pkgs []checkupdates.Package, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(pkgs)
	}

	if len(pkgs) == 0 {
		_, err := fmt.Fprintln(stderr, "Everything is up to date.")
		return err
	}

	for _, p := range pkgs {
		if _, err := fmt.Fprintln(stdout, p); err != nil {
			return err
		}
	}

	return nil
}

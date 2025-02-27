// Package main runs the fake game server.
package main

import (
	"context"
	"errors"
	"os"

	"github.com/ettle/strcase"
	"github.com/hamba/cmd/v2"
	"github.com/urfave/cli/v2"
)

const (
	flagExitCode       = "exit-code"
	flagExitSignal     = "exit-signal"
	flagExitAfter      = "exit-after"
	flagAgonesAddr     = "agones-addr"
	flagReadyAfter     = "ready-after"
	flagAllocatedAfter = "allocated-after"
	flagShutdownAfter  = "shutdown-after"
	flagHealthReport   = "health-report-every"

	catExit   = "Exit behavior"
	catAgones = "Agones integration"
)

var version = "<unknown>"

var flags = cmd.Flags{
	&cli.IntFlag{
		Name:     flagExitCode,
		Usage:    "Exit with this code, when an exit condition is met.",
		EnvVars:  []string{strcase.ToSNAKE(flagExitCode)},
		Category: catExit,
	},
	&cli.IntFlag{
		Name:     flagExitSignal,
		Usage:    "Send this signal, when an exit condition is met.",
		EnvVars:  []string{strcase.ToSNAKE(flagExitSignal)},
		Category: catExit,
	},
	&cli.DurationFlag{
		Name:     flagExitAfter,
		Usage:    "Duration after which to exit.",
		EnvVars:  []string{strcase.ToSNAKE(flagExitAfter)},
		Category: catExit,
	},
	&cli.StringFlag{
		Name:     flagAgonesAddr,
		Usage:    "Address to reach the Agones SDK server.",
		Value:    "localhost:9357",
		EnvVars:  []string{strcase.ToSNAKE(flagAgonesAddr)},
		Category: catAgones,
	},
	&cli.DurationFlag{
		Name:     flagReadyAfter,
		Usage:    "Duration after which to transition to Agones state `Ready`.",
		EnvVars:  []string{strcase.ToSNAKE(flagReadyAfter)},
		Category: catAgones,
	},
	&cli.DurationFlag{
		Name:     flagAllocatedAfter,
		Usage:    "Duration after which to transition to Agones state `Allocated`. When the ready timer is not set, the timer starts immediately.",
		EnvVars:  []string{strcase.ToSNAKE(flagAllocatedAfter)},
		Category: catAgones,
	},
	&cli.DurationFlag{
		Name:     flagShutdownAfter,
		Usage:    "Duration after which to transition to Agones state `Shutdown`. When the ready and/or allocated timer is not set, the timer starts immediately.",
		EnvVars:  []string{strcase.ToSNAKE(flagShutdownAfter)},
		Category: catAgones,
	},
	&cli.DurationFlag{
		Name: flagHealthReport,
		Usage: "Period after which to send the Agones health report. " +
			"The first health report is sent after the configured duration, or if an Agones status timer is configured, right after the first transition.",
		EnvVars:  []string{strcase.ToSNAKE(flagHealthReport)},
		Category: catAgones,
	},
}.Merge(cmd.LogFlags)

func main() {
	app := cli.NewApp()
	app.Name = "Fake Game Server with Agones integration"
	app.Version = version
	app.Flags = flags
	app.Action = run

	if err := app.RunContext(context.Background(), os.Args); err != nil {
		var exitErr *exitError
		if errors.As(err, &exitErr) {
			exitErr.runHooks()
		}
		os.Exit(1)
	}
	os.Exit(0)
}

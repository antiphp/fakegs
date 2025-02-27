package main

import (
	"fmt"
	"os/signal"
	"syscall"

	"github.com/antiphp/fakegs"
	"github.com/hamba/cmd/v2"
	lctx "github.com/hamba/logger/v2/ctx"
	"github.com/urfave/cli/v2"
)

func run(c *cli.Context) error {
	ctx, cancel := signal.NotifyContext(c.Context, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log, err := cmd.NewLogger(c)
	if err != nil {
		return fmt.Errorf("creating logger: %w", err)
	}

	srv := fakegs.NewServer(log)

	if c.IsSet(flagExitAfter) {
		srv.Add(fakegs.NewExitAfter(c.Duration(flagExitAfter)))
	}

	exitErr := createExitErr(c)

	log.Info("Game server started")
	reason, err := srv.Run(ctx)
	if err != nil {
		wrapErr := wrapExitErrs(err, exitErr, newExitErr(ptr(1), nil))
		log.Info("Fake game server stopped", lctx.Err(err), lctx.Str("how", wrapErr.Error()))

		return wrapErr
	}

	wrapErr := wrapExitErrs(nil, exitErr, newExitErr(ptr(0), nil))
	log.Info("Game server stopped", lctx.Str("reason", reason), lctx.Str("how", wrapErr.Error()))
	return wrapErr
}

func createExitErr(c *cli.Context) *exitError {
	var sig, code *int
	if c.IsSet(flagExitSignal) {
		*sig = c.Int(flagExitSignal)
	}
	if c.IsSet(flagExitCode) {
		*code = c.Int(flagExitCode)
	}
	return newExitErr(sig, code)
}

func ptr(i int) *int {
	return &i
}

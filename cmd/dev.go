package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"recall/internal/apiserver"
)

type devOptions struct {
	apiPort int
	uiPort  int
	install bool
}

func (o devOptions) apiURL() string {
	return "http://localhost:" + strconv.Itoa(o.apiPort)
}

func parseDevArgs(args []string) (devOptions, error) {
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	parsed := devOptions{apiPort: 8888, uiPort: 5173}
	fs.IntVar(&parsed.apiPort, "api-port", parsed.apiPort, "Go API port")
	fs.IntVar(&parsed.uiPort, "ui-port", parsed.uiPort, "Vite UI port")
	fs.BoolVar(&parsed.install, "install", false, "run npm ci before starting Vite")
	if err := fs.Parse(args); err != nil {
		return devOptions{}, err
	}
	if fs.NArg() != 0 {
		return devOptions{}, fmt.Errorf("usage: recall dev [--api-port N] [--ui-port N] [--install]")
	}
	return parsed, nil
}

// Dev starts the local Go API server and the Vite UI dev server together.
func Dev(args []string) error {
	opts, err := parseDevArgs(args)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if opts.install {
		install := exec.CommandContext(ctx, "npm", "--prefix", "ui", "ci")
		install.Stdout = os.Stdout
		install.Stderr = os.Stderr
		install.Stdin = os.Stdin
		if err := install.Run(); err != nil {
			return fmt.Errorf("npm install: %w", err)
		}
	}

	e, err := openEngine()
	if err != nil {
		return err
	}
	defer e.Close()

	app := apiserver.New(e)
	apiAddr := "localhost:" + strconv.Itoa(opts.apiPort)
	apiErr := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stderr, "recall api  →  %s\n", opts.apiURL())
		apiErr <- app.Listen(apiAddr)
	}()
	defer func() { _ = app.Shutdown() }()

	select {
	case err := <-apiErr:
		if err != nil && ctx.Err() == nil {
			return fmt.Errorf("api server: %w", err)
		}
		return err
	case <-time.After(250 * time.Millisecond):
	}

	vite := exec.CommandContext(ctx, "npm", "--prefix", "ui", "run", "dev", "--", "--host", "localhost", "--port", strconv.Itoa(opts.uiPort))
	vite.Stdout = os.Stdout
	vite.Stderr = os.Stderr
	vite.Stdin = os.Stdin
	vite.Env = append(os.Environ(), "RECALL_API_URL="+opts.apiURL())
	fmt.Fprintf(os.Stderr, "recall vite →  http://localhost:%d\n", opts.uiPort)

	if err := vite.Run(); err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("vite dev server: %w", err)
	}
	return nil
}

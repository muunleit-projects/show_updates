/*
Package checkupdates refreshes Homebrew and reports outdated formulae and casks.

	pkgs, err := checkupdates.Outdated(ctx)

or, with options:

	c, err := checkupdates.NewChecker(checkupdates.WithCommandTimeout(5 * time.Minute))
	pkgs, err := c.Outdated(ctx)
*/
package checkupdates

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
)

// Checker runs `brew update` followed by `brew outdated`. Create one with
// NewChecker; the zero value is not usable.
type Checker struct {
	update            []string
	outdated          []string
	connected         bool
	connectionTimeout time.Duration
	commandTimeout    time.Duration
	dialTimeout       func(network, address string, timeout time.Duration) (net.Conn, error)
}

// Option configures a Checker.
type Option func(c *Checker) error

const (
	// defaultConnectionTimeout is the default time for the connectivity
	// probe. The probe is only used to enrich error messages, so it must
	// stay short.
	defaultConnectionTimeout = 3 * time.Second

	// probeAddress is the host Homebrew talks to (over HTTPS).
	probeAddress = "github.com:443"

	// defaultCommandTimeout bounds each brew command. `brew update` can take
	// a few minutes on a slow connection, but should never hang forever
	// (e.g. on a stale lock or a stuck network connection).
	defaultCommandTimeout = 10 * time.Minute

	// waitDelay is how long to wait for output pipes to close after a
	// command was killed, since brew may leave child processes behind.
	waitDelay = 5 * time.Second
)

var (
	// ErrBrewNotFound is returned when no brew executable can be located and
	// no custom commands were provided.
	ErrBrewNotFound = errors.New("brew not found in PATH or standard locations")

	// ErrCommandTimeout is wrapped in the error returned when a command
	// exceeds the command timeout.
	ErrCommandTimeout = errors.New("command timed out")
)

// commonBrewPaths are checked when brew is not in PATH, since PATH is often
// limited for GUI apps launched from Finder or at login.
var commonBrewPaths = []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"}

/*
NewChecker returns a new Checker.

By default the values are
update command: "<brew> update"
outdated command: "<brew> outdated --json=v2 --greedy"
connection timeout: 3 seconds
command timeout: 10 minutes per command
connection check: enabled.

The brew executable is only looked up if at least one of the commands was not
set via WithUpdate or WithOutdated.
*/
func NewChecker(opts ...Option) (Checker, error) {
	c := Checker{
		connectionTimeout: defaultConnectionTimeout,
		commandTimeout:    defaultCommandTimeout,
		dialTimeout:       net.DialTimeout,
	}

	for _, opt := range opts {
		if err := opt(&c); err != nil {
			return Checker{}, fmt.Errorf("error in option: %w", err)
		}
	}

	if c.update != nil && c.outdated != nil {
		return c, nil
	}

	brewPath, err := findBrew(exec.LookPath, fileExists)
	if err != nil {
		return Checker{}, err
	}

	if c.update == nil {
		c.update = []string{brewPath, "update"}
	}

	if c.outdated == nil {
		c.outdated = []string{brewPath, "outdated", "--json=v2", "--greedy"}
	}

	return c, nil
}

// findBrew locates the brew executable, first in PATH, then in the standard
// Homebrew install locations.
func findBrew(lookPath func(string) (string, error), exists func(string) bool) (string, error) {
	if p, err := lookPath("brew"); err == nil {
		return p, nil
	}

	for _, p := range commonBrewPaths {
		if exists(p) {
			return p, nil
		}
	}

	return "", ErrBrewNotFound
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

// WithConnectedTrue disables the connectivity probe, primarily for tests.
func WithConnectedTrue() Option {
	return func(c *Checker) error {
		c.connected = true

		return nil
	}
}

// WithUpdate sets the command used for the update step.
func WithUpdate(cmd ...string) Option {
	return func(c *Checker) error {
		if len(cmd) == 0 || cmd[0] == "" {
			return errors.New("empty update command")
		}

		c.update = cmd

		return nil
	}
}

// WithOutdated sets the command that lists outdated packages. It must print
// the same JSON as `brew outdated --json=v2`.
func WithOutdated(cmd ...string) Option {
	return func(c *Checker) error {
		if len(cmd) == 0 || cmd[0] == "" {
			return errors.New("empty outdated command")
		}

		c.outdated = cmd

		return nil
	}
}

// WithConnectionTimeout sets the maximum time the checker should spend probing GitHub.
func WithConnectionTimeout(t time.Duration) Option {
	return func(c *Checker) error {
		if t <= 0 {
			return errors.New("timeout should be greater than 0")
		}

		c.connectionTimeout = t

		return nil
	}
}

// WithCommandTimeout sets the maximum run time of each command (update and
// outdated). The command is killed when the timeout is exceeded.
func WithCommandTimeout(t time.Duration) Option {
	return func(c *Checker) error {
		if t <= 0 {
			return errors.New("command timeout should be greater than 0")
		}

		c.commandTimeout = t

		return nil
	}
}

// WithDialTimeoutFunc overrides the dialer used for connectivity probes.
func WithDialTimeoutFunc(dial func(network, address string, timeout time.Duration) (net.Conn, error)) Option {
	return func(c *Checker) error {
		if dial == nil {
			return errors.New("dial function should not be nil")
		}

		c.dialTimeout = dial

		return nil
	}
}

/*
Outdated refreshes Homebrew and returns the outdated packages, formulae first,
then casks, each sorted by name. An empty slice means everything is up to date.

A failed connectivity probe does not abort the check, because Homebrew may
still work through a proxy, mirror or local network setup. If the update step
fails afterwards, the probe error is attached to make the cause easier to spot.

Each command is killed after the command timeout (see WithCommandTimeout) or
when ctx is cancelled.
*/
func (c Checker) Outdated(ctx context.Context) ([]Package, error) {
	var connErr error
	if !c.connected {
		connErr = c.connectivity()
	}

	output, err := c.run(ctx, c.update, nil)
	if err != nil {
		err = fmt.Errorf("update cmd %v returned %s: %w", c.update, bytes.TrimSpace(output), err)
		if connErr != nil {
			err = errors.Join(err, fmt.Errorf("connectivity check failed: %w", connErr))
		}

		return nil, err
	}

	// Only stdout holds the JSON; stderr may carry brew warnings.
	var stderr bytes.Buffer

	output, err = c.run(ctx, c.outdated, &stderr)
	if err != nil {
		return nil, fmt.Errorf("outdated cmd %v returned %s: %w", c.outdated, bytes.TrimSpace(stderr.Bytes()), err)
	}

	pkgs, err := ParseOutdatedJSON(output)
	if err != nil {
		return nil, fmt.Errorf("outdated cmd %v: %w", c.outdated, err)
	}

	return pkgs, nil
}

// Outdated creates a Checker with the given options and calls its Outdated method.
func Outdated(ctx context.Context, opts ...Option) ([]Package, error) {
	c, err := NewChecker(opts...)
	if err != nil {
		return nil, err
	}

	return c.Outdated(ctx)
}

/*
run executes cmd with the command timeout. If stderr is nil, stdout and stderr
are combined in the returned output; otherwise only stdout is returned and
stderr is written to the given buffer.
*/
func (c Checker) run(ctx context.Context, cmd []string, stderr *bytes.Buffer) ([]byte, error) {
	ctx, cancel := context.WithTimeoutCause(ctx, c.commandTimeout, ErrCommandTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	command.WaitDelay = waitDelay

	var (
		output []byte
		err    error
	)

	if stderr == nil {
		output, err = command.CombinedOutput()
	} else {
		command.Stderr = stderr
		output, err = command.Output()
	}

	if err != nil && ctx.Err() != nil {
		// context.Cause distinguishes our own timeout from a cancellation or
		// deadline of the caller's context.
		if cause := context.Cause(ctx); errors.Is(cause, ErrCommandTimeout) {
			return output, fmt.Errorf("%w after %v: %w", ErrCommandTimeout, c.commandTimeout, err)
		}

		return output, fmt.Errorf("%w: %w", context.Cause(ctx), err)
	}

	return output, err
}

/* connectivity probes the connection to GitHub once. */
func (c Checker) connectivity() error {
	con, err := c.dialTimeout("tcp", probeAddress, c.connectionTimeout)
	if err != nil {
		return err
	}

	return con.Close()
}

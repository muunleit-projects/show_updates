package checkupdates

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

/*
	basics
*/

type checker struct {
	update            []string
	upgrade           []string
	connected         bool
	connectionTimeout time.Duration
	commandTimeout    time.Duration
	dialTimeout       func(network, address string, timeout time.Duration) (net.Conn, error)
}

type (
	options func(c *checker) error
)

/*
	Setting up new checkers
*/

const (
	// timeoutTime is the default time for the connectivity probe. The probe is
	// only used to enrich error messages, so it must stay short.
	timeoutTime = 3 * time.Second

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

// ErrBrewNotFound is returned when no brew executable can be located and no
// custom commands were provided.
var ErrBrewNotFound = errors.New("brew not found in PATH or standard locations")

// commonBrewPaths are checked when brew is not in PATH, since PATH is often
// limited for GUI apps launched from Finder or at login.
var commonBrewPaths = []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"}

/*
NewChecker returns a new checker.

By default the values are
update command: "<brew> update"
upgrade command: "<brew> outdated -g"
connection timeout: 3 seconds
command timeout: 10 minutes per command
connection check: enabled.

The brew executable is only looked up if at least one of the commands was not
set via WithUpdate or WithUpgradeable.
*/
func NewChecker(opts ...options) (checker, error) {
	c := checker{
		connectionTimeout: timeoutTime,
		commandTimeout:    defaultCommandTimeout,
		connected:         false,
		dialTimeout:       net.DialTimeout,
	}

	for _, opt := range opts {
		if err := opt(&c); err != nil {
			return checker{}, fmt.Errorf("error in option: %w", err)
		}
	}

	if c.update != nil && c.upgrade != nil {
		return c, nil
	}

	brewPath, err := findBrew(exec.LookPath, fileExists)
	if err != nil {
		return checker{}, err
	}

	if c.update == nil {
		c.update = []string{brewPath, "update"}
	}

	if c.upgrade == nil {
		c.upgrade = []string{brewPath, "outdated", "-g"}
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

// WithConnectedTrue disables the connectivity check, primarily for tests.
func WithConnectedTrue() options {
	return func(c *checker) error {
		c.connected = true

		return nil
	}
}

// WithUpdate sets the command used for the update step.
func WithUpdate(cmd ...string) options {
	return func(c *checker) error {
		if len(cmd) == 0 || cmd[0] == "" {
			return errors.New("empty update command")
		}

		c.update = cmd

		return nil
	}
}

// WithUpgradeable sets the command used to detect upgradable packages.
func WithUpgradeable(cmd ...string) options {
	return func(c *checker) error {
		if len(cmd) == 0 || cmd[0] == "" {
			return errors.New("empty upgradeable command")
		}

		c.upgrade = cmd

		return nil
	}
}

// WithConnectionTimeout sets the maximum time the checker should spend probing GitHub.
func WithConnectionTimeout(t time.Duration) options {
	return func(c *checker) error {
		if t <= 0 {
			return errors.New("timeout should be greater than 0")
		}

		c.connectionTimeout = t

		return nil
	}
}

// WithCommandTimeout sets the maximum run time of each command (update and
// upgradeable check). The command is killed when the timeout is exceeded.
func WithCommandTimeout(t time.Duration) options {
	return func(c *checker) error {
		if t <= 0 {
			return errors.New("command timeout should be greater than 0")
		}

		c.commandTimeout = t

		return nil
	}
}

// WithDialTimeoutFunc overrides the dialer used for connectivity probes.
func WithDialTimeoutFunc(dial func(network, address string, timeout time.Duration) (net.Conn, error)) options {
	return func(c *checker) error {
		if dial == nil {
			return errors.New("dial function should not be nil")
		}

		c.dialTimeout = dial

		return nil
	}
}

/*
	Methods and Functions
*/

/*
Upgradable refreshes the package list and returns the upgradable packages.

A failed connectivity probe does not abort the check, because Homebrew may
still work through a proxy, mirror or local network setup. If the update step
fails afterwards, the probe error is attached to make the cause easier to spot.

Each command is killed after the command timeout (see WithCommandTimeout).
*/
func (c checker) Upgradable() (string, error) {
	return c.UpgradableContext(context.Background())
}

// UpgradableContext is like Upgradable, but stops the running command when
// ctx is cancelled.
func (c checker) UpgradableContext(ctx context.Context) (string, error) {
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

		return "", err
	}

	// Only stdout holds the package list; stderr may carry brew warnings.
	var stderr bytes.Buffer

	output, err = c.run(ctx, c.upgrade, &stderr)
	if err != nil {
		return "", fmt.Errorf("upgrade cmd %v returned %s: %w", c.upgrade, bytes.TrimSpace(stderr.Bytes()), err)
	}

	return strings.TrimSpace(string(output)), nil
}

// ErrCommandTimeout is wrapped in the error returned when a command exceeds
// the command timeout.
var ErrCommandTimeout = errors.New("command timed out")

/*
run executes cmd with the command timeout. If stderr is nil, stdout and stderr
are combined in the returned output; otherwise only stdout is returned and
stderr is written to the given buffer.
*/
func (c checker) run(ctx context.Context, cmd []string, stderr *bytes.Buffer) ([]byte, error) {
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

// Upgradable is the wrapper-function for the Upgradable-method.
func Upgradable() (string, error) {
	c, err := NewChecker()
	if err != nil {
		return "", err
	}

	return c.Upgradable()
}

/* connectivity probes the connection to GitHub once. */
func (c checker) connectivity() error {
	con, err := c.dialTimeout("tcp", probeAddress, c.connectionTimeout)
	if err != nil {
		return err
	}

	return con.Close()
}

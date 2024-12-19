package checkupdates

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

/*
	basics
*/

const path = "/opt/homebrew/bin/"

type checker struct {
	update            []string
	upgrade           []string
	connected         bool
	connectionTimeout time.Duration
}

type (
	options func(c *checker) error
)

/*
	Setting up new checkers
*/

// NewChecker returns a new checker with a default connection timeout of 30 seconds.
// The default upgrade command is "/opt/homebrew/bin/brew outdated -g".
// The default update command is "/opt/homebrew/bin/brew update".
func NewChecker(opts ...options) (checker, error) {
	c := checker{
		connectionTimeout: time.Second * 30,
		update:            []string{path + "brew", "update"},
		upgrade:           []string{path + "brew", "outdated", "-g"},
	}

	for _, opt := range opts {
		if err := opt(&c); err != nil {
			return checker{}, err
		}
	}

	return c, nil
}

// WithConnectedTrue disables connectivity check
func WithConnectedTrue() options {
	return func(c *checker) error {
		c.connected = true

		return nil
	}
}

// WithUpdate sets the command used for updates
func WithUpdate(cmd ...string) options {
	return func(c *checker) error {
		if cmd[0] == "" {
			return errors.New("empty update command")
		}

		c.update = cmd

		return nil
	}
}

// WithUpgradeable sets the command to check for upgradable packages
func WithUpgradeable(cmd ...string) options {
	return func(c *checker) error {
		if cmd[0] == "" {
			return errors.New("empty upgradeable command")
		}

		c.upgrade = cmd

		return nil
	}
}

// WithConnetionTries sets the count for how often the program should try to
// connect github
func WithConnectionTimeout(t time.Duration) options {
	return func(c *checker) error {
		if t <= 0 {
			return errors.New("timeout should be greater than 0")
		}

		c.connectionTimeout = t

		return nil
	}
}

/*
	Methods and Functions
*/

// Upgradable updates the packagelist and returns the upgradeable packages
func (c checker) Upgradable() (string, error) {
	if !c.connected {
		if err := c.connectivity(); err != nil {
			return "", err
		}
	}

	update := exec.Command(c.update[0], c.update[1:]...)

	output, err := update.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("update cmd %v returned %s, %w", c.update, output, err)
	}

	upgrade := exec.Command(c.upgrade[0], c.upgrade[1:]...)

	output, err = upgrade.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("upgrade cmd %v returned %s, %w", c.upgrade, output, err)
	}

	return strings.TrimSpace(string(output)), nil
}

// Upgradable is the wrapper-function for the Upgradable-method
func Upgradable() (string, error) {
	c, err := NewChecker()
	if err != nil {
		panic("internal error")
	}

	return c.Upgradable()
}

// connectivity checks the connection to GitHub
func (c checker) connectivity() error {
	var (
		con   net.Conn
		err   error
		begin = time.Now()
		dur   time.Duration
	)

	for dur <= c.connectionTimeout {
		con, err = net.Dial("tcp", "github.com:80")
		if err != nil {
			dur = time.Since(begin)
			time.Sleep(time.Second) // Sleep for 1 second before retrying

			continue
		}

		defer con.Close()

		break
	}

	return err
}

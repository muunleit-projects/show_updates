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

// NewChecker returns a new checker .....
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
			return errors.New("timeout should be bigger then 0")
		}

		c.connectionTimeout = t

		return nil
	}
}

/*
	Methods and Functions
*/

// Upgradable updates the packagelist and returs the upgradeable packages
func (c checker) Upgradable() (string, error) {
	if !c.connected {
		if err := c.connectivity(); err != nil {
			return "", err
		}
	}

	update := exec.Command(c.update[0], c.update[1:]...)

	output, err := update.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("update cmd %v rerturned %s, %w", c.update, output, err)
		return "", err
	}

	upgrade := exec.Command(c.upgrade[0], c.upgrade[1:]...)

	output, err = upgrade.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("upgrade cmd %v rerturned %s, %w", c.upgrade, output, err)
		return "", err
	}

	out := string(output)
	out = strings.TrimSpace(out)

	return out, err
}

// Upgradeable is the wrapper-function for the Upgradeable-method
func Upgradable() (string, error) {
	c, err := NewChecker()
	if err != nil {
		panic("internal error")
	}

	return c.Upgradable()
}

// connectivity checks the connect to github
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
			continue
		}

		defer func() {
			con.Close()
		}()

		break
	}

	return err
}

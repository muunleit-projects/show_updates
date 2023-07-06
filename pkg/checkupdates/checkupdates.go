package checkupdates

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"time"
)

/*
	basics
*/

type checker struct {
	connected       bool
	connectionTries int
	update          []string
	upgrade         []string
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
		update:          []string{"brew", "update", "-g"},
		upgrade:         []string{"brew", "outdated", "-g"},
		connectionTries: 10,
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
		c.update = cmd
		return nil
	}
}

// WithUpgradeable sets the command to check for upgradable packages
func WithUpgradeable(cmd ...string) options {
	return func(c *checker) error {
		c.upgrade = cmd
		return nil
	}
}

// WithConnetionTries sets the count for how often the program should try to
// connect github
func WithConnectionTries(t int) options {
	return func(c *checker) error {
		c.connectionTries = t
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
	err := update.Run()
	if err != nil {
		err = fmt.Errorf("update cmd %v, %w", c.update, err)
		return "", err
	}

	upgrade := exec.Command(c.upgrade[0], c.upgrade[1:]...)
	output, err := upgrade.Output()
	if err != nil {
		err = fmt.Errorf("upgrade cmd %v, %w", c.upgrade, err)
	}
	return string(output), err
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
func (c checker) connectivity() (err error) {
	for i := 0; i < c.connectionTries; i++ {
		conn, derr := net.Dial("tcp", "github.com:80")
		err = derr
		if derr != nil {
			time.Sleep(time.Second)
			continue
		}

		if cerr := conn.Close(); cerr != nil {
			err = errors.Join(err, cerr)
		}
		break
	}
	return err
}

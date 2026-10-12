package checkupdates

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Kind tells formulae and casks apart.
type Kind string

const (
	Formula Kind = "formula"
	Cask    Kind = "cask"
)

// Package is one outdated formula or cask.
type Package struct {
	Name              string   `json:"name"`
	Kind              Kind     `json:"kind"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`

	// Pinned packages are skipped by `brew upgrade` until they are unpinned.
	Pinned        bool   `json:"pinned"`
	PinnedVersion string `json:"pinned_version,omitempty"`
}

// String formats p like the plain `brew outdated --verbose` output, e.g.
// "git (2.44.0) < 2.45.1" or "node (20.1.0) < 22.2.0 [pinned at 20.1.0]".
func (p Package) String() string {
	var b strings.Builder

	b.WriteString(p.Name)

	if len(p.InstalledVersions) > 0 {
		b.WriteString(" (" + strings.Join(p.InstalledVersions, ", ") + ")")
	}

	if p.CurrentVersion != "" {
		b.WriteString(" < " + p.CurrentVersion)
	}

	if p.Pinned {
		b.WriteString(" [pinned")
		if p.PinnedVersion != "" {
			b.WriteString(" at " + p.PinnedVersion)
		}
		b.WriteString("]")
	}

	return b.String()
}

// outdatedJSON mirrors `brew outdated --json=v2`.
type outdatedJSON struct {
	Formulae *[]rawPackage `json:"formulae"`
	Casks    *[]rawPackage `json:"casks"`
}

type rawPackage struct {
	Name              string   `json:"name"`
	InstalledVersions versions `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
	Pinned            bool     `json:"pinned"`
	PinnedVersion     *string  `json:"pinned_version"`
}

// versions accepts both a list of strings (current brew) and a single string
// (some older brew releases did this for casks).
type versions []string

func (v *versions) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*v = nil
		return nil
	}

	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*v = versions{single}
		return nil
	}

	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("installed_versions: %w", err)
	}

	*v = list

	return nil
}

// ErrUnexpectedOutput is wrapped when the outdated command does not print the
// JSON of `brew outdated --json=v2`.
var ErrUnexpectedOutput = errors.New("unexpected output from brew outdated")

/*
ParseOutdatedJSON parses the output of `brew outdated --json=v2`. It returns
formulae first, then casks, each sorted by name.
*/
func ParseOutdatedJSON(data []byte) ([]Package, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: empty output", ErrUnexpectedOutput)
	}

	var parsed outdatedJSON
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnexpectedOutput, err)
	}

	// Both keys are always present in brew's output. Missing keys mean we got
	// some other JSON (or a future format), which should not be read as
	// "everything up to date".
	if parsed.Formulae == nil || parsed.Casks == nil {
		return nil, fmt.Errorf("%w: missing \"formulae\" or \"casks\"", ErrUnexpectedOutput)
	}

	pkgs := make([]Package, 0, len(*parsed.Formulae)+len(*parsed.Casks))
	pkgs = appendPackages(pkgs, *parsed.Formulae, Formula)
	pkgs = appendPackages(pkgs, *parsed.Casks, Cask)

	return pkgs, nil
}

func appendPackages(pkgs []Package, raw []rawPackage, kind Kind) []Package {
	start := len(pkgs)

	for _, r := range raw {
		p := Package{
			Name:              r.Name,
			Kind:              kind,
			InstalledVersions: []string(r.InstalledVersions),
			CurrentVersion:    r.CurrentVersion,
			Pinned:            r.Pinned,
		}
		if r.PinnedVersion != nil {
			p.PinnedVersion = *r.PinnedVersion
		}

		pkgs = append(pkgs, p)
	}

	slices.SortStableFunc(pkgs[start:], func(a, b Package) int {
		return strings.Compare(a.Name, b.Name)
	})

	return pkgs
}

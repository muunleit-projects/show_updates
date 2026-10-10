package checkupdates_test

import (
	"errors"
	"os"
	"reflect"
	"testing"

	cu "github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

func TestParseOutdatedJSON(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testfiles/outdated.json")
	if err != nil {
		t.Fatal(err)
	}

	got, err := cu.ParseOutdatedJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Formulae first, then casks, each sorted by name.
	want := []cu.Package{
		{Name: "git", Kind: cu.Formula, InstalledVersions: []string{"2.44.0", "2.44.1"}, CurrentVersion: "2.45.1"},
		{Name: "node", Kind: cu.Formula, InstalledVersions: []string{"20.1.0"}, CurrentVersion: "22.2.0", Pinned: true, PinnedVersion: "20.1.0"},
		{Name: "firefox", Kind: cu.Cask, InstalledVersions: []string{"125.0"}, CurrentVersion: "126.0"},
		{Name: "visual-studio-code", Kind: cu.Cask, InstalledVersions: []string{"1.89.0"}, CurrentVersion: "1.90.0"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestParseOutdatedJSONEmpty(t *testing.T) {
	t.Parallel()

	got, err := cu.ParseOutdatedJSON([]byte("{\n  \"formulae\": [],\n  \"casks\": []\n}\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got == nil || len(got) != 0 {
		t.Errorf("expected empty non-nil slice, got %#v", got)
	}
}

func TestParseOutdatedJSONRejectsUnexpectedOutput(t *testing.T) {
	t.Parallel()

	for _, input := range []string{
		"",
		"   \n",
		"waldi\nbello\nrex",
		`{}`,
		`{"formulae": []}`,
		`[]`,
		`{"formulae": [{"name": "x", "installed_versions": 42}], "casks": []}`,
	} {
		if _, err := cu.ParseOutdatedJSON([]byte(input)); !errors.Is(err, cu.ErrUnexpectedOutput) {
			t.Errorf("ParseOutdatedJSON(%q): expected ErrUnexpectedOutput, got %v", input, err)
		}
	}
}

func TestPackageString(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		pkg  cu.Package
		want string
	}{
		{cu.Package{Name: "git", InstalledVersions: []string{"2.44.0"}, CurrentVersion: "2.45.1"}, "git (2.44.0) < 2.45.1"},
		{cu.Package{Name: "git", InstalledVersions: []string{"2.44.0", "2.44.1"}, CurrentVersion: "2.45.1"}, "git (2.44.0, 2.44.1) < 2.45.1"},
		{cu.Package{Name: "node", InstalledVersions: []string{"20"}, CurrentVersion: "22", Pinned: true, PinnedVersion: "20"}, "node (20) < 22 [pinned at 20]"},
		{cu.Package{Name: "x", Pinned: true}, "x [pinned]"},
		{cu.Package{Name: "bare"}, "bare"},
	} {
		if got := tt.pkg.String(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

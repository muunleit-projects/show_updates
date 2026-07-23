package main

import (
	"errors"
	"testing"
)

func TestBuildErrorMessage(t *testing.T) {
	got := buildErrorMessage(errors.New("boom"), "/tmp/outdated_apps.log")
	want := "Error: boom"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFormatNoUpdatesMessage(t *testing.T) {
	got := formatNoUpdatesMessage(7)
	want := "No updates found.\nWindow closes in 7"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

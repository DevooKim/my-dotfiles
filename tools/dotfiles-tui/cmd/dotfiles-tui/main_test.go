package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type restartModel struct {
	restart bool
}

func (m restartModel) Init() tea.Cmd                       { return nil }
func (m restartModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m restartModel) View() tea.View                      { return tea.NewView("") }
func (m restartModel) RestartRequested() bool              { return m.restart }

func TestReexecIfRequestedRunsOnlyAfterProgramReturns(t *testing.T) {
	called := false
	var gotPath string
	var gotArguments []string
	err := reexecIfRequested(restartModel{restart: true}, "/repo/setup", "/repo", func(path string, arguments []string, _ []string) error {
		called = true
		gotPath = path
		gotArguments = arguments
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called || gotPath != "/repo/setup" {
		t.Fatalf("called/path = %v/%q", called, gotPath)
	}
	want := []string{"/repo/setup", "--repo", "/repo", "--after-update"}
	if !reflect.DeepEqual(gotArguments, want) {
		t.Fatalf("arguments = %#v, want %#v", gotArguments, want)
	}
}

func TestReexecIfRequestedDoesNothingWithoutRestartFlag(t *testing.T) {
	called := false
	err := reexecIfRequested(restartModel{}, "/repo/setup", "/repo", func(string, []string, []string) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("exec called without a restart request")
	}
}

func TestValidateRepositoryAllowsMissingPackagesSoDoctorCanRun(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := validateRepository(repo); err != nil {
		t.Fatalf("validateRepository blocked Doctor: %v", err)
	}
}

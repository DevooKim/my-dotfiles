package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"github.com/DevooKim/my-dotfiles/internal/dotfiles"
)

type uiGitRunner struct {
	calls [][]string
}

func (r *uiGitRunner) Run(_ context.Context, dir, name string, args ...string) (string, error) {
	r.calls = append(r.calls, append([]string{dir, name}, args...))
	if len(args) > 0 && args[0] == "rev-parse" {
		return "true\n", nil
	}
	return "", nil
}

func uiFixture(t *testing.T) (string, string, []dotfiles.Package) {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	writeUIFile(t, filepath.Join(repo, "zsh", ".zshrc"), "zsh")
	writeUIFile(t, filepath.Join(repo, "wezterm", ".config", "wezterm", "wezterm.lua"), "wezterm")
	if err := os.Symlink(filepath.Join(repo, "zsh", ".zshrc"), filepath.Join(home, ".zshrc")); err != nil {
		t.Fatal(err)
	}
	return repo, home, []dotfiles.Package{
		{Name: "zsh", Description: "Zsh shell", Command: "zsh"},
		{Name: "wezterm", Description: "WezTerm terminal", Command: "wezterm"},
	}
}

func writeUIFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testDependencies(runner dotfiles.CommandRunner) Dependencies {
	return Dependencies{
		Runner: runner,
		Lookup: func(string) (string, error) { return "", errors.New("missing") },
		Now:    func() time.Time { return time.Date(2026, 8, 21, 1, 2, 3, 0, time.Local) },
	}
}

func keyMsg(code rune, text string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Text: text})
}

func updateModel(t *testing.T, model *Model, msg tea.Msg) (*Model, tea.Cmd) {
	t.Helper()
	next, cmd := model.Update(msg)
	updated, ok := next.(*Model)
	if !ok {
		t.Fatalf("Update returned %T, want *Model", next)
	}
	return updated, cmd
}

func TestInstallOpensFlatPackageSelectionWithNotInstalledDefaults(t *testing.T) {
	repo, home, packages := uiFixture(t)
	model, err := NewModel(repo, home, packages, testDependencies(&uiGitRunner{}), false)
	if err != nil {
		t.Fatal(err)
	}

	model, _ = updateModel(t, model, keyMsg(tea.KeyEnter, ""))
	if model.stage != stagePackages || model.action != ActionInstall {
		t.Fatalf("stage/action = %v/%v, want packages/install", model.stage, model.action)
	}
	if model.selected["zsh"] || !model.selected["wezterm"] {
		t.Fatalf("default selection = %#v, want only not-installed wezterm", model.selected)
	}
}

func TestPackageSelectionSupportsSelectAllClearAllAndToggle(t *testing.T) {
	repo, home, packages := uiFixture(t)
	model, _ := NewModel(repo, home, packages, testDependencies(&uiGitRunner{}), false)
	model, _ = updateModel(t, model, keyMsg(tea.KeyEnter, ""))

	model, _ = updateModel(t, model, keyMsg('a', "a"))
	if !model.selected["zsh"] || !model.selected["wezterm"] {
		t.Fatalf("select all = %#v", model.selected)
	}
	model, _ = updateModel(t, model, keyMsg('n', "n"))
	if model.selected["zsh"] || model.selected["wezterm"] {
		t.Fatalf("clear all = %#v", model.selected)
	}
	model, _ = updateModel(t, model, keyMsg(tea.KeySpace, " "))
	if !model.selected["zsh"] {
		t.Fatalf("space did not toggle current package: %#v", model.selected)
	}
}

func TestConflictDefaultsToSkipAndCanSwitchToBackup(t *testing.T) {
	repo, home, packages := uiFixture(t)
	weztermTarget := filepath.Join(home, ".config", "wezterm", "wezterm.lua")
	writeUIFile(t, weztermTarget, "user")
	model, _ := NewModel(repo, home, packages, testDependencies(&uiGitRunner{}), false)
	model, _ = updateModel(t, model, keyMsg(tea.KeyEnter, ""))
	model, _ = updateModel(t, model, keyMsg(tea.KeyDown, ""))
	model, _ = updateModel(t, model, keyMsg(tea.KeySpace, " "))
	model, _ = updateModel(t, model, keyMsg(tea.KeyEnter, ""))

	if model.stage != stageConflicts || model.conflictChoices[weztermTarget] != dotfiles.SkipConflict {
		t.Fatalf("stage/choice = %v/%v", model.stage, model.conflictChoices[weztermTarget])
	}
	model, _ = updateModel(t, model, keyMsg('b', "b"))
	if model.conflictChoices[weztermTarget] != dotfiles.BackUpConflict {
		t.Fatalf("backup choice = %v", model.conflictChoices[weztermTarget])
	}
	model, _ = updateModel(t, model, keyMsg(tea.KeyEnter, ""))
	if model.stage != stagePreview {
		t.Fatalf("stage = %v, want preview", model.stage)
	}
}

func TestPreviewRequiresConfirmationAndShowsResult(t *testing.T) {
	repo, home, packages := uiFixture(t)
	model, _ := NewModel(repo, home, packages, testDependencies(&uiGitRunner{}), false)
	model, _ = updateModel(t, model, keyMsg(tea.KeyEnter, ""))
	model, _ = updateModel(t, model, keyMsg(tea.KeyEnter, ""))
	if model.stage != stagePreview {
		t.Fatalf("stage = %v, want preview", model.stage)
	}
	model, _ = updateModel(t, model, keyMsg(tea.KeyEnter, ""))
	if model.stage != stageConfirm {
		t.Fatalf("stage = %v, want confirm", model.stage)
	}
	model, cmd := updateModel(t, model, keyMsg('y', "y"))
	if cmd == nil || model.stage != stageRunning {
		t.Fatalf("stage/cmd = %v/%v, want running/non-nil", model.stage, cmd)
	}
	model, _ = updateModel(t, model, cmd())
	if model.stage != stageResult || model.operationErr != nil {
		t.Fatalf("stage/error = %v/%v, want successful result", model.stage, model.operationErr)
	}
}

func TestQuitCancelsWithoutMutation(t *testing.T) {
	repo, home, packages := uiFixture(t)
	model, _ := NewModel(repo, home, packages, testDependencies(&uiGitRunner{}), false)
	model, cmd := updateModel(t, model, keyMsg('q', "q"))
	if !model.quitting || cmd == nil {
		t.Fatalf("quitting/cmd = %v/%v", model.quitting, cmd)
	}
}

func TestApplicationInterruptCancelsRunningOperationWithoutQuitting(t *testing.T) {
	canceled := false
	model := &Model{stage: stageRunning, cancel: func() { canceled = true }}
	updated, cmd := model.Update(InterruptMsg{})
	got := updated.(*Model)
	if !canceled || got.quitting || !got.quitAfterOperation || cmd != nil {
		t.Fatalf("canceled/quitting/after/cmd = %v/%v/%v/%v", canceled, got.quitting, got.quitAfterOperation, cmd)
	}
	updated, cmd = got.Update(operationDoneMsg{err: context.Canceled})
	got = updated.(*Model)
	if !got.quitting || cmd == nil {
		t.Fatalf("quitting/cmd after rollback = %v/%v", got.quitting, cmd)
	}
}

func TestUpdateQuitsForReexecOnlyAfterBubbleTeaCanRestoreTerminal(t *testing.T) {
	repo, home, packages := uiFixture(t)
	runner := &uiGitRunner{}
	model, _ := NewModel(repo, home, packages, testDependencies(runner), false)
	model.actionCursor = 2
	model, _ = updateModel(t, model, keyMsg(tea.KeyEnter, ""))
	if model.stage != stageUpdateConfirm {
		t.Fatalf("stage = %v, want update confirm", model.stage)
	}
	model, cmd := updateModel(t, model, keyMsg('y', "y"))
	if cmd == nil || model.stage != stageRunning {
		t.Fatalf("stage/cmd = %v/%v", model.stage, cmd)
	}
	model, quitCmd := updateModel(t, model, cmd())
	if !model.RestartRequested() || quitCmd == nil || model.operationErr != nil {
		t.Fatalf("restart/cmd/error = %v/%v/%v", model.RestartRequested(), quitCmd, model.operationErr)
	}
}

func TestPostUpdateApplyFailureKeepsGitSuccessVisible(t *testing.T) {
	repo, home, packages := uiFixture(t)
	model, err := NewModel(repo, home, packages, testDependencies(&uiGitRunner{}), true)
	if err != nil {
		t.Fatal(err)
	}
	model, _ = updateModel(t, model, operationDoneMsg{err: errors.New("link failed")})
	view := model.View().Content
	if !strings.Contains(view, "Repository updated") || !strings.Contains(view, "settings apply failed") {
		t.Fatalf("post-update failure view does not separate outcomes:\n%s", view)
	}
}

func TestResultDistinguishesIncompleteRollback(t *testing.T) {
	model := &Model{
		stage: stageResult, resultTitle: "Operation failed", operationErr: errors.New("apply failed"),
		operationResult: dotfiles.ApplyResult{RolledBack: true, RollbackErrors: []error{errors.New("restore failed")}},
		help:            help.New(), keys: defaultKeyMap(),
	}
	view := model.View().Content
	if !strings.Contains(view, "Rollback incomplete") || strings.Contains(view, "Filesystem changes rolled back") {
		t.Fatalf("rollback result is mislabeled:\n%s", view)
	}
}

func TestResultReportsSkippedPaths(t *testing.T) {
	model := &Model{
		stage: stageResult, resultTitle: "Operation complete",
		plan: dotfiles.Plan{Skipped: []dotfiles.Mapping{{Target: "/one"}, {Target: "/two"}}},
		help: help.New(), keys: defaultKeyMap(),
	}
	if view := model.View().Content; !strings.Contains(view, "2 path(s) skipped") {
		t.Fatalf("result does not report skipped paths:\n%s", view)
	}
}

func TestDoctorRemainsAvailableWhenPackageSourceIsMissing(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	model, err := NewModel(repo, home, []dotfiles.Package{{Name: "missing"}}, testDependencies(&uiGitRunner{}), false)
	if err != nil {
		t.Fatalf("NewModel blocked Doctor: %v", err)
	}
	model.actionCursor = int(ActionDoctor)
	model, _ = updateModel(t, model, keyMsg(tea.KeyEnter, ""))
	if model.stage != stageResult || len(model.findings) == 0 || model.findings[1].Code != "source-invalid" {
		t.Fatalf("Doctor findings = %#v", model.findings)
	}
}

func TestResetClearsWorkflowSpecificResultState(t *testing.T) {
	model := &Model{
		packages: nil, selected: map[string]bool{}, inspections: map[string]dotfiles.Inspection{},
		inspectionErrors: map[string]error{}, plan: dotfiles.Plan{Skipped: []dotfiles.Mapping{{Target: "/old"}}},
		operationResult: dotfiles.ApplyResult{RolledBack: true}, afterUpdate: true,
	}
	model.resetToActions()
	if len(model.plan.Skipped) != 0 || model.operationResult.RolledBack || model.afterUpdate {
		t.Fatalf("workflow state survived reset: plan=%+v result=%+v afterUpdate=%v", model.plan, model.operationResult, model.afterUpdate)
	}
}

func TestPostUpdatePreviewFailureKeepsGitSuccessVisible(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	model, err := NewModel(repo, home, []dotfiles.Package{{Name: "missing"}}, testDependencies(&uiGitRunner{}), true)
	if err != nil {
		t.Fatal(err)
	}
	model, _ = updateModel(t, model, keyMsg(tea.KeySpace, " "))
	model, _ = updateModel(t, model, keyMsg(tea.KeyEnter, ""))
	if model.stage != stageResult || !strings.Contains(model.resultTitle, "Repository updated") {
		t.Fatalf("stage/title = %v/%q", model.stage, model.resultTitle)
	}
}

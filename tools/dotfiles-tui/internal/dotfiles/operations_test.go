package dotfiles

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestApplyPlanCreatesMissingLeafLinks(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	plan, err := BuildApplyPlan(repo, home, filepath.Join(home, ".dotfiles-backups", "stamp"), []Package{pkg}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), plan, ApplyOptions{}); err != nil {
		t.Fatal(err)
	}
	assertSameFile(t, filepath.Join(repo, "zsh", ".zshrc"), filepath.Join(home, ".zshrc"))
	assertSameFile(t, filepath.Join(repo, "zsh", ".config", "zsh", "alias.zsh"), filepath.Join(home, ".config", "zsh", "alias.zsh"))
}

func TestApplyPlanSkipsConflictsByDefault(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	writeFixtureFile(t, filepath.Join(home, ".zshrc"), "keep me")

	plan, err := BuildApplyPlan(repo, home, filepath.Join(home, ".dotfiles-backups", "stamp"), []Package{pkg}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), plan, ApplyOptions{}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "keep me" {
		t.Fatalf("conflict content = %q, want %q", content, "keep me")
	}
	assertSameFile(t, filepath.Join(repo, "zsh", ".config", "zsh", "alias.zsh"), filepath.Join(home, ".config", "zsh", "alias.zsh"))
}

func TestApplyPlanBacksUpApprovedConflict(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	target := filepath.Join(home, ".zshrc")
	backupRoot := filepath.Join(home, ".dotfiles-backups", "20260821-010203")
	writeFixtureFile(t, target, "back me up")

	plan, err := BuildApplyPlan(repo, home, backupRoot, []Package{pkg}, map[string]ConflictChoice{target: BackUpConflict})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), plan, ApplyOptions{}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(backupRoot, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "back me up" {
		t.Fatalf("backup content = %q", content)
	}
	assertSameFile(t, filepath.Join(repo, "zsh", ".zshrc"), target)
}

func TestApplyRollsBackAfterInjectedFailure(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	if err := os.MkdirAll(filepath.Join(home, ".config", "zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildApplyPlan(repo, home, filepath.Join(home, ".dotfiles-backups", "stamp"), []Package{pkg}, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply(context.Background(), plan, ApplyOptions{FailAfter: 1})
	if err == nil {
		t.Fatal("Apply succeeded, want injected failure")
	}
	if !result.RolledBack || len(result.RollbackErrors) != 0 {
		t.Fatalf("result = %+v, want clean rollback", result)
	}
	assertAbsent(t, filepath.Join(home, ".zshrc"))
	assertAbsent(t, filepath.Join(home, ".config", "zsh", "alias.zsh"))
}

func TestApplyRollsBackWhenContextIsCanceledMidTransaction(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	if err := os.MkdirAll(filepath.Join(home, ".config", "zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildApplyPlan(repo, home, filepath.Join(home, ".dotfiles-backups", "stamp"), []Package{pkg}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result, err := Apply(ctx, plan, ApplyOptions{AfterMutation: func(applied int) {
		if applied == 1 {
			cancel()
		}
	}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Apply error = %v, want context cancellation", err)
	}
	if !result.RolledBack || len(result.RollbackErrors) != 0 {
		t.Fatalf("result = %+v, want clean rollback", result)
	}
	assertAbsent(t, filepath.Join(home, ".zshrc"))
	assertAbsent(t, filepath.Join(home, ".config", "zsh", "alias.zsh"))
}

func TestApplyRejectsParentDirectoryReplacementAfterPreview(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	config := filepath.Join(home, ".config")
	outside := filepath.Join(filepath.Dir(home), "outside")
	if err := os.MkdirAll(config, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(outside, "zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildApplyPlan(repo, home, filepath.Join(home, ".dotfiles-backups", "stamp"), []Package{pkg}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(config, config+"-old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, config); err != nil {
		t.Fatal(err)
	}

	if _, err := Apply(context.Background(), plan, ApplyOptions{}); err == nil {
		t.Fatal("Apply succeeded after a previewed parent directory was replaced")
	}
	assertAbsent(t, filepath.Join(outside, "zsh", "alias.zsh"))
	assertAbsent(t, filepath.Join(home, ".zshrc"))
}

func TestApplyRevalidatesParentsImmediatelyBeforeCreatingLink(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	config := filepath.Join(home, ".config")
	outside := filepath.Join(filepath.Dir(home), "outside-during-apply")
	if err := os.MkdirAll(config, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(outside, "zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildApplyPlan(repo, home, filepath.Join(home, ".dotfiles-backups", "stamp"), []Package{pkg}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Apply(context.Background(), plan, ApplyOptions{AfterMutation: func(applied int) {
		if applied != 1 {
			return
		}
		if renameErr := os.Rename(config, config+"-old"); renameErr != nil {
			t.Fatalf("replace parent: %v", renameErr)
		}
		if linkErr := os.Symlink(outside, config); linkErr != nil {
			t.Fatalf("replace parent with symlink: %v", linkErr)
		}
	}})
	if err == nil {
		t.Fatal("Apply succeeded after a validated parent changed during the transaction")
	}
	assertAbsent(t, filepath.Join(outside, "zsh", "alias.zsh"))
	if info, statErr := os.Stat(filepath.Join(outside, "zsh")); statErr != nil || !info.IsDir() {
		t.Fatalf("rollback touched the replacement parent: info=%v err=%v", info, statErr)
	}
}

func TestApplyRevalidatesCorrectMappingsBeforeMutation(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	symlinkFixture(t, filepath.Join(repo, "zsh", ".zshrc"), filepath.Join(home, ".zshrc"))
	plan, err := BuildApplyPlan(repo, home, filepath.Join(home, ".dotfiles-backups", "stamp"), []Package{pkg}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(home, ".zshrc")); err != nil {
		t.Fatal(err)
	}
	symlinkFixture(t, filepath.Join(home, "user-target"), filepath.Join(home, ".zshrc"))

	if _, err := Apply(context.Background(), plan, ApplyOptions{}); err == nil {
		t.Fatal("Apply succeeded after a correct mapping changed")
	}
	assertAbsent(t, filepath.Join(home, ".config", "zsh", "alias.zsh"))
}

func TestRollbackDoesNotRestoreBackupThroughReplacedParent(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, ".config")
	backupDir := filepath.Join(home, ".backup")
	target := filepath.Join(config, "settings")
	backup := filepath.Join(backupDir, "settings")
	outside := filepath.Join(filepath.Dir(home), "outside-backup-rollback")
	for _, directory := range []string{config, backupDir, outside} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFixtureFile(t, target, "user settings")
	targetInfo, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	directories := make(map[string]os.FileInfo)
	for _, directory := range []string{home, config, backupDir} {
		info, statErr := os.Lstat(directory)
		if statErr != nil {
			t.Fatal(statErr)
		}
		directories[directory] = info
	}
	plan := Plan{
		Root:        home,
		directories: directories,
		Operations: []Operation{
			{Kind: MoveToBackup, Path: target, Destination: backup, expectedInfo: targetInfo},
			{Kind: OperationKind("fail"), Path: filepath.Join(home, "failure")},
		},
	}

	result, err := Apply(context.Background(), plan, ApplyOptions{AfterMutation: func(applied int) {
		if applied != 1 {
			return
		}
		if renameErr := os.Rename(config, config+"-old"); renameErr != nil {
			t.Fatalf("replace original parent: %v", renameErr)
		}
		if linkErr := os.Symlink(outside, config); linkErr != nil {
			t.Fatalf("link replacement parent: %v", linkErr)
		}
	}})
	if err == nil || len(result.RollbackErrors) == 0 {
		t.Fatalf("result/error = %+v/%v, want failed apply and incomplete safe rollback", result, err)
	}
	assertAbsent(t, filepath.Join(outside, "settings"))
	content, readErr := os.ReadFile(backup)
	if readErr != nil || string(content) != "user settings" {
		t.Fatalf("backup was not preserved: content=%q err=%v", content, readErr)
	}
}

func TestRemovePlanOnlyDeletesManagedLinks(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	symlinkFixture(t, filepath.Join(repo, "zsh", ".zshrc"), filepath.Join(home, ".zshrc"))
	symlinkFixture(t, filepath.Join(home, "user-target"), filepath.Join(home, ".config", "zsh", "alias.zsh"))

	plan, err := BuildRemovePlan(repo, home, []Package{pkg})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), plan, ApplyOptions{}); err != nil {
		t.Fatal(err)
	}
	assertAbsent(t, filepath.Join(home, ".zshrc"))
	if info, err := os.Lstat(filepath.Join(home, ".config", "zsh", "alias.zsh")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("wrong-target user link was removed: info=%v err=%v", info, err)
	}
}

func TestRemovePlanRecognizesStowDirectoryLink(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	symlinkFixture(t, filepath.Join(repo, "zsh", ".zshrc"), filepath.Join(home, ".zshrc"))
	symlinkFixture(t, filepath.Join(repo, "zsh", ".config", "zsh"), filepath.Join(home, ".config", "zsh"))

	plan, err := BuildRemovePlan(repo, home, []Package{pkg})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), plan, ApplyOptions{}); err != nil {
		t.Fatal(err)
	}
	assertAbsent(t, filepath.Join(home, ".zshrc"))
	assertAbsent(t, filepath.Join(home, ".config", "zsh"))
}

type fakeGitRunner struct {
	status string
	calls  [][]string
}

func (f *fakeGitRunner) Run(_ context.Context, dir, name string, args ...string) (string, error) {
	call := append([]string{dir, name}, args...)
	f.calls = append(f.calls, call)
	if len(args) > 0 && args[0] == "rev-parse" {
		return "true\n", nil
	}
	if len(args) > 0 && args[0] == "status" {
		return f.status, nil
	}
	return "", nil
}

func TestGitPreflightRejectsDirtyWorktree(t *testing.T) {
	runner := &fakeGitRunner{status: "?? dirty.txt\n"}
	if err := GitPreflight(context.Background(), "/repo", runner); err == nil {
		t.Fatal("GitPreflight succeeded, want dirty-worktree error")
	}
}

func TestGitUpdateUsesOnlyFastForwardPull(t *testing.T) {
	runner := &fakeGitRunner{}
	if err := GitUpdate(context.Background(), "/repo", runner); err != nil {
		t.Fatal(err)
	}
	want := []string{"/repo", "git", "pull", "--ff-only"}
	if got := runner.calls[len(runner.calls)-1]; !reflect.DeepEqual(got, want) {
		t.Fatalf("last Git call = %#v, want %#v", got, want)
	}
}

func TestDoctorReportsLinksCommandsAndReferences(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	writeFixtureFile(t, filepath.Join(home, ".zshrc"), "conflict")
	hammerspoon := Package{
		Name:       "hammerspoon",
		Command:    "hs-that-is-missing",
		References: []string{".hammerspoon/modules/input/tmux_lang.lua"},
	}
	writeFixtureFile(t, filepath.Join(repo, "hammerspoon", ".hammerspoon", "init.lua"), "require modules.input.tmux_lang")

	findings := Doctor(repo, home, []Package{pkg, hammerspoon}, func(string) (string, error) {
		return "", errors.New("missing")
	})
	joined := make([]string, 0, len(findings))
	for _, finding := range findings {
		joined = append(joined, string(finding.Severity)+" "+finding.Package+" "+finding.Code+" "+finding.Detail)
	}
	report := strings.Join(joined, "\n")
	for _, want := range []string{
		"warning zsh link-status conflict",
		"warning hammerspoon command-missing hs-that-is-missing",
		"warning hammerspoon reference-missing .hammerspoon/modules/input/tmux_lang.lua",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("Doctor report missing %q:\n%s", want, report)
		}
	}
}

func assertSameFile(t *testing.T, first, second string) {
	t.Helper()
	firstInfo, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(firstInfo, secondInfo) {
		t.Fatalf("%q and %q are not the same file", first, second)
	}
}

func assertAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("path %q exists or could not be checked: %v", path, err)
	}
}

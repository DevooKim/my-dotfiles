package dotfiles

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CommandRunner makes Git behavior deterministic in tests.
type CommandRunner interface {
	Run(context.Context, string, string, ...string) (string, error)
}

// OSCommandRunner executes commands without a shell.
type OSCommandRunner struct{}

func (OSCommandRunner) Run(ctx context.Context, dir, name string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

// GitPreflight requires a Git worktree with no tracked or untracked changes.
func GitPreflight(ctx context.Context, repo string, runner CommandRunner) error {
	inside, err := runner.Run(ctx, repo, "git", "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(inside) != "true" {
		return errorsWithContext("dotfiles directory is not a Git worktree", err)
	}
	status, err := runner.Run(ctx, repo, "git", "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return fmt.Errorf("inspect Git worktree: %w", err)
	}
	if strings.TrimSpace(status) != "" {
		return fmt.Errorf("dotfiles worktree has uncommitted changes:\n%s", strings.TrimSpace(status))
	}
	return nil
}

// GitUpdate performs only a fast-forward pull after preflight.
func GitUpdate(ctx context.Context, repo string, runner CommandRunner) error {
	if err := GitPreflight(ctx, repo, runner); err != nil {
		return err
	}
	if _, err := runner.Run(ctx, repo, "git", "pull", "--ff-only"); err != nil {
		return fmt.Errorf("fast-forward update failed: %w", err)
	}
	return nil
}

func errorsWithContext(message string, err error) error {
	if err == nil {
		return fmt.Errorf("%s", message)
	}
	return fmt.Errorf("%s: %w", message, err)
}

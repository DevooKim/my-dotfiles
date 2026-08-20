package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/DevooKim/my-dotfiles/internal/dotfiles"
	"github.com/DevooKim/my-dotfiles/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "dotfiles setup:", err)
		os.Exit(1)
	}
}

func run() error {
	repoFlag := flag.String("repo", "", "dotfiles repository root")
	afterUpdate := flag.Bool("after-update", false, "open reapply after a repository update")
	flag.Parse()

	repo, err := repositoryRoot(*repoFlag)
	if err != nil {
		return err
	}
	packages := dotfiles.Catalog()
	if err := validateRepository(repo); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	operationContext, cancelOperations := context.WithCancel(context.Background())
	defer cancelOperations()
	model, err := ui.NewModel(repo, home, packages, ui.Dependencies{
		Context: operationContext,
		Runner:  dotfiles.OSCommandRunner{},
	}, *afterUpdate)
	if err != nil {
		return fmt.Errorf("inspect dotfiles: %w", err)
	}

	program := tea.NewProgram(model, tea.WithoutSignalHandler())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	programDone := make(chan struct{})
	go func() {
		for {
			select {
			case <-signals:
				program.Send(ui.InterruptMsg{})
			case <-programDone:
				return
			}
		}
	}()

	returned, err := program.Run()
	close(programDone)
	if err != nil {
		return fmt.Errorf("run terminal UI: %w", err)
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}
	return reexecIfRequested(returned, executable, repo, syscall.Exec)
}

func repositoryRoot(explicit string) (string, error) {
	if explicit != "" {
		root, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolve repository root: %w", err)
		}
		return root, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	return repositoryRootFromExecutable(executable)
}

func repositoryRootFromExecutable(executable string) (string, error) {
	absolute, err := filepath.Abs(executable)
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	binDirectory := filepath.Dir(absolute)
	if filepath.Base(binDirectory) != "bin" {
		return "", fmt.Errorf("executable must be located in the repository bin directory")
	}
	return filepath.Dir(binDirectory), nil
}

type execFunc func(string, []string, []string) error

func reexecIfRequested(returned tea.Model, executable, repo string, execute execFunc) error {
	restartable, ok := returned.(interface{ RestartRequested() bool })
	if !ok || !restartable.RestartRequested() {
		return nil
	}
	arguments := []string{executable, "--repo", repo, "--after-update"}
	if err := execute(executable, arguments, os.Environ()); err != nil {
		return fmt.Errorf("restart updated TUI: %w", err)
	}
	return nil
}

func validateRepository(repo string) error {
	info, err := os.Stat(repo)
	if err != nil {
		return fmt.Errorf("repository root is unavailable: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("repository root is not a directory")
	}
	return nil
}

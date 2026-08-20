package dotfiles

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ApplyOptions contains deterministic failure injection used by tests.
type ApplyOptions struct {
	FailAfter     int
	AfterMutation func(int)
}

// ApplyResult describes whether a failed transaction returned to its start state.
type ApplyResult struct {
	Applied        int
	RolledBack     bool
	RollbackErrors []error
}

type undo func() error

// Apply executes a plan and reverses completed mutations on any failure.
func Apply(ctx context.Context, plan Plan, options ApplyOptions) (ApplyResult, error) {
	result := ApplyResult{}
	if err := validateKeptMappings(plan.Kept); err != nil {
		return result, err
	}
	trackedDirectories := make(map[string]os.FileInfo, len(plan.directories))
	for path, expected := range plan.directories {
		trackedDirectories[path] = expected
	}
	if err := validateTrackedDirectories(trackedDirectories); err != nil {
		return result, err
	}
	var journal []undo
	for _, operation := range plan.Operations {
		if err := ctx.Err(); err != nil {
			return rollback(result, journal, err)
		}
		if err := validateOperationParents(operation, plan.Root, trackedDirectories); err != nil {
			return rollback(result, journal, err)
		}
		inverse, mutated, err := applyOperation(operation)
		if mutated {
			journal = append(journal, inverse)
			result.Applied++
		}
		if err != nil {
			return rollback(result, journal, err)
		}
		if !mutated {
			if operation.Kind == MakeDirectory {
				trackedDirectories[operation.Path] = operation.expectedInfo
			}
			continue
		}
		if operation.Kind == MakeDirectory {
			createdInfo, err := os.Lstat(operation.Path)
			if err != nil {
				return rollback(result, journal, fmt.Errorf("inspect created directory %q: %w", operation.Path, err))
			}
			trackedDirectories[operation.Path] = createdInfo
		}
		if options.AfterMutation != nil {
			options.AfterMutation(result.Applied)
		}
		if err := ctx.Err(); err != nil {
			return rollback(result, journal, err)
		}
		if options.FailAfter > 0 && result.Applied >= options.FailAfter {
			return rollback(result, journal, errors.New("injected apply failure"))
		}
	}
	return result, nil
}

func validateTrackedDirectories(directories map[string]os.FileInfo) error {
	for path, expected := range directories {
		current, err := os.Lstat(path)
		if err != nil || !current.IsDir() || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(current, expected) {
			return fmt.Errorf("directory %q changed after preview", path)
		}
	}
	return nil
}

func validateOperationParents(operation Operation, root string, directories map[string]os.FileInfo) error {
	paths := []string{operation.Path}
	if operation.Kind == MoveToBackup {
		paths = append(paths, operation.Destination)
	}
	for _, path := range paths {
		relative, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil || relative == ".." || filepath.IsAbs(relative) || startsWithParent(relative) {
			return fmt.Errorf("operation path %q is outside home", path)
		}
		cursor := root
		if err := validateTrackedDirectory(cursor, directories); err != nil {
			return err
		}
		for _, component := range splitPath(relative) {
			cursor = filepath.Join(cursor, component)
			if err := validateTrackedDirectory(cursor, directories); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTrackedDirectory(path string, directories map[string]os.FileInfo) error {
	expected, ok := directories[path]
	if !ok {
		return fmt.Errorf("directory %q was not part of the preview", path)
	}
	current, err := os.Lstat(path)
	if err != nil || !current.IsDir() || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(current, expected) {
		return fmt.Errorf("directory %q changed during apply", path)
	}
	return nil
}

func applyOperation(operation Operation) (undo, bool, error) {
	switch operation.Kind {
	case MakeDirectory:
		info, err := os.Lstat(operation.Path)
		if operation.expectedInfo != nil {
			if err != nil || !info.IsDir() || !os.SameFile(info, operation.expectedInfo) {
				return nil, false, fmt.Errorf("directory %q changed after preview", operation.Path)
			}
			return nil, false, nil
		}
		if err == nil {
			return nil, false, fmt.Errorf("directory path %q became occupied", operation.Path)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, false, err
		}
		parent := filepath.Dir(operation.Path)
		parentInfo, err := os.Lstat(parent)
		if err != nil {
			return nil, false, fmt.Errorf("inspect directory parent %q: %w", parent, err)
		}
		if err := os.Mkdir(operation.Path, 0o755); err != nil {
			return nil, false, fmt.Errorf("create directory %q: %w", operation.Path, err)
		}
		createdInfo, snapshotErr := os.Lstat(operation.Path)
		if snapshotErr != nil {
			return func() error {
				return fmt.Errorf("cannot safely roll back directory %q without its identity", operation.Path)
			}, true, fmt.Errorf("inspect created directory %q: %w", operation.Path, snapshotErr)
		}
		return func() error {
			currentParent, err := os.Lstat(parent)
			if err != nil || !os.SameFile(currentParent, parentInfo) {
				return fmt.Errorf("directory parent %q changed before rollback", parent)
			}
			current, err := os.Lstat(operation.Path)
			if err != nil || !current.IsDir() || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(current, createdInfo) {
				return fmt.Errorf("created directory %q changed before rollback", operation.Path)
			}
			return os.Remove(operation.Path)
		}, true, nil

	case MoveToBackup:
		current, err := os.Lstat(operation.Path)
		if err != nil || !os.SameFile(current, operation.expectedInfo) {
			return nil, false, fmt.Errorf("conflict %q changed after preview", operation.Path)
		}
		if _, err := os.Lstat(operation.Destination); !errors.Is(err, os.ErrNotExist) {
			return nil, false, fmt.Errorf("backup destination %q is occupied", operation.Destination)
		}
		originalParent := filepath.Dir(operation.Path)
		originalParentInfo, err := os.Lstat(originalParent)
		if err != nil {
			return nil, false, fmt.Errorf("inspect conflict parent %q: %w", originalParent, err)
		}
		if err := os.Rename(operation.Path, operation.Destination); err != nil {
			return nil, false, fmt.Errorf("back up %q: %w", operation.Path, err)
		}
		return func() error {
			currentParent, err := os.Lstat(originalParent)
			if err != nil || !os.SameFile(currentParent, originalParentInfo) {
				return fmt.Errorf("conflict parent %q changed before rollback", originalParent)
			}
			current, err := os.Lstat(operation.Destination)
			if err != nil || !os.SameFile(current, operation.expectedInfo) {
				return fmt.Errorf("backup %q changed before rollback", operation.Destination)
			}
			return os.Rename(operation.Destination, operation.Path)
		}, true, nil

	case CreateLink:
		currentSource, err := os.Lstat(operation.Source)
		if err != nil || !os.SameFile(currentSource, operation.sourceInfo) {
			return nil, false, fmt.Errorf("source %q changed after preview", operation.Source)
		}
		if _, err := os.Lstat(operation.Path); !errors.Is(err, os.ErrNotExist) {
			return nil, false, fmt.Errorf("link target %q became occupied", operation.Path)
		}
		parent := filepath.Dir(operation.Path)
		parentInfo, err := os.Lstat(parent)
		if err != nil {
			return nil, false, fmt.Errorf("inspect link parent %q: %w", parent, err)
		}
		if err := os.Symlink(operation.Source, operation.Path); err != nil {
			return nil, false, fmt.Errorf("link %q: %w", operation.Path, err)
		}
		createdInfo, snapshotErr := os.Lstat(operation.Path)
		if snapshotErr != nil {
			return func() error {
				return fmt.Errorf("cannot safely roll back link %q without its identity", operation.Path)
			}, true, fmt.Errorf("inspect created link %q: %w", operation.Path, snapshotErr)
		}
		return func() error {
			currentParent, err := os.Lstat(parent)
			if err != nil || !os.SameFile(currentParent, parentInfo) {
				return fmt.Errorf("link parent %q changed before rollback", parent)
			}
			current, err := os.Lstat(operation.Path)
			if err != nil || !os.SameFile(current, createdInfo) {
				return fmt.Errorf("created link %q changed before rollback", operation.Path)
			}
			link, err := os.Readlink(operation.Path)
			if err != nil || link != operation.Source {
				return fmt.Errorf("created link %q changed before rollback", operation.Path)
			}
			return os.Remove(operation.Path)
		}, true, nil

	case RemoveLink:
		current, err := os.Lstat(operation.Path)
		if err != nil || !os.SameFile(current, operation.expectedInfo) {
			return nil, false, fmt.Errorf("managed link %q changed after preview", operation.Path)
		}
		link, err := os.Readlink(operation.Path)
		if err != nil || link != operation.ExpectedLink {
			return nil, false, fmt.Errorf("managed link %q changed after preview", operation.Path)
		}
		parentInfo, err := os.Lstat(filepath.Dir(operation.Path))
		if err != nil {
			return nil, false, fmt.Errorf("inspect link parent %q: %w", filepath.Dir(operation.Path), err)
		}
		if err := os.Remove(operation.Path); err != nil {
			return nil, false, fmt.Errorf("remove link %q: %w", operation.Path, err)
		}
		return func() error {
			currentParent, err := os.Lstat(filepath.Dir(operation.Path))
			if err != nil || !os.SameFile(currentParent, parentInfo) {
				return fmt.Errorf("link parent %q changed before rollback", filepath.Dir(operation.Path))
			}
			return os.Symlink(operation.ExpectedLink, operation.Path)
		}, true, nil
	default:
		return nil, false, fmt.Errorf("unknown operation %q", operation.Kind)
	}
}

func validateKeptMappings(mappings []Mapping) error {
	for _, mapping := range mappings {
		sourceInfo, sourceErr := os.Stat(mapping.Source)
		targetInfo, targetErr := os.Stat(mapping.Target)
		if sourceErr != nil || targetErr != nil || !os.SameFile(sourceInfo, targetInfo) {
			return fmt.Errorf("correct mapping %q changed after preview", mapping.Target)
		}
	}
	return nil
}

func rollback(result ApplyResult, journal []undo, cause error) (ApplyResult, error) {
	result.RolledBack = true
	for index := len(journal) - 1; index >= 0; index-- {
		if err := journal[index](); err != nil {
			result.RollbackErrors = append(result.RollbackErrors, err)
		}
	}
	return result, cause
}

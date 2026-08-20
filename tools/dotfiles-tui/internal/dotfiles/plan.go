package dotfiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ConflictChoice controls what an apply plan does with an occupied path.
type ConflictChoice string

const (
	SkipConflict   ConflictChoice = "skip"
	BackUpConflict ConflictChoice = "backup"
)

// OperationKind is one reversible filesystem mutation.
type OperationKind string

const (
	MakeDirectory OperationKind = "mkdir"
	MoveToBackup  OperationKind = "backup"
	CreateLink    OperationKind = "link"
	RemoveLink    OperationKind = "unlink"
)

// Operation is an immutable mutation with preview-time preconditions.
type Operation struct {
	Kind         OperationKind
	Path         string
	Source       string
	Destination  string
	ExpectedLink string
	expectedInfo os.FileInfo
	sourceInfo   os.FileInfo
}

// Plan is a previewable set of operations and intentionally untouched mappings.
type Plan struct {
	Operations  []Operation
	Kept        []Mapping
	Skipped     []Mapping
	Root        string
	directories map[string]os.FileInfo
}

// BuildApplyPlan creates a leaf-link plan for selected packages.
func BuildApplyPlan(repo, home, backupRoot string, packages []Package, choices map[string]ConflictChoice) (Plan, error) {
	plan, err := newPlan(home)
	if err != nil {
		return Plan{}, err
	}
	plannedDirs := make(map[string]bool)
	plannedBackups := make(map[string]bool)

	for _, pkg := range packages {
		inspection, err := InspectPackage(repo, home, pkg)
		if err != nil {
			return Plan{}, err
		}
		for _, mapping := range inspection.Mappings {
			if mapping.State == TargetCorrect {
				plan.Kept = append(plan.Kept, mapping)
				continue
			}
			if mapping.State == TargetConflict {
				if choices[mapping.ConflictPath] != BackUpConflict {
					plan.Skipped = append(plan.Skipped, mapping)
					continue
				}
				if !plannedBackups[mapping.ConflictPath] {
					info, err := os.Lstat(mapping.ConflictPath)
					if err != nil {
						return Plan{}, fmt.Errorf("inspect conflict %q: %w", mapping.ConflictPath, err)
					}
					relative, err := filepath.Rel(home, mapping.ConflictPath)
					if err != nil || relative == ".." || filepath.IsAbs(relative) || startsWithParent(relative) {
						return Plan{}, fmt.Errorf("conflict path %q is outside home", mapping.ConflictPath)
					}
					backupPath := filepath.Join(backupRoot, relative)
					if err := appendDirectoryChain(&plan, plannedDirs, home, filepath.Dir(backupPath), nil); err != nil {
						return Plan{}, err
					}
					plan.Operations = append(plan.Operations, Operation{
						Kind: MoveToBackup, Path: mapping.ConflictPath, Destination: backupPath, expectedInfo: info,
					})
					plannedBackups[mapping.ConflictPath] = true
				}
			}

			if err := appendDirectoryChain(&plan, plannedDirs, home, filepath.Dir(mapping.Target), plannedBackups); err != nil {
				return Plan{}, err
			}
			sourceInfo, err := os.Lstat(mapping.Source)
			if err != nil {
				return Plan{}, fmt.Errorf("inspect source %q: %w", mapping.Source, err)
			}
			plan.Operations = append(plan.Operations, Operation{
				Kind: CreateLink, Path: mapping.Target, Source: mapping.Source, sourceInfo: sourceInfo,
			})
		}
	}
	return plan, nil
}

// BuildRemovePlan removes only links proven to belong to selected packages.
func BuildRemovePlan(repo, home string, packages []Package) (Plan, error) {
	plan, err := newPlan(home)
	if err != nil {
		return Plan{}, err
	}
	planned := make(map[string]bool)
	for _, pkg := range packages {
		inspection, err := InspectPackage(repo, home, pkg)
		if err != nil {
			return Plan{}, err
		}
		for _, mapping := range inspection.Mappings {
			if mapping.State != TargetCorrect {
				plan.Skipped = append(plan.Skipped, mapping)
				continue
			}
			anchor, err := managedLinkAnchor(repo, home, pkg.Name, mapping.Target)
			if errors.Is(err, os.ErrNotExist) {
				plan.Kept = append(plan.Kept, mapping)
				continue
			}
			if err != nil {
				return Plan{}, err
			}
			if planned[anchor] {
				continue
			}
			if err := captureExistingDirectories(&plan, home, filepath.Dir(anchor)); err != nil {
				return Plan{}, err
			}
			linkText, err := os.Readlink(anchor)
			if err != nil {
				return Plan{}, fmt.Errorf("read managed link %q: %w", anchor, err)
			}
			info, err := os.Lstat(anchor)
			if err != nil {
				return Plan{}, err
			}
			plan.Operations = append(plan.Operations, Operation{
				Kind: RemoveLink, Path: anchor, ExpectedLink: linkText, expectedInfo: info,
			})
			planned[anchor] = true
		}
	}
	return plan, nil
}

func newPlan(home string) (Plan, error) {
	info, err := os.Lstat(home)
	if err != nil {
		return Plan{}, fmt.Errorf("inspect home %q: %w", home, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return Plan{}, fmt.Errorf("home %q must be a real directory", home)
	}
	return Plan{Root: home, directories: map[string]os.FileInfo{home: info}}, nil
}

func appendDirectoryChain(plan *Plan, planned map[string]bool, home, directory string, removedBefore map[string]bool) error {
	relative, err := filepath.Rel(home, directory)
	if err != nil || relative == ".." || filepath.IsAbs(relative) || startsWithParent(relative) {
		return fmt.Errorf("directory %q is outside home", directory)
	}
	if relative == "." {
		return nil
	}
	cursor := home
	for _, component := range splitPath(relative) {
		cursor = filepath.Join(cursor, component)
		if !planned[cursor] {
			var expectedInfo os.FileInfo
			if !pathRemovedBefore(cursor, removedBefore) {
				info, statErr := os.Lstat(cursor)
				switch {
				case statErr == nil && !info.IsDir():
					return fmt.Errorf("directory path %q is occupied", cursor)
				case statErr == nil:
					expectedInfo = info
					plan.directories[cursor] = info
				case !errors.Is(statErr, os.ErrNotExist):
					return fmt.Errorf("inspect directory %q: %w", cursor, statErr)
				}
			}
			plan.Operations = append(plan.Operations, Operation{Kind: MakeDirectory, Path: cursor, expectedInfo: expectedInfo})
			planned[cursor] = true
		}
	}
	return nil
}

func captureExistingDirectories(plan *Plan, home, directory string) error {
	relative, err := filepath.Rel(home, directory)
	if err != nil || relative == ".." || filepath.IsAbs(relative) || startsWithParent(relative) {
		return fmt.Errorf("directory %q is outside home", directory)
	}
	cursor := home
	for _, component := range splitPath(relative) {
		cursor = filepath.Join(cursor, component)
		info, err := os.Lstat(cursor)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("managed link parent %q is not a stable directory", cursor)
		}
		plan.directories[cursor] = info
	}
	return nil
}

func pathRemovedBefore(path string, removals map[string]bool) bool {
	for removal := range removals {
		if path == removal || startsWithParentOf(path, removal) {
			return true
		}
	}
	return false
}

func startsWithParentOf(path, parent string) bool {
	relative, err := filepath.Rel(parent, path)
	return err == nil && relative != "." && !startsWithParent(relative) && !filepath.IsAbs(relative)
}

func splitPath(path string) []string {
	var parts []string
	for path != "." && path != string(filepath.Separator) {
		parts = append([]string{filepath.Base(path)}, parts...)
		path = filepath.Dir(path)
	}
	return parts
}

func startsWithParent(path string) bool {
	return path == ".." || len(path) > 3 && path[:3] == ".."+string(filepath.Separator)
}

func managedLinkAnchor(repo, home, packageName, target string) (string, error) {
	for cursor := target; cursor != home && cursor != filepath.Dir(cursor); cursor = filepath.Dir(cursor) {
		info, err := os.Lstat(cursor)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		relative, err := filepath.Rel(home, cursor)
		if err != nil || startsWithParent(relative) {
			return "", os.ErrNotExist
		}
		expected := filepath.Join(repo, packageName, relative)
		expectedInfo, expectedErr := os.Stat(expected)
		cursorInfo, cursorErr := os.Stat(cursor)
		if expectedErr == nil && cursorErr == nil && os.SameFile(expectedInfo, cursorInfo) {
			return cursor, nil
		}
		return "", os.ErrNotExist
	}
	return "", os.ErrNotExist
}

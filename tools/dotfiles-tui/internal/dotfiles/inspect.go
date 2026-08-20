package dotfiles

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Status summarizes how a package is represented below the target home.
type Status string

const (
	NotInstalled Status = "not installed"
	Installed    Status = "installed"
	Incomplete   Status = "incomplete"
	Conflict     Status = "conflict"
)

// TargetState is the state of one source-to-home mapping.
type TargetState string

const (
	TargetCorrect  TargetState = "correct"
	TargetMissing  TargetState = "missing"
	TargetConflict TargetState = "conflict"
)

// Mapping connects one package source leaf to its destination below HOME.
type Mapping struct {
	Package      string
	Relative     string
	Source       string
	Target       string
	State        TargetState
	ConflictPath string
}

// Inspection contains the aggregate status and individual file mappings.
type Inspection struct {
	Package  Package
	Status   Status
	Mappings []Mapping
}

// InspectPackage discovers package leaves and classifies their destinations.
func InspectPackage(repo, home string, pkg Package) (Inspection, error) {
	mappings, err := inspectMappings(repo, home, pkg)
	if err != nil {
		return Inspection{}, err
	}
	if len(mappings) == 0 {
		return Inspection{}, fmt.Errorf("package %q has no managed files", pkg.Name)
	}

	correct := 0
	missing := 0
	conflicts := 0
	for _, mapping := range mappings {
		switch mapping.State {
		case TargetCorrect:
			correct++
		case TargetMissing:
			missing++
		case TargetConflict:
			conflicts++
		}
	}

	status := Incomplete
	switch {
	case conflicts > 0:
		status = Conflict
	case correct == len(mappings):
		status = Installed
	case missing == len(mappings):
		status = NotInstalled
	}

	return Inspection{Package: pkg, Status: status, Mappings: mappings}, nil
}

func inspectMappings(repo, home string, pkg Package) ([]Mapping, error) {
	packageRoot := filepath.Join(repo, pkg.Name)
	info, err := os.Stat(packageRoot)
	if err != nil {
		return nil, fmt.Errorf("inspect package %q: %w", pkg.Name, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("package %q is not a directory", pkg.Name)
	}

	var mappings []Mapping
	err = filepath.WalkDir(packageRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == packageRoot || entry.IsDir() {
			return nil
		}
		if entry.Type().IsRegular() || entry.Type()&os.ModeSymlink != 0 {
			relative, relErr := filepath.Rel(packageRoot, path)
			if relErr != nil {
				return relErr
			}
			target := filepath.Join(home, relative)
			state, conflictPath, stateErr := inspectTarget(path, target, home)
			if stateErr != nil {
				return stateErr
			}
			mappings = append(mappings, Mapping{
				Package: pkg.Name, Relative: relative, Source: path, Target: target,
				State: state, ConflictPath: conflictPath,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk package %q: %w", pkg.Name, err)
	}
	sort.Slice(mappings, func(i, j int) bool { return mappings[i].Relative < mappings[j].Relative })
	return mappings, nil
}

func inspectTarget(source, target, home string) (TargetState, string, error) {
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return "", "", fmt.Errorf("stat source %q: %w", source, err)
	}
	targetInfo, targetErr := os.Stat(target)
	if targetErr == nil && os.SameFile(sourceInfo, targetInfo) {
		return TargetCorrect, "", nil
	}
	if targetErr != nil && !errors.Is(targetErr, os.ErrNotExist) {
		return "", "", fmt.Errorf("stat target %q: %w", target, targetErr)
	}
	if _, err := os.Lstat(target); err == nil {
		return TargetConflict, target, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", "", fmt.Errorf("inspect target %q: %w", target, err)
	}

	home = filepath.Clean(home)
	for parent := filepath.Dir(target); parent != home && parent != filepath.Dir(parent); parent = filepath.Dir(parent) {
		info, err := os.Lstat(parent)
		switch {
		case err == nil && info.Mode()&os.ModeSymlink != 0:
			return TargetConflict, parent, nil
		case err == nil && !info.IsDir():
			return TargetConflict, parent, nil
		case err != nil && !errors.Is(err, os.ErrNotExist):
			return "", "", fmt.Errorf("inspect parent %q: %w", parent, err)
		}
	}
	return TargetMissing, "", nil
}

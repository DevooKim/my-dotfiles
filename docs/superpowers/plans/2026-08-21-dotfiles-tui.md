# Dotfiles TUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and commit a Go/Bubble Tea TUI plus native executables for Apple Silicon and Intel macOS.

**Architecture:** A testable `tools/dotfiles-tui/internal/dotfiles` package owns filesystem and Git behavior, while `tools/dotfiles-tui/internal/ui` owns Bubble Tea state and rendering. Two static architecture builds are combined into the single root-level universal executable `./setup`; Go is only a development dependency.

**Tech Stack:** Go 1.26.7, `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`, Apple `lipo`.

---

### Task 1: Core catalog and inspection

**Files:**
- Create: `tools/dotfiles-tui/go.mod`
- Create: `tools/dotfiles-tui/internal/dotfiles/catalog.go`
- Create: `tools/dotfiles-tui/internal/dotfiles/inspect.go`
- Create: `tools/dotfiles-tui/internal/dotfiles/inspect_test.go`

- [ ] Write temporary-tree tests for `NotInstalled`, `Installed`, `Incomplete`, and `Conflict`, including Stow parent links.
- [ ] Run `go test ./internal/dotfiles` and verify the package is missing.
- [ ] Implement the fixed catalog, leaf discovery, conflict-anchor detection, and status inspection.
- [ ] Rerun the focused tests and verify GREEN.

### Task 2: Transaction plans and Doctor

**Files:**
- Create: `tools/dotfiles-tui/internal/dotfiles/plan.go`
- Create: `tools/dotfiles-tui/internal/dotfiles/apply.go`
- Create: `tools/dotfiles-tui/internal/dotfiles/doctor.go`
- Create: `tools/dotfiles-tui/internal/dotfiles/operations_test.go`

- [ ] Write tests for link plans, default conflict skipping, backups, managed-only removal, injected-failure rollback, dirty Git rejection, exact `pull --ff-only`, and Doctor warnings.
- [ ] Run focused tests and verify undefined APIs fail compilation.
- [ ] Implement immutable operations, precondition checks, reverse rollback, a Git runner interface, and deterministic Doctor findings.
- [ ] Rerun focused tests and verify GREEN.

### Task 3: Bubble Tea application

**Files:**
- Create: `tools/dotfiles-tui/internal/ui/model.go`
- Create: `tools/dotfiles-tui/internal/ui/model_test.go`
- Create: `tools/dotfiles-tui/cmd/dotfiles-tui/main.go`

- [ ] Write model tests for action choice, flat selection, Select all, Clear all, cancellation, preview, confirmation, and results.
- [ ] Run `go test ./internal/ui` and verify missing APIs fail compilation.
- [ ] Implement the Bubble Tea v2 state machine, Bubbles help, Lip Gloss presentation, async operations, and post-update process replacement.
- [ ] Rerun `go test ./...` and verify GREEN.

### Task 4: Universal native binary

**Files:**
- Create: `setup`
- Create: `tools/dotfiles-tui/build`
- Create: `tools/dotfiles-tui/tests/universal_build_test.sh`

- [ ] Write a test that requires `setup` to be a universal Mach-O with arm64 and x86_64 slices.
- [ ] Run it while `setup` is still a shell launcher and verify RED.
- [ ] Implement the test-first universal build script.
- [ ] Run all tests, build both slices with `CGO_ENABLED=0`, trimmed paths, and stripped symbols, combine them with `lipo`, then inspect both architectures.
- [ ] Run native cancellation against a temporary HOME and verify no filesystem changes.

### Task 5: Review, rebuild, and commit

**Files:**
- Modify only files required by findings.

- [ ] Run `go test ./...`, `go vet ./...`, universal binary tests, smoke test, and `git diff --check`.
- [ ] Review against the approved design and fix every Critical or Important finding.
- [ ] Rebuild both committed binaries after the final source change and verify architectures and checksums.
- [ ] Commit source, tests, the universal binary, module files, Brewfile, and updated documents.

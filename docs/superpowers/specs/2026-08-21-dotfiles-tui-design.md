# Dotfiles TUI Design

## Goal

Provide a macOS terminal UI, launched with `./setup`, that installs, reapplies,
updates, removes, and diagnoses the nine dotfile packages in this repository.
It must not manage Homebrew or depend on Homebrew-installed tools such as GNU
Stow, fzf, gum, dialog, Bun, or Node.

## Scope

The managed packages are `aerospace`, `ghostty`, `hammerspoon`, `herdr`,
`rift`, `starship`, `wezterm`, `zed`, and `zsh`. They appear in one flat,
multi-select list with `Select all` and `Clear all`; there are no categories.
The public interface is interactive only. There is no automation-oriented
command-line mode.

The tool does not install applications, formulae, casks, fonts, App Store
software, or external Git repositories. The existing `homebrew/Brewfile` is
neither displayed nor executed.

## Runtime and Entry Point

`./setup` resolves the repository root from its own location, so the checkout
may live anywhere. The implementation uses the macOS-provided Zsh plus common
POSIX/macOS utilities. Git is used only by the Update action and is diagnosed
as a missing requirement rather than installed.

The TUI uses ANSI terminal control sequences and Zsh key reading. Arrow keys
move, Space toggles a package, Enter confirms, and `q` or Escape cancels. It
restores cursor visibility and terminal state on success, error, interrupt, or
cancellation.

## Structure

- `setup`: executable entry point and repository-root discovery.
- `scripts/packages.zsh`: ordered package manifest containing display names,
  descriptions, runtime commands, and explicitly diagnosed internal
  references.
- `scripts/core.zsh`: package discovery, status calculation, operation
  planning, link changes, backups, rollback, removal, Git checks, and Doctor
  results. It has no terminal rendering responsibility and can be sourced by
  tests.
- `scripts/tui.zsh`: menus, checkbox interaction, previews, confirmations,
  conflict choices, progress, and result summaries.
- `tests/run.zsh`: dependency-free behavioral test runner using temporary HOME
  and repository fixtures.

The repository already excludes `scripts` from Stow through
`.stow-local-ignore`; `setup`, `tests`, and `docs` are outside managed package
directories and therefore are not installation inputs.

## Package Mapping and Status

Each regular file or symlink beneath a package directory maps to the same
relative path beneath `$HOME`. New installations create leaf-file symlinks,
creating missing parent directories as needed. This allows managed files to
coexist with application-created or user-owned files in the same directory.
Empty source directories are not installed.

Existing GNU Stow layouts remain valid. A destination counts as installed when
it resolves to the expected source file, including when a parent directory is
the Stow-created symlink. The TUI reports each package as:

- `not installed`: none of its files resolve to their sources and there are no
  conflicting destinations;
- `installed`: every file resolves to its expected source;
- `incomplete`: some, but not all, files resolve correctly;
- `conflict`: at least one required destination or parent is occupied by a
  regular file, directory of the wrong kind, or link to another target.

Broken and wrong-target symlinks are conflicts, not installed files.

## Actions and Flow

The first screen selects one of five actions.

### Install

Show the flat package checklist, initially selecting packages already fully
installed. After selection, calculate and display the paths that will be
created, left unchanged, or treated as conflicts. Require final confirmation.
Install creates only missing links; valid existing links remain untouched.

For every conflict, default to `Skip` and allow `Back up and continue`.
Backups go to `~/.dotfiles-backups/<YYYYMMDD-HHMMSS>/<original-relative-path>`.
The backup directory is outside the repository, so no `.gitignore` entry is
needed. Nothing is ever overwritten without first being moved to that backup.

### Reapply

Use the same selection, preview, conflict, and confirmation flow as Install.
Reapply leaves correct links unchanged and repairs missing, broken, or
wrong-target links only when the applicable conflict was approved for backup.

### Update

Before changing the repository, require Git and require a clean worktree,
including untracked files. Show the pending Git update step and request
confirmation, then run `git pull --ff-only`. A failure stops the action without
link changes. After a successful pull, reload the package manifest, show the
latest package checklist, and use the Reapply flow.

A successful Git pull is outside the link transaction. If link application
later fails, link changes are rolled back but the updated Git checkout remains.
The result screen reports those outcomes separately.

### Remove

Show the package checklist and preview managed links that will be removed.
Remove only symlinks that point into this repository and correspond to the
selected package. It never removes a regular file, user-owned directory, or
wrong-target link. Existing Stow-created directory links are recognized and
may be removed when that directory represents the selected package. Parent
directories and backups are retained. Backups are never restored
automatically; their location is displayed when applicable.

### Doctor

Doctor is read-only. It checks:

- source files declared by all nine packages;
- correct, missing, broken, partial, and conflicting destinations;
- declared runtime commands for each package;
- declared repository-internal references used by configuration files;
- availability of Git for Update.

Warnings never trigger Homebrew, installation, or automatic repair. Internal
references are explicit package metadata, not heuristic parsing, so checks are
deterministic. The known Hammerspoon reference to
`modules.input.tmux_lang` is included and therefore reported while its target
file is absent.

## Transaction and Error Handling

Install, Reapply, and Remove each apply one precomputed plan as a transaction.
The engine journals every created link and directory, moved conflict, and
removed managed link. If any filesystem operation fails, it reverses journaled
operations in reverse order and reports whether rollback completed. Git changes
are never included in this rollback.

Cancellation before confirmation makes no changes. Signals during application
trigger the same rollback path. Unexpected source changes between preview and
application abort rather than applying a stale plan. Messages distinguish
skipped conflicts, operation failure, rollback failure, Git failure, and Doctor
warnings.

## Testing

Tests source `scripts/core.zsh` and operate only on temporary repository and
HOME fixtures. They cover:

- package-to-HOME leaf mapping and status classification;
- compatibility with both leaf links and Stow-style parent directory links;
- missing, partial, broken, and conflicting destinations;
- conflict skipping and timestamped backup placement;
- transaction rollback after an injected mid-application failure;
- safe removal of managed links and refusal to remove user-owned paths;
- dirty-worktree and fast-forward-only Update guards without mutating the real
  checkout;
- Doctor findings for commands, destinations, and internal references;
- selection defaults and TUI cancellation/terminal cleanup through scripted
  input where practical.

The final verification runs the complete test script, Zsh syntax checks for all
new scripts, a TUI smoke test against a temporary HOME, and `git diff --check`.

## Acceptance Criteria

- `./setup` opens the action menu on macOS without a Homebrew-provided runtime.
- A user can independently select any of the nine settings.
- Every mutating action shows a preview and requires confirmation.
- Existing correct Stow links are recognized and preserved.
- Conflicts cannot be overwritten; approved conflicts are recoverably backed
  up outside the repository.
- Partial failures do not leave partial filesystem changes.
- Update refuses dirty or non-fast-forward repository changes and reports Git
  and link outcomes separately.
- Remove touches only selected links owned by this repository.
- Doctor is read-only and never installs or repairs anything.

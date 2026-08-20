# Dotfiles TUI Design

## Goal

Provide a polished macOS terminal UI, launched with `./setup`, that installs,
reapplies, updates, removes, and diagnoses the nine dotfile packages in this
repository. Runtime use must not require Homebrew, GNU Stow, Go, fzf, gum,
dialog, Node, or Bun.

## Distribution

The source is Go using Bubble Tea v2, Bubbles v2, and Lip Gloss v2. The
repository commits one universal Mach-O executable, `./setup`, containing both
arm64 and x86_64 slices. Go is needed only to develop, test, and rebuild this
executable. It is recorded as a
developer tool in `homebrew/Brewfile`, but the TUI never reads or runs that
Brewfile.

Builds use `CGO_ENABLED=0`, `GOOS=darwin`, and reproducibility-oriented linker
flags. `tools/dotfiles-tui/build` tests both architecture builds, combines them
with `lipo`, and atomically replaces `./setup`. The module and checksums live
with all other TUI development files below `tools/dotfiles-tui/`.

## Scope

The managed packages are `aerospace`, `ghostty`, `hammerspoon`, `herdr`,
`rift`, `starship`, `wezterm`, `zed`, and `zsh`. They appear in one flat list
with individual checkboxes plus Select all and Clear all. There are no
categories and no non-interactive user interface.

The tool never installs applications, Homebrew formulae or casks, fonts,
App Store software, or external Git repositories. `homebrew/Brewfile` is not
displayed or executed.

## Components

- `tools/dotfiles-tui/cmd/dotfiles-tui`: process startup, repository path validation, and Bubble
  Tea program launch.
- `tools/dotfiles-tui/internal/dotfiles`: catalog, source-file discovery, status inspection,
  preview plans, backup choices, transactional application, Git update guards,
  and Doctor checks. It has no terminal rendering.
- `tools/dotfiles-tui/internal/ui`: Bubble Tea state machine and Lip Gloss presentation for action
  selection, flat package selection, conflict decisions, preview,
  confirmation, progress, and results.
- `setup`: the only root-level TUI artifact and universal runtime executable.
- `tools/dotfiles-tui/build`: developer-only test and universal build command.

The repository root is derived from the executable location and must be a
directory. Package sources are inspected inside the UI so Doctor remains
available when one is missing. The checkout can live anywhere. Bubble Tea owns
terminal restoration on normal exit, cancellation, and signals.

## Package Mapping and Status

Every regular file or symlink below a package directory maps to the same path
below the user's home directory. New installations create leaf-file symlinks
and missing parent directories, allowing application-created files to coexist.
Empty source directories are ignored.

Existing GNU Stow layouts remain valid. A destination is correct when it
resolves to the expected source, including through a Stow-created parent
directory link. Each package is classified as `not installed`, `installed`,
`incomplete`, or `conflict`. Broken links, wrong-target links, and blocking
parent paths are conflicts.

## Actions

The first screen offers Install, Reapply, Update, Remove, and Doctor.

Install and Reapply show the flat package checklist. Install initially selects
only not-installed packages, while Reapply initially selects fully installed
packages. They calculate a preview before mutation. Each conflict
defaults to Skip and may be changed to Back up and continue. Approved conflicts
move to `~/.dotfiles-backups/<YYYYMMDD-HHMMSS>/<original-relative-path>` before
linking. The backup location is outside the repository and needs no gitignore
rule. Correct links remain untouched.

Update requires Git and a completely clean worktree, shows a confirmation,
then runs only `git pull --ff-only`. On success it replaces the running process
with the newly checked-out architecture-matching executable, which reloads the
catalog and opens the Reapply package selection. A successful pull is retained
if later link application fails, and the two outcomes are reported separately.

Remove previews and deletes only symlinks that resolve into the selected
package. It recognizes both leaf links and Stow-created directory links. It
never removes regular files, wrong-target links, parent directories, or
backups, and it never restores backups automatically.

Doctor is read-only. It reports missing package sources, correct/missing/
partial/conflicting destinations, declared runtime commands, and Git
availability. Doctor never installs or repairs anything.

## Transaction Safety

Install, Reapply, and Remove apply an immutable, precomputed plan. The engine
journals each created directory and link, moved conflict, and removed managed
link. Any error or cancellation reverses completed steps in reverse order.
If rollback itself fails, the exact remaining paths are reported. Source or
destination changes after preview abort stale operations.

Git pull is outside the filesystem transaction and is never automatically
reversed. Cancellation before final confirmation changes nothing.

## UI

Bubble Tea provides arrow-key navigation, Space toggles, Enter confirmation,
and `q`/Escape cancellation. Bubbles supplies reusable help components; Lip
Gloss supplies restrained status colors and layout. Every mutating action has
a preview and explicit final confirmation. Result screens distinguish success,
skipped conflicts, Git failure, operation failure, rollback success, rollback
failure, and Doctor warnings.

## Testing and Acceptance

Go tests use temporary repository and home fixtures. They cover mapping and all
four statuses, leaf and Stow links, conflict choices and backup paths,
transaction rollback with injected failures, managed-only removal, Git dirty
guards and exact `pull --ff-only` invocation, Doctor findings, and Bubble Tea
model transitions. Tests never mutate the real home or checkout.

Final verification runs `go test ./...`, `go vet ./...`, both cross-builds,
universal binary slice inspection, a PTY cancellation smoke test with temporary
HOME, and `git diff --check`.

Acceptance requires `./setup` to work without development tools on both Apple
Silicon and Intel macOS; independent selection of all nine packages; preview
and confirmation for mutations; preservation of correct Stow links; recoverable
conflict backups; transactional filesystem changes; fast-forward-only Update;
managed-only Remove; and read-only Doctor behavior.

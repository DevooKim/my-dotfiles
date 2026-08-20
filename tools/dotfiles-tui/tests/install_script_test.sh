#!/bin/sh

set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)
installer="$project_root/scripts/install.sh"
test_root=$(mktemp -d "${TMPDIR:-/tmp}/dotfiles-install-test.XXXXXX")

cleanup() {
  rm -rf "$test_root"
}
trap cleanup EXIT HUP INT TERM

fail() {
  printf '%s\n' "$1" >&2
  exit 1
}

make_setup() {
  target=$1
  mkdir -p "$target/bin" "$target/.git"
  printf '#!/bin/sh\nprintf "ran\\n" > "$DOTFILES_TEST_MARKER"\n' > "$target/bin/setup"
  chmod +x "$target/bin/setup"
}

[ -x "$installer" ] || fail "scripts/install.sh is missing or not executable"

existing="$test_root/existing"
existing_marker="$test_root/existing-ran"
make_setup "$existing"
DOTFILES_DIR="$existing" DOTFILES_TEST_MARKER="$existing_marker" sh "$installer"
[ -f "$existing_marker" ] || fail "existing checkout did not launch bin/setup"

fake_git="$test_root/git"
cat > "$fake_git" <<'EOF'
#!/bin/sh
set -eu
[ "$1" = "clone" ] || exit 64
shift
if [ "${1:-}" = "--" ]; then shift; fi
repository=$1
target=$2
mkdir -p "$target/bin" "$target/.git"
printf '%s\n' "$repository" > "$target/cloned-from"
printf '#!/bin/sh\nprintf "ran\\n" > "$DOTFILES_TEST_MARKER"\n' > "$target/bin/setup"
chmod +x "$target/bin/setup"
EOF
chmod +x "$fake_git"

cloned="$test_root/cloned"
clone_marker="$test_root/cloned-ran"
DOTFILES_DIR="$cloned" DOTFILES_GIT="$fake_git" DOTFILES_TEST_MARKER="$clone_marker" sh "$installer"
[ -f "$clone_marker" ] || fail "new checkout did not launch bin/setup"
[ "$(cat "$cloned/cloned-from")" = "https://github.com/DevooKim/my-dotfiles.git" ] || fail "wrong clone repository"

occupied="$test_root/occupied"
mkdir -p "$occupied"
if DOTFILES_DIR="$occupied" sh "$installer" >/dev/null 2>&1; then
  fail "occupied non-repository directory was accepted"
fi

printf 'install script tests passed\n'

#!/bin/sh

set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)
binary="$project_root/bin/setup"

if [ -e "$project_root/setup" ] || [ -L "$project_root/setup" ]; then
  printf 'legacy root setup artifact still exists\n' >&2
  exit 1
fi

description=$(file "$binary")
case "$description" in
  *"Mach-O universal binary"*) ;;
  *) printf 'setup is not a universal Mach-O binary: %s\n' "$description" >&2; exit 1 ;;
esac

architectures=$(lipo -archs "$binary")
case " $architectures " in
  *" arm64 "*) ;;
  *) printf 'setup is missing arm64: %s\n' "$architectures" >&2; exit 1 ;;
esac
case " $architectures " in
  *" x86_64 "*) ;;
  *) printf 'setup is missing x86_64: %s\n' "$architectures" >&2; exit 1 ;;
esac

printf 'universal binary test passed\n'

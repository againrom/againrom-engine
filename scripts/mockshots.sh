#!/usr/bin/env bash
# Writes <out dir>/<task>/main/<name>-<lang>-1920x1080.png for every named shot
# of every task, or of the one task given, through the editor's headless frame.
# One run per install: AGAINROM_MOCKSHOT_INSTALLS lists install roots separated
# by spaces (default: gameversions/en and gameversions/ru beside the repository).
# Each install gives its own language to the file names.
# Usage: scripts/mockshots.sh <out dir> [task]
set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
	echo "usage: scripts/mockshots.sh <out dir> [task]" >&2
	exit 2
fi
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "$1"
out="$(cd "$1" && pwd)"
task="${2:-}"
installs="${AGAINROM_MOCKSHOT_INSTALLS:-$root/../gameversions/en $root/../gameversions/ru}"

status=0
for install in $installs; do
	AGAINROM_MOCKSHOTS="$out" AGAINROM_ASSETS="$install" \
		go -C "$root" test -count=1 ./pkg/game -run "TestMockShots${task:+/$task}" || status=1
done

find "$out${task:+/$task}" -name '*-1920x1080.png' | sort
exit "$status"

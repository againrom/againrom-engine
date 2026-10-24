#!/usr/bin/env bash
# check-fast.sh — the small gate: gofmt, the asset scan and every ordinary
# test faster than a limit; `go test` compiles every package, one without tests
# too. A lane runs it before each commit; a landing that cuts no release runs it
# on the merge. The full set (every ordinary test, the release tests on both
# installs, the mission scenarios and the save acceptance) runs at a release.
#
# Usage:
#   scripts/check-fast.sh [timings]
#
# <timings> holds one line per test of the last full run: package, test and
# seconds, separated by tabs (default: AGAINROM_TEST_TIMINGS, else
# "$(go env GOCACHE).test-timings.txt"). A test that took AGAINROM_FAST_LIMIT
# seconds (default 0.25) or longer there is skipped by name; a test the file
# does not name runs. With no timings file every ordinary test runs.
#
# pkg/game runs in its own `go test` beside the other packages: its skip list
# alone fills most of a Windows command line. Results come from the Go test
# cache where its inputs did not change, so an unchanged package costs nothing.
set -u
cd "$(dirname "$0")/.." || exit 2

limit=${AGAINROM_FAST_LIMIT:-0.25}
timings=${1:-${AGAINROM_TEST_TIMINGS:-$(go env GOCACHE).test-timings.txt}}
started=$SECONDS
rc=0

unformatted=$(git ls-files -z '*.go' | xargs -0 gofmt -l)
if [ -n "$unformatted" ]; then
	printf '%s\n' "$unformatted"
	echo "check-fast: gofmt lists the files above"
	rc=1
fi
bash scripts/check-no-game-assets.sh || rc=1

module=$(go list -m) || exit 2
game="$module/pkg/game"
others=$(go list ./... | grep -vx "$game") || exit 2

# heavy <package|-> prints the skip pattern of one package, or with "-" of
# every package but pkg/game; empty when nothing is skipped.
heavy() {
	[ -f "$timings" ] || return 0
	awk -F '\t' -v pkg="$1" -v game="$game" -v limit="$limit" '
		(pkg == "-" ? $1 != game : $1 == pkg) && $3 + 0 >= limit + 0 { names = names sep $2; sep = "|" }
		END { if (names != "") printf "^(%s)$", names }
	' "$timings"
}
skipgame=$(heavy "$game")
skipothers=$(heavy -)
if [ -f "$timings" ]; then
	count=$(awk -F '\t' -v limit="$limit" '$3 + 0 >= limit + 0' "$timings" | wc -l)
	echo "check-fast: skipping $count tests of $limit s or longer in $timings"
else
	echo "check-fast: no timings file $timings; every ordinary test runs"
fi

go test -trimpath ${skipgame:+-skip "$skipgame"} "$game" &
gamepid=$!
# shellcheck disable=SC2086 # one argument per package
go test -trimpath ${skipothers:+-skip "$skipothers"} $others || rc=1
wait "$gamepid" || rc=1

if [ "$rc" -eq 0 ]; then
	echo "check-fast: ok in $((SECONDS - started)) s"
else
	echo "check-fast: FAIL in $((SECONDS - started)) s"
fi
exit "$rc"

#!/usr/bin/env bash
# check-fast.sh — the small gate, bounded at 60 seconds (owner): gofmt, the
# asset scan and the fast set of ordinary tests; `go test` compiles every
# package, one without tests too. A lane runs it before each commit; a landing
# that cuts no release runs it on the merge. The full set (every ordinary test,
# the release tests on both installs, the mission scenarios and the save
# acceptance) runs at a release.
#
# Usage:
#   scripts/check-fast.sh [timings]
#
# <timings> holds one line per test of the last full run: package, test and
# seconds, separated by tabs (default: AGAINROM_TEST_TIMINGS, else
# "$(go env GOCACHE).test-timings.txt"). The fast set of a package is its tests
# under AGAINROM_FAST_LIMIT seconds (default 0.25), fastest first, while their
# sum stays within AGAINROM_FAST_BUDGET seconds (default 25); pkg/game's sum has
# AGAINROM_FAST_GAME_BUDGET (default 80), as the full run records its tests at
# about five times their seconds here and pkg/ui's at one and a half. A test
# the file does not name runs; with no timings file every ordinary test runs.
# A test left out runs in the full set.
#
# pkg/game and pkg/ui, the two packages whose tests take longest, each run in
# their own `go test` beside the other packages; pkg/game's skip list alone
# fills most of a Windows command line. Results come from the Go test cache
# where their inputs did not change. Each phase prints its seconds; the last
# line names a run over AGAINROM_FAST_BOUND seconds (default 60).
set -u
cd "$(dirname "$0")/.." || exit 2

limit=${AGAINROM_FAST_LIMIT:-0.25}
budget=${AGAINROM_FAST_BUDGET:-25}
gamebudget=${AGAINROM_FAST_GAME_BUDGET:-80}
bound=${AGAINROM_FAST_BOUND:-60}
timings=${1:-${AGAINROM_TEST_TIMINGS:-$(go env GOCACHE).test-timings.txt}}
rc=0

# now prints microseconds; secs <start> prints the seconds since, one decimal.
now() {
	if [ -n "${EPOCHREALTIME:-}" ]; then
		echo "${EPOCHREALTIME//[!0-9]/}"
	else
		echo $((SECONDS * 1000000))
	fi
}
secs() {
	local d=$(($(now) - $1))
	printf '%d.%d' $((d / 1000000)) $((d / 100000 % 10))
}
started=$(now)

phase=$(now)
unformatted=$(git ls-files -z '*.go' | xargs -0 gofmt -l)
if [ -n "$unformatted" ]; then
	printf '%s\n' "$unformatted"
	echo "check-fast: gofmt lists the files above"
	rc=1
fi
echo "check-fast: gofmt $(secs "$phase") s"

phase=$(now)
bash scripts/check-no-game-assets.sh || rc=1
echo "check-fast: asset scan $(secs "$phase") s"

phase=$(now)
module=$(go list -m) || exit 2
game="$module/pkg/game"
ui="$module/pkg/ui"
others=$(go list ./... | grep -vx -e "$game" -e "$ui") || exit 2

# left holds "package<TAB>test<TAB>reason" for every test the fast set leaves
# out: "limit" at or over the per-test limit, "budget" past its package's sum.
left=
if [ -f "$timings" ]; then
	left=$(LC_ALL=C sort -t "$(printf '\t')" -k1,1 -k3,3n -k2,2 "$timings" | awk -F '\t' \
		-v limit="$limit" -v budget="$budget" -v gamebudget="$gamebudget" -v game="$game" '
		$1 != pkg { pkg = $1; sum = 0; cap = ($1 == game ? gamebudget : budget) + 0 }
		$3 + 0 >= limit + 0 { print $1 "\t" $2 "\tlimit"; next }
		sum + $3 > cap { print $1 "\t" $2 "\tbudget"; next }
		{ sum += $3 }
	')
	printf '%s\n' "$left" | awk -F '\t' -v limit="$limit" -v budget="$budget" -v gamebudget="$gamebudget" -v file="$timings" '
		$3 == "limit" { over++ }
		$3 == "budget" { cut++; per[$1]++ }
		END {
			printf "check-fast: leaving out %d tests of %s s or longer and %d past the package budget (%s s, pkg/game %s s) of %s\n", over, limit, cut, budget, gamebudget, file
			for (p in per) printf "check-fast:   %s %d past the budget\n", p, per[p] | "sort"
		}'
else
	echo "check-fast: no timings file $timings; every ordinary test runs"
fi

# skip <package|-> prints the skip pattern of one package, or with "-" of
# every package but pkg/game and pkg/ui; empty when nothing is skipped.
skip() {
	[ -n "$left" ] || return 0
	printf '%s\n' "$left" | awk -F '\t' -v pkg="$1" -v game="$game" -v ui="$ui" '
		(pkg == "-" ? $1 != game && $1 != ui : $1 == pkg) { names = names sep $2; sep = "|" }
		END { if (names != "") printf "^(%s)$", names }
	'
}
skipgame=$(skip "$game")
skipui=$(skip "$ui")
skipothers=$(skip -)

# own <name> <skip> <package...> runs one `go test` and prints its seconds.
own() {
	local name=$1 pattern=$2 start code
	shift 2
	start=$(now)
	go test -trimpath ${pattern:+-skip "$pattern"} "$@"
	code=$?
	echo "check-fast: go test $name $(secs "$start") s"
	return "$code"
}
own pkg/game "$skipgame" "$game" &
gamepid=$!
own pkg/ui "$skipui" "$ui" &
uipid=$!
# shellcheck disable=SC2086 # one argument per package
own "other packages" "$skipothers" $others || rc=1
wait "$gamepid" || rc=1
wait "$uipid" || rc=1
echo "check-fast: compile and tests $(secs "$phase") s"

total=$(secs "$started")
over=
if [ $(($(now) - started)) -gt $((bound * 1000000)) ]; then
	over=", over the $bound s bound"
fi
if [ "$rc" -eq 0 ]; then
	echo "check-fast: ok in $total s$over"
else
	echo "check-fast: FAIL in $total s$over"
fi
exit "$rc"

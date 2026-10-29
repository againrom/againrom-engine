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
# pkg/game, pkg/ui and the other packages each have their own `go test`, side
# by side with gofmt and the scan; pkg/game's skip list alone fills most of a
# Windows command line. pkg/game's test binary takes about 30 s to compile, the
# longest path, so the other two run at a lower priority, and pkg/game's
# `go test` compiles first (-run '^$') and then runs the fast set in
# AGAINROM_FAST_GAME_SHARDS (default 3) runs side by side, split by the letter
# after "Test" into about equal timed sums, with -count=1, as storing a pkg/game
# result in the test cache costs about 5 s. Other results come from the Go
# test cache where their inputs did not change. Each phase prints its seconds;
# the last line names a run over AGAINROM_FAST_BOUND seconds (default 60).
set -u
cd "$(dirname "$0")/.." || exit 2

limit=${AGAINROM_FAST_LIMIT:-0.25}
budget=${AGAINROM_FAST_BUDGET:-25}
gamebudget=${AGAINROM_FAST_GAME_BUDGET:-80}
shards=${AGAINROM_FAST_GAME_SHARDS:-3}
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

module=$(go list -m) || exit 2
game="$module/pkg/game"
ui="$module/pkg/ui"

# sel holds "package<TAB>test<TAB>verdict<TAB>seconds" for every timed test:
# "limit" at or over the per-test limit, "budget" past its package's sum,
# "fast" in the fast set. left holds the first three fields of the others.
sel=
left=
if [ -f "$timings" ]; then
	sel=$(LC_ALL=C sort -t "$(printf '\t')" -k1,1 -k3,3n -k2,2 "$timings" | awk -F '\t' \
		-v limit="$limit" -v budget="$budget" -v gamebudget="$gamebudget" -v game="$game" '
		$1 != pkg { pkg = $1; sum = 0; cap = ($1 == game ? gamebudget : budget) + 0 }
		$3 + 0 >= limit + 0 { print $1 "\t" $2 "\tlimit\t" $3; next }
		sum + $3 > cap { print $1 "\t" $2 "\tbudget\t" $3; next }
		{ sum += $3; print $1 "\t" $2 "\tfast\t" $3 }
	')
	left=$(printf '%s\n' "$sel" | awk -F '\t' '$3 != "fast" { print $1 "\t" $2 "\t" $3 }')
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

# skip <package|-> [alternative] [keep] [drop] prints the skip pattern of one
# package, or with "-" of every package but pkg/game and pkg/ui, with the
# alternative inside its one group; empty when nothing is skipped. A `|` outside
# the group would make the testing package compile each part again for every
# name. A shard lists only the names its alternative does not already skip:
# those matching keep and not drop, so no shard's pattern outgrows a Windows
# command line.
skip() {
	printf '%s\n' "$left" | awk -F '\t' -v pkg="$1" -v game="$game" -v ui="$ui" -v extra="${2:-}" \
		-v keep="${3:-}" -v drop="${4:-}" '
		NF && (pkg == "-" ? $1 != game && $1 != ui : $1 == pkg) &&
			(keep == "" || $2 ~ keep) && (drop == "" || $2 !~ drop) { names = names sep $2; sep = "|" }
		END { if (extra != "") names = names sep extra; if (names != "") printf "^(%s)$", names }
	'
}
skipui=$(skip "$ui")
skipothers=$(skip -)

# classes holds the letter ranges of every pkg/game shard but the last, which
# takes the rest: the first range whose fast-set sum reaches 1/n of the total
# closes a shard. Empty with no timings file or one shard: one run.
classes=
if [ -n "$sel" ] && [ "$shards" -gt 1 ]; then
	classes=$(printf '%s\n' "$sel" | awk -F '\t' -v game="$game" -v n="$shards" '
		$1 == game && $3 == "fast" { w[substr($2, 5, 1)] += $4; total += $4 }
		END {
			L = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"; k = 1; lo = "A"
			for (i = 1; i < 26 && k < n; i++) {
				c = substr(L, i, 1); acc += w[c]
				if (acc >= total * k / n) { print lo "-" c; k++; lo = substr(L, i + 1, 1) }
			}
		}')
fi

# own <name> <skip> <go test argument...> runs one `go test` and prints its
# seconds; with low set, under nice.
own() {
	local name=$1 pattern=$2 start code
	shift 2
	start=$(now)
	${low:+nice -n 10} go test -trimpath ${pattern:+-skip "$pattern"} "$@"
	code=$?
	echo "check-fast: go test $name $(secs "$start") s"
	return "$code"
}

# gamerun compiles pkg/game's test binary, then runs its shards side by side.
# A shard is chosen by -skip alone: -run beside -skip, two patterns, has the
# testing package compile the long skip pattern again for every test.
gamerun() {
	local class pids= pid code=0 rest=
	own "pkg/game compile" "" -run '^$' "$game" || return 1
	for class in $classes; do
		own "pkg/game Test[$class]" "$(skip "$game" "[^T].*|Test|Test[^$class].*" "^Test[$class]")" -count=1 "$game" &
		pids="$pids $!"
		rest="Test[A-${class#*-}].*"
	done
	own "pkg/game${rest:+ rest}" "$(skip "$game" "$rest" "" "${rest:+^${rest%.\*}}")" -count=1 "$game" || code=1
	for pid in $pids; do
		wait "$pid" || code=1
	done
	return "$code"
}

phase=$(now)
gamerun &
gamepid=$!
low=1 own pkg/ui "$skipui" "$ui" &
uipid=$!
others=$(go list ./... | grep -vx -e "$game" -e "$ui") || others=

gofmtphase=$(now)
unformatted=$(git ls-files -z '*.go' | xargs -0 gofmt -l)
if [ -n "$unformatted" ]; then
	printf '%s\n' "$unformatted"
	echo "check-fast: gofmt lists the files above"
	rc=1
fi
echo "check-fast: gofmt $(secs "$gofmtphase") s"

scanphase=$(now)
bash scripts/check-no-game-assets.sh || rc=1
echo "check-fast: asset scan $(secs "$scanphase") s"

if [ -n "$others" ]; then
	# shellcheck disable=SC2086 # one argument per package
	low=1 own "other packages" "$skipothers" $others || rc=1
else
	echo "check-fast: go list found no other packages"
	rc=1
fi
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

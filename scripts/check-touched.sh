#!/usr/bin/env bash
# check-touched.sh — runs the tests a change can reach. A lane runs it once on
# its final candidate, beside check-fast.sh, which runs a time-cut subset.
#
# Usage:
#   scripts/check-touched.sh <base-ref>
#
# cmd/touchedtests selects, from `git diff <base-ref>...HEAD`, every ordinary
# and release test of the module that calls a changed function, method, type,
# var, const or test helper, or names a changed testdata file. The checkout
# must be at HEAD with no change to a tracked file.
#
# The selected tests run fastest first by the timings file check-fast.sh reads
# (a test it does not name counts one second), while their summed seconds stay
# within AGAINROM_TOUCHED_BUDGET seconds (default 180); the packages run one
# after another and none starts once that many seconds have passed. A test left
# over is not a failure. Release tests run when AGAINROM_ASSETS names the Russian
# install root (gameversions/ru), and are reported as not run when it is unset. Each package has
# one `go test -count=1`, split only where its -run pattern would outgrow a
# Windows command line.
#
# The output gives the counts selected, run, passed, failed, left over budget
# and not run, then the names of the failed and the left tests. The exit status
# is non-zero when a test failed or the selection could not be made.
set -u
cd "$(dirname "$0")/.." || exit 2
. scripts/timings.sh

base=${1:-}
if [ -z "$base" ]; then
	echo "usage: scripts/check-touched.sh <base-ref>" >&2
	exit 2
fi
budget=${AGAINROM_TOUCHED_BUDGET:-180}
patternmax=${AGAINROM_TOUCHED_PATTERN_MAX:-16000}
timings=$(timings_file)
tab=$(printf '\t')
started=$SECONDS

selected=$(go run ./cmd/touchedtests -root . "$base") || {
	echo "check-touched: FAIL: no selection"
	exit 2
}
echo "check-touched: selection took $((SECONDS - started)) s"

# rows holds "seconds<TAB>package<TAB>test<TAB>kind" fastest first; the seconds
# of a test the timings file does not name are 1.
if [ -f "$timings" ]; then
	known=$(timings_sorted "$timings")
else
	known=
fi
rows=$(
	{
		printf '%s\n' "$known" | awk -F '\t' 'NF == 3 { print "T\t" $1 "\t" $2 "\t" $3 }'
		printf '%s\n' "$selected" | awk -F '\t' 'NF == 3 { print "S\t" $1 "\t" $2 "\t" $3 }'
	} | awk -F '\t' '
		$1 == "T" { sec[$2 SUBSEP $3] = $4; next }
		{ s = (($2 SUBSEP $3) in sec) ? sec[$2 SUBSEP $3] : 1; printf "%s\t%s\t%s\t%s\n", s, $2, $3, $4 }
	' | LC_ALL=C sort -t "$tab" -k1,1n -k2,2 -k3,3
)

total=$(printf '%s\n' "$rows" | awk 'NF' | wc -l)
norun=
if [ -z "${AGAINROM_ASSETS:-}" ]; then
	norun=$(printf '%s\n' "$rows" | awk -F '\t' '$4 == "release"')
	rows=$(printf '%s\n' "$rows" | awk -F '\t' '$4 != "release"')
fi
nnorun=$(printf '%s\n' "$norun" | awk 'NF' | wc -l)

# chosen holds the tests within the budget, left the others.
chosen=$(printf '%s\n' "$rows" | awk -F '\t' -v budget="$budget" 'NF { sum += $1; if (sum <= budget) print }')
left=$(printf '%s\n' "$rows" | awk -F '\t' -v budget="$budget" 'NF { sum += $1; if (sum > budget) print }')

ran=0
passed=0
failed=
# Packages by their fastest test.
pkgs=$(printf '%s\n' "$chosen" | awk -F '\t' 'NF && !($2 in seen) { seen[$2] = 1; print $2 }')
for pkg in $pkgs; do
	if [ $((SECONDS - started)) -ge "$budget" ]; then
		left=$(printf '%s\n%s\n' "$left" "$(printf '%s\n' "$chosen" | awk -F '\t' -v p="$pkg" '$2 == p')" | awk 'NF')
		continue
	fi
	names=$(printf '%s\n' "$chosen" | awk -F '\t' -v p="$pkg" '$2 == p { print $3 }')
	chunk=
	flush() {
		[ -n "$chunk" ] || return 0
		local out got name n
		out=$(go test -count=1 -v -run "^($chunk)\$" "$pkg" 2>&1)
		got=$(printf '%s\n' "$out" | sed -n 's/^--- \(PASS\|SKIP\): \([A-Za-z0-9_]*\).*/\2/p')
		for name in ${chunk//|/ }; do
			ran=$((ran + 1))
			if printf '%s\n' "$got" | grep -qx "$name"; then
				passed=$((passed + 1))
			else
				failed="$failed$pkg $name"$'\n'
			fi
		done
		if [ "$(printf '%s\n' "$got" | awk 'NF' | wc -l)" -ne "$(printf '%s' "$chunk" | awk -F '|' '{ print NF }')" ]; then
			printf '%s\n' "$out" | grep -E '^(FAIL|panic|--- FAIL|# )' | head -n 20
		fi
		chunk=
	}
	for name in $names; do
		if [ -n "$chunk" ] && [ $((${#chunk} + ${#name} + 1)) -gt "$patternmax" ]; then
			flush
		fi
		chunk=${chunk:+$chunk|}$name
	done
	flush
done

nleft=$(printf '%s\n' "$left" | awk 'NF' | wc -l)
nfailed=$(printf '%s' "$failed" | awk 'NF' | wc -l)
assets=unset
[ -n "${AGAINROM_ASSETS:-}" ] && assets=set
echo "check-touched: selected $total, run $ran, passed $passed, failed $nfailed, left over the ${budget} s budget $nleft, release not run $nnorun (AGAINROM_ASSETS ${assets})"
if [ "$nfailed" -gt 0 ]; then
	echo "check-touched: failed:"
	printf '%s' "$failed" | awk 'NF { print "  " $0 }'
fi
if [ "$nleft" -gt 0 ]; then
	echo "check-touched: left:"
	printf '%s\n' "$left" | awk -F '\t' 'NF { print "  " $2 " " $3 }'
fi
if [ "$nnorun" -gt 0 ]; then
	echo "check-touched: release tests not run:"
	printf '%s\n' "$norun" | awk -F '\t' 'NF { print "  " $2 " " $3 }'
fi
if [ "$nfailed" -gt 0 ]; then
	echo "check-touched: FAIL in $((SECONDS - started)) s"
	exit 1
fi
echo "check-touched: ok in $((SECONDS - started)) s"

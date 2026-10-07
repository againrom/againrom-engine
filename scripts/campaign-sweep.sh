#!/usr/bin/env bash
# Drive every campaign mission headless, in order, and print one row each.
#
# WHAT THIS IS. The census in pipeline/check-milestone.sh LOADS each campaign
# map for one tick and counts the script arms this build cannot run. This
# command DRIVES each of them: it starts the mission, steps the simulation, and
# reports where it got to. The two answer different questions. A map can compile
# a script perfectly and still fail the moment something steps on it, and until
# this script existed nothing in the repository would have noticed.
#
# IT IS AN INSTRUMENT AND NOT A VERDICT. It issues no orders, so the drive is
# unattended and nothing walks anywhere. `lost` is not a defect -- mission 10 is
# an escort and an unattended drive loses by design (owner, 2026-08-11) -- and
# `undecided` is not one either: a mission nobody plays is a mission nobody
# finishes. THE EXIT CODE IGNORES THE OUTCOME WORD ENTIRELY. It is non-zero only
# when a mission failed to LOAD or the drive returned an error, which are
# failures of this build rather than facts about the game.
#
# WHAT MOVES A ROW. `unsupported` falls when a story implements a script
# operation. `reached` falls with it and can also change when the simulation
# reaches a different part of a map. `hash` is the simulation's own digest and
# is meaningful only against byte-identical assets, so it compares a run with
# another run over the same root and with nothing else.
#
# DETERMINISM. pkg/sim steps on a fixed integer tick with no clock, no
# math/rand and no floats, and nothing here passes a seed or a time. Two runs
# over the same root therefore print the same table; --check runs the sweep
# twice and diffs it rather than asserting that from the doc comment.
#
# Usage, from the repository root:
#
#   bash scripts/campaign-sweep.sh                     # AGAINROM_ASSETS
#   bash scripts/campaign-sweep.sh --assets DIR
#   bash scripts/campaign-sweep.sh --ticks 20000
#   bash scripts/campaign-sweep.sh --missions "10 20 30"
#   bash scripts/campaign-sweep.sh --census            # per-operation breakdown
#   bash scripts/campaign-sweep.sh --check             # run twice, diff

set -u

root="$(cd "$(dirname "$0")/.." && pwd)"
assets="${AGAINROM_ASSETS:-}"
binary=""
census=0
check=0
status=0

# THE TICK BUDGET. Most campaign missions never decide unattended, so a budget
# is what keeps the sweep a census rather than a wait. 2000 is above the tick at
# which mission 10's escort can be lost when it is driven, so the missions that
# CAN decide unattended still show a verdict, and 28 maps at this budget is
# under a minute.
ticks=2000

# EVERY campaign map scenario.res ships, on both preserved roots, in ascending
# order: fifteen tens (10 .. 150) and thirteen side maps (31, 41 .. 151). It is
# the same list pipeline/check-milestone.sh walks. It is REPEATED here rather
# than read from there because that file is outside this repository and a lane
# worktree has no pipeline/ beside it.
missions="10 20 30 31 40 41 50 51 60 61 70 71 80 81 90 91 100 101 110 111 120 121 130 131 140 141 150 151"

while [ $# -gt 0 ]; do
	case "$1" in
	--assets) assets="${2:-}"; shift 2 ;;
	--ticks) ticks="${2:-}"; shift 2 ;;
	--missions) missions="${2:-}"; shift 2 ;;
	--binary) binary="${2:-}"; shift 2 ;;
	--census) census=1; shift ;;
	--check) check=1; shift ;;
	-h | --help) sed -n '2,32p' "$0"; exit 0 ;;
	*) echo "unknown argument $1" >&2; exit 2 ;;
	esac
done

if [ -z "$assets" ]; then
	echo "no asset root: set AGAINROM_ASSETS or pass --assets DIR" >&2
	exit 2
fi
if [ ! -d "$assets" ]; then
	echo "no asset root at $assets" >&2
	exit 2
fi

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

if [ -z "$binary" ]; then
	binary="$work/againrom.exe"
	( cd "$root" && go build -o "$binary" ./cmd/againrom ) || {
		echo "could not build ./cmd/againrom" >&2
		exit 2
	}
fi

# field pulls one value out of a headless world event. The events are JSON
# Lines with no nesting below `world`, so one sed per field is enough and this
# script needs no JSON parser on the machine running it.
field() {
	printf '%s\n' "$2" | sed -n "s/.*\"$1\":\\([0-9-]*\\).*/\\1/p" | tail -1
}

word() {
	printf '%s\n' "$2" | sed -n "s/.*\"$1\":\"\\([^\"]*\\)\".*/\\1/p" | tail -1
}

sweep() {
	printf 'mission  tick  outcome     unsupported  reached  alive  fallen  hash\n'
	local total_unsupported=0 total_reached=0 decided=0 count=0
	for m in $missions; do
		local scenario="$work/m$m.json"
		if [ "$census" -eq 1 ]; then
			printf '{"version":2,"stage":"mission","mission":%s,"steps":[{"command":"wait_ticks","ticks":%s},{"command":"report","name":"end"}]}' \
				"$m" "$ticks" >"$scenario"
		else
			printf '{"version":2,"stage":"mission","mission":%s,"steps":[{"command":"wait_ticks","ticks":%s}]}' \
				"$m" "$ticks" >"$scenario"
		fi
		local out rc
		out="$(AGAINROM_ASSETS="$assets" "$binary" --headless "$scenario" 2>"$work/err")"
		rc=$?
		local event
		event="$(printf '%s\n' "$out" | grep '"world"' | tail -1)"
		if [ $rc -ne 0 ] || [ -z "$event" ]; then
			printf '%7s  %s\n' "$m" "DID NOT DRIVE: $(tail -1 "$work/err")"
			status=1
			continue
		fi
		local tick outcome unsupported reached alive fallen hash
		tick="$(field tick "$event")"
		outcome="$(word outcome "$event")"
		unsupported="$(field unsupported "$event")"
		reached="$(field reached_unsupported "$event")"
		alive="$(field alive "$event")"
		fallen="$(field fallen "$event")"
		hash="$(field hash "$event")"
		printf '%7s  %4s  %-10s  %11s  %7s  %5s  %6s  %s\n' \
			"$m" "$tick" "$outcome" "$unsupported" "$reached" "$alive" "$fallen" "$hash"
		total_unsupported=$((total_unsupported + unsupported))
		total_reached=$((total_reached + reached))
		count=$((count + 1))
		[ "$outcome" != "undecided" ] && decided=$((decided + 1))
		if [ "$census" -eq 1 ]; then
			# The report step's census rows, one per reached operation:
			# {"kind":"instant","op":2,"times":11}. They are printed under
			# their mission so a story can name its own before and after.
			printf '%s\n' "$event" |
				grep -o '{"kind":"[^"]*","op":[0-9]*,"times":[0-9]*}' |
				sed -n 's/{"kind":"\([^"]*\)","op":\([0-9]*\),"times":\([0-9]*\)}/           \3 x \1 op \2/p'
		fi
	done
	printf -- '-------  ----  ----------  -----------  -------  -----  ------  ----\n'
	printf '%7s  %4s  %-10s  %11s  %7s\n' \
		"$count" "$ticks" "$decided decided" "$total_unsupported" "$total_reached"
}

if [ "$check" -eq 0 ]; then
	sweep
	exit $status
fi

# --check: the same sweep twice, diffed. A difference is nondeterminism, which
# is a defect in this build and not a property of the game.
first="$work/first.txt"
second="$work/second.txt"
sweep >"$first"
sweep >"$second"
cat "$first"
if diff -u "$first" "$second" >"$work/diff.txt"; then
	echo
	echo "deterministic: two sweeps over $assets printed the same table"
else
	echo
	echo "NOT DETERMINISTIC: two sweeps over $assets differ"
	cat "$work/diff.txt"
	status=1
fi
exit $status

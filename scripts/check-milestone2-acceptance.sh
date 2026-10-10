#!/usr/bin/env bash
#
# check-milestone2-acceptance.sh — run the owner-milestone-2 independent
# acceptance instruments (story 1140, pipeline/SAV-COMPLETION.md "Owner
# milestone 2") against one or more lawful install roots, and fail loudly if
# any of them go red.
#
# WHY THIS EXISTS (story1140-pass1.md F-6). The four TestMilestone2* tests
# this story added live behind the sessioncorpusaudit build tag, the tag
# every earlier corpus-audit story (1130-1139) also used. pipeline/
# check-release-tests.sh runs `go test` with no build tag, so it never builds
# or selects them; nothing else in scripts/ or pipeline/ mentions the tag
# either. The acceptance declaration these four tests carry was silently red
# on 2027-09-07 -- the preserved save corpus grew five files between the
# story's own measurement and its push, and one of them then hit a
# pre-existing importer refusal -- and no gate the seat runs could have said
# so before an adversarial pass measured it by hand. "An acceptance
# instrument nothing runs is not an instrument" (story1140-pass1.md). That
# specific refusal is gone: seat hotfix f1053f5 (ledger row, docs/HOTFIXES.md)
# narrowed importSavedItemObjects's ambiguity guard.
#
# This does not mean the "excluded (ResumeOriginalSave refused)" mechanism
# below has nothing to name. The corpus is discovered, not pinned (F-1), and
# it can carry refusals from any importer family, reported the same way, by
# file and reason, in each instrument's own -v log. That is by design: a
# refusal is named here whichever importer produces it, and this script's own
# output is the record of which files, if any, are currently excluded -- not
# this comment. It DID carry one such family: a dead-actor identity bind and
# a late-dead simulation stage, both guards this project wrote narrower than
# published law (SAV-DEADLOAD-125, SAV-DEADLOAD-128). Story 1143 widened both;
# the corpus this comment was last measured against carries zero refusals on
# either lawful root.
#
# This script is what "a gate the seat actually runs" names for that family.
# It does not replace check-release-tests.sh, which still owns the untagged,
# registered gated-test population; it is the sessioncorpusaudit family's own
# equivalent, selecting TestMilestone2*. Story 1144 adds an independent
# dying-owner-actor witness to the four instruments from story 1140; story
# 1145 adds Buildings; 1151 adds Sacks and their item graphs. Other opt-in
# corpus audits are not selected by this gate.
#
# STORY 1195 adds the permanent SAV round-trip regression instrument
# (pipeline/SAV-ENDGAME.md "Permanent instrument", pkg/game/
# savroundtrip1195_corpus_test.go) to the same selection, on the same
# reasoning story1140-pass1.md F-6 already established: "an acceptance
# instrument nothing runs is not an instrument"
# (pipeline/reviews/story1195-review.md F2). Its original-corpus half reads the
# corpus this script already exports. The retired AGS half is replaced by the
# converted-corpus continuation (pkg/game/savconvertedcorpus_corpus_test.go):
# each SAV the one-time AGS conversion wrote is loaded cold, saved again through
# the ordinary producer, loaded cold a second time and compared with no loss
# admitted, beside the conversion manifest's input identities
# (AGAINROM_CONVERTED_CORPUS, one directory per root).
#
# STORY 1197 adds the original-SAV byte-identity census (pkg/game/
# savbyteidentity1197_corpus_test.go) to the same selection, on the same
# reasoning. It is the only instrument here whose two sides do not both pass
# through this project's reader: it decodes each discovered original save to
# the reader's own intermediate, re-emits a whole container from that
# intermediate with no intended change, and compares the produced bytes with
# the file they came from. Its SAV-BYTEID-1197- lines are whitelisted below so
# a passing run still prints the census -- including the two numbers owner
# direction put first, the campaign document records and the campaign
# position. It reads no corpus this script does not already export.
#
# The writer census over changed worlds (pkg/game/
# savwritercensus_corpus_test.go) joins the same selection: each original is
# loaded, changed, written through the SAV dialog route and checked field by
# field. Its SAV-WRITERCENSUS- lines are whitelisted below.
#
# The seat AGENTS.md requires this script for original-resume changes and
# acceptance-instrument changes, on both lawful asset roots.
#
# Usage:
#   scripts/check-milestone2-acceptance.sh <asset-root> [<asset-root> ...]
#
# AGAINROM_SAVE_CORPUS overrides the original `.sav` corpus directory
# (default: gameversions/saves next to this checkout, matching
# check-release-tests.sh's own default). AGAINROM_CONVERTED_CORPUS names one
# converted-corpus directory used for every root; without it each root reads
# `out-<root directory name>` (out-en, out-ru) under
# AGAINROM_CONVERTED_CORPUS_BASE, whose default is
# review/story1257-ags-migration beside the engine checkout. The resolved
# paths are printed before the first root runs, so a lane never has to guess
# which corpus a green run measured. The corpora this script reads are DISCOVERED, not
# pinned to a fixed file count (story1140-pass1.md F-1) -- this script does
# not assert a population size, only that every file each corpus currently
# contains, minus any it names as a refused exclusion, agrees with this
# build's own live state.

set -u

here="$(cd "$(dirname "$0")/.." && pwd)"
corpus="${AGAINROM_SAVE_CORPUS:-$here/../gameversions/saves}"

fail() { echo "check-milestone2-acceptance: $*" >&2; exit 2; }

convbase="${AGAINROM_CONVERTED_CORPUS_BASE:-$here/../review/story1257-ags-migration}"
# convdir names the converted-corpus directory one root reads.
convdir() {
	if [ -n "${AGAINROM_CONVERTED_CORPUS:-}" ]; then
		printf '%s' "$AGAINROM_CONVERTED_CORPUS"
	else
		printf '%s/out-%s' "$convbase" "$(basename "$1")"
	fi
}

[ -d "$corpus" ] || fail "no save corpus at $corpus; set AGAINROM_SAVE_CORPUS"
[ "$#" -gt 0 ] || fail "usage: $0 <asset-root> [<asset-root> ...]"

echo "check-milestone2-acceptance: SAV corpus $corpus"

cache="${AGAINROM_GOCACHE:-${TMPDIR:-/tmp}/againrom-go-cache}"
mkdir -p "$cache" || fail "cannot create Go cache at $cache"
GOCACHE="$(cd "$cache" && { pwd -W 2>/dev/null || pwd; })" || exit 2
export GOCACHE

cd "$here" || fail "cannot enter $here"

# Every instrument walks the whole corpus on its own and they share no state,
# so each (root, instrument) pair runs as its own process from one prebuilt
# test binary per package. AGAINROM_M2_JOBS processes run at once (default:
# cores minus four, at most 12), longest first by the previous run's times,
# kept next to the Go cache. A process that failed to launch (0xc0000142) is
# rerun alone. Each root prints its instruments in name order.
# AGAINROM_M2_SAVGATE selects the SAV round-trip gate's share of the run:
# unset runs every instrument, "skip" leaves the gate out and "only" runs the
# gate alone, so a chain can run it as its own leg beside the rest
# (scripts/check-sav-roundtrip-gate.sh).
case "${AGAINROM_M2_SAVGATE:-}" in
"") pattern='^(TestMilestone2|TestSAVRoundTrip1195|TestSAVConvertedCorpus|TestSAVByteIdentity1197|TestSAVWriterCensus|TestSAVRoundTripGate)' ;;
skip) pattern='^(TestMilestone2|TestSAVRoundTrip1195|TestSAVConvertedCorpus|TestSAVByteIdentity1197|TestSAVWriterCensus)' ;;
only) pattern='^TestSAVRoundTripGate' ;;
*) fail "AGAINROM_M2_SAVGATE must be unset, skip or only" ;;
esac
listing="$(env -u GOMEMLIMIT go test -tags sessioncorpusaudit -list "$pattern" ./pkg/game/... 2>&1)" || {
	printf '%s\n' "$listing"
	fail "cannot list the acceptance instruments"
}
# AGAINROM_M2_WORK keeps each instrument's output file and exit receipt in a
# named directory, readable while the workers run and after a deadline kills
# the run.
if [ -n "${AGAINROM_M2_WORK:-}" ]; then
	work="$AGAINROM_M2_WORK"
	mkdir -p "$work" || fail "cannot create $work"
else
	work="$(mktemp -d)" || fail "cannot create a work directory"
	trap 'rm -rf "$work"' EXIT
fi
module="$(env -u GOMEMLIMIT go list -m)" || fail "cannot read the module path"
packages=() dirs=() tests=() owners=() pending=()
while IFS= read -r line; do
	case "$line" in
	Test*) pending+=("$line") ;;
	ok*)
		[ "${#pending[@]}" -gt 0 ] || continue
		read -r _ pkg _ <<<"$line"
		p="${#packages[@]}"
		packages+=("$pkg")
		dirs+=("$here/${pkg#"$module"/}")
		env -u GOMEMLIMIT go test -c -tags sessioncorpusaudit -o "$work/p$p.test.exe" "$pkg" || fail "cannot build $pkg"
		for name in "${pending[@]}"; do
			tests+=("$name")
			owners+=("$p")
		done
		pending=()
		;;
	esac
done <<<"$listing"
[ "${#tests[@]}" -gt 0 ] || fail "no acceptance instrument matches $pattern"

timings="$cache.m2-timings${AGAINROM_M2_SAVGATE:+-$AGAINROM_M2_SAVGATE}"
declare -A seconds=()
if [ -f "$timings" ]; then
	while read -r name took; do
		seconds[$name]="$took"
	done <"$timings"
fi
jobs="${AGAINROM_M2_JOBS:-}"
if [ -z "$jobs" ]; then
	cores="$(nproc 2>/dev/null || echo 4)"
	jobs=$((cores > 5 ? cores - 4 : 1))
	[ "$jobs" -le 12 ] || jobs=12
fi
roots=("$@")
for root in "${roots[@]}"; do
	echo "check-milestone2-acceptance: converted corpus $(convdir "$root")"
done

# The SAV round-trip gate (pkg/game/savroundtripgate_corpus_test.go,
# docs/1228/story.md) reads the owner kits named by
# AGAINROM_SAV_ROUNDTRIP_KITS: a ';'-separated list of files or glob patterns.
# Its default names the seat's own review kits; a listed kit that matches
# nothing is reported ABSENT and fails the gate, never a pass.
winpath() { if command -v cygpath >/dev/null 2>&1; then cygpath -m "$1"; else printf '%s' "$1"; fi; }
seat="$(winpath "$(cd "$here/.." && pwd)")"
kits="${AGAINROM_SAV_ROUNDTRIP_KITS:-$seat/review/owner-sav-story1226/game9255.sav;$seat/review/owner-sav-story1225/*.sav;$seat/review/owner-saver-terminal86/*.sav}"
echo "check-milestone2-acceptance: SAV round-trip kits $kits"

# A long instrument runs as part tests; the part that completes a set checks
# the whole from the summaries the parts leave in AGAINROM_M2_SHARE, a fresh
# directory per root and run.
shares=()
for r in "${!roots[@]}"; do
	shares[$r]="$(mktemp -d "$work/share-$r.XXXXXX")" || fail "cannot create a part share directory"
done

run() {
	local r="$1" t="$2" started=$SECONDS
	echo "check-milestone2-acceptance: start $(basename "${roots[$r]}") ${tests[$t]} at ${SECONDS}s" >&2
	# AGAINROM_CENSUS_ONLY and AGAINROM_SAV_ROUNDTRIP_ONLY narrow a census
	# and AGAINROM_SAV_ROUNDTRIP_WORKERS sizes one, so a caller's value never
	# reaches this gate.
	(cd "${dirs[${owners[$t]}]}" &&
		env -u AGAINROM_CENSUS_ONLY -u AGAINROM_SAV_ROUNDTRIP_ONLY -u AGAINROM_SAV_ROUNDTRIP_WORKERS \
			AGAINROM_ASSETS="${roots[$r]}" AGAINROM_M2_SHARE="$(winpath "${shares[$r]}")" AGAINROM_SAVE_CORPUS="$corpus" AGAINROM_CONVERTED_CORPUS="$(convdir "${roots[$r]}")" AGAINROM_SAV_ROUNDTRIP_KITS="$kits" \
			"$work/p${owners[$t]}.test.exe" -test.count=1 -test.v -test.timeout=30m -test.run "^${tests[$t]}\$") \
		>"$work/$r-$t.out" 2>&1
	local rc=$?
	echo "$rc $((SECONDS - started))" >"$work/$r-$t.rc"
	echo "check-milestone2-acceptance: end $(basename "${roots[$r]}") ${tests[$t]} exit $rc after $((SECONDS - started))s at ${SECONDS}s" >&2
}

# An instrument with no recorded time sorts first.
order=()
while read -r _ r t; do
	order+=("$r $t")
done < <(for r in "${!roots[@]}"; do
	[ -d "${roots[$r]}" ] || continue
	for t in "${!tests[@]}"; do
		echo "${seconds[${tests[$t]}]:-999999} $r $t"
	done
done | sort -k1,1nr -k3,3n -k2,2n)
running=0
for job in "${order[@]}"; do
	if [ "$running" -ge "$jobs" ]; then
		wait -n
		running=$((running - 1))
	fi
	run $job &
	running=$((running + 1))
done
wait
for job in "${order[@]}"; do
	read -r r t <<<"$job"
	read -r rc _ <"$work/$r-$t.rc"
	if [ "$rc" -ne 0 ] && grep -q '0xc0000142' "$work/$r-$t.out"; then
		run "$r" "$t"
	fi
done

# An instrument that did not exit 0 keeps its earlier time: a run that failed
# early would otherwise record zero seconds and put the longest instruments
# last in the next run's queue.
declare -A took=()
for job in "${order[@]}"; do
	read -r r t <<<"$job"
	read -r code elapsed <"$work/$r-$t.rc"
	if [ "$code" -ne 0 ]; then
		[ -z "${seconds[${tests[$t]}]:-}" ] || took[${tests[$t]}]="${seconds[${tests[$t]}]}"
		continue
	fi
	[ "${took[${tests[$t]}]:-0}" -ge "$elapsed" ] || took[${tests[$t]}]="$elapsed"
done
for name in "${!took[@]}"; do
	echo "$name ${took[$name]}"
done | sort >"$timings.tmp" && mv -f "$timings.tmp" "$timings"

status=0
for r in "${!roots[@]}"; do
	root="${roots[$r]}"
	[ -d "$root" ] || { echo "check-milestone2-acceptance: no install at $root" >&2; status=1; continue; }
	echo "check-milestone2-acceptance: root $root"
	rc=0 slowest=0
	out=""
	for t in "${!tests[@]}"; do
		read -r code elapsed <"$work/$r-$t.rc"
		[ "$code" -eq 0 ] || rc="$code"
		[ "$slowest" -ge "$elapsed" ] || slowest="$elapsed"
		out+="$(cat "$work/$r-$t.out")"$'\n'
	done
	out+="ok  ${packages[*]} ${#tests[@]} instrument(s), slowest ${slowest}s"
	if [ "$rc" -ne 0 ]; then
		# Keep every file/field mismatch and source-walk refusal visible.
		printf '%s\n' "$out"
		echo "check-milestone2-acceptance: root $root FAILED (exit $rc)" >&2
		status=1
	else
		# A passing test's own t.Logf output is otherwise invisible outside
		# -v: this filter is the whitelist of summary lines this script
		# prints even on success, not a general log suppressor. Story1195
		# adds its own -CENSUS/-REASON/-NOTE lines here on the same terms
		# (pipeline/reviews/story1195-review.md F2: "a product that go test
		# suppresses is not a product").
		printf '%s\n' "$out" | grep -E '(^[[:space:]]+milestone2_acceptance_[a-z]+_test\.go:[0-9]+: ((map reopen|terrain block plane|cell records|session block|dying owner actors|buildings|sacks|spell effects|trailer|players|groups|unit scalars|projectiles|unit combat|mover routes|actor roots|dead roots|audited)|.*unreadable, skipped:)|^[[:space:]]+savroundtrip1195_corpus_test\.go:[0-9]+: (SAV-ROUNDTRIP-|REFUSED |[^:]+: REFUSED )|^[[:space:]]+savconvertedcorpus_corpus_test\.go:[0-9]+: (SAV-ROUNDTRIP-|REFUSED |[^:]+: REFUSED )|^[[:space:]]+savbyteidentity1197_corpus_test\.go:[0-9]+: (SAV-BYTEID-|audited |.*unreadable, skipped:)|^[[:space:]]+savwritercensus_corpus_test\.go:[0-9]+: (SAV-WRITERCENSUS-|.*unreadable, skipped:)|^[[:space:]]+savroundtripgate_corpus_test\.go:[0-9]+: SAV-ROUNDTRIP-GATE (CENSUS|KEY|REFUSED|MISMATCHED|UNREADABLE|ABSENT|FIXED|ADDED|RUNTIME)|^ok )'
	fi
done

exit $status

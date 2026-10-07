#!/usr/bin/env bash
#
# check-claim-citations.sh — every claim id this repository cites resolves in
# the pinned knowledge snapshot.
#
# THE FAILURE THIS WAS WRITTEN FOR (2026-08-19). While drafting story 1017's
# round-2 spec.md, a lane wrote a specific technical fact -- that the original
# draws a button label through an undecoded vtable slot -- and attributed it to
# claim id TOWN-183. That id does not exist in the story's own pinned ledgers;
# research had published it on a branch the pin does not carry. The lane caught
# it itself before committing, and nothing else would have: the id is
# well-formed, the sentence is plausible, go build and go vet do not read
# comments, and no reader memorises which of 1300 ids the current pin holds.
#
# check-pin-forward.sh already names this failure from the other direction: a
# shipped story citing a claim its own pinned research did not contain. There
# the pin moved backward under a correct citation; here the citation was written
# ahead of the pin. Both end with a reader following an id to a row that is not
# there.
#
# THE ENGINE HAS NO EXPERIMENTS DIRECTORY (2026-09-05). This script used to run
# against the research submodule, which carries both claims/ and experiments/:
# a claim id resolved to a ledger row, an EXP id to a directory. The engine's
# knowledge submodule is research's public snapshot and by design publishes
# claims/, formats/ and tools/claim/ only -- experiments/, with every evidence
# file, is excluded (pipeline/KNOWLEDGE-PUBLISH.md). An EXP-NNNN citation is
# therefore never resolvable here, on purpose, not because a pin lags: this
# repository's docs and comments cite the experiment that produced a claim as
# provenance, the same way a paper cites a dataset it does not ship. Counting
# these and printing the count is information; failing the gate on them would
# be failing on a structural fact instead of a defect.
#
# A CLAIM id is a different matter -- claims/ IS published, so a claim id that
# does not resolve is either wrong or ahead of the snapshot, exactly the
# failure this script was written for, and it still fails on one. Fix a real
# miss by correcting or dropping the citation at its source; do not delete a
# citation just to silence this gate when the hit names a claim family the
# snapshot does not carry yet (e.g. R2-ASSET-* before its own audit) -- report
# it instead, the same as any other unresolved claim id.
#
# WHAT IT CHECKS. Every token in this repository shaped like a claim id, whose
# prefix is a prefix knowledge/claims/ actually uses, must resolve to a ledger
# row there. Prefixes are read from the pinned tree at every run, so a new
# ledger is covered with no edit here. A token shaped EXP-NNNN is recognised
# and counted separately; it is never required to resolve.
#
# The prefix condition is what keeps DIV-155, FR-2 and SC-2 out of the claim
# selection without a hand-maintained exclusion list: DIV is not a claim
# prefix, so DIV-155 is never a candidate.
#
# WHAT IT DOES NOT CHECK. Whether the cited row says what the citing sentence
# claims it says. No script can hold that. It reads no .gitignore'd tree, so
# SDD/ and builds/ are outside it.
#
# It prints the number of citations it SELECTED before its verdict. A selection
# of zero is a broken selector, not a clean tree, and it exits non-zero.

set -u

cd "$(dirname "$0")/.." || exit 2

if [ ! -d knowledge/claims ]; then
	echo "check-claim-citations: FAIL - the knowledge submodule is not checked out." >&2
	echo "  Run: git -c protocol.file.allow=always submodule update --init knowledge" >&2
	exit 2
fi

# Claim ids: ledger rows keyed on the id in the first column. Two of the
# ledgers (registry.md, retracted.md) are keyed on an experiment id instead;
# a leading EXP- is not a claim prefix, so those rows are dropped by shape
# rather than by subtracting a directory listing the engine does not have.
claim_ids=$(grep -hoE '^\| *[A-Z][A-Z0-9]*(-[A-Z0-9]+)*-[0-9]{3,4}\b' knowledge/claims/*.md |
	sed -E 's/^\| *//' | grep -v '^EXP-' | LC_ALL=C sort -u)

if [ -z "$claim_ids" ]; then
	echo "check-claim-citations: FAIL - no claim ids found in the pinned snapshot." >&2
	echo "  The ledger row format changed, or knowledge/claims/ is empty." >&2
	exit 2
fi

# EXP is always a recognised prefix, even with no experiments/ directory to
# read it from: the shape EXP-NNNN is unambiguous and no claim family is ever
# named EXP.
prefixes=$(printf '%s\nEXP\n' "$claim_ids" | sed -E 's/-[0-9]{3,4}$//' | LC_ALL=C sort -u)
claim_count=$(printf '%s\n' "$claim_ids" | grep -c .)
prefix_count=$(printf '%s\n' "$prefixes" | grep -c .)

# Tracked and untracked-but-not-ignored Go and Markdown, minus the submodule
# itself: a ledger citing its own ids is not this repository citing them.
cited=$(git grep --untracked -hoE '\b[A-Z][A-Z0-9]*(-[A-Z0-9]+)*-[0-9]{3,4}\b' \
	-- '*.go' '*.md' ':(exclude)knowledge' | LC_ALL=C sort -u)

candidates=$(printf '%s\n' "$cited" | awk -v pfx="$prefixes" '
	BEGIN { n = split(pfx, a, "\n"); for (i = 1; i <= n; i++) if (a[i] != "") P[a[i]] = 1 }
	{ t = $0; sub(/-[0-9]{3,4}$/, "", t); if (t in P) print }')

selected=$(printf '%s\n' "$candidates" | grep -c .)
if [ "$selected" -eq 0 ]; then
	echo "check-claim-citations: FAIL - selected 0 citations." >&2
	echo "  A selection of zero is a broken selector, not a clean tree." >&2
	echo "  The snapshot carries $claim_count claims under $prefix_count prefixes." >&2
	exit 1
fi

# EXP-shaped candidates are counted, never verified -- see above.
exp_candidates=$(printf '%s\n' "$candidates" | grep -E '^EXP-[0-9]{3,4}$' | LC_ALL=C sort -u)
exp_count=$(printf '%s\n' "$exp_candidates" | grep -c .)
claim_candidates=$(printf '%s\n' "$candidates" | grep -vE '^EXP-[0-9]{3,4}$' | LC_ALL=C sort -u)

unknown=$(LC_ALL=C comm -23 <(printf '%s\n' "$claim_candidates") <(printf '%s\n' "$claim_ids"))

if [ -n "$unknown" ]; then
	unresolved_count=$(printf '%s\n' "$unknown" | grep -c .)
	echo "check-claim-citations: FAIL - $unresolved_count of $selected distinct citation(s) selected do not" >&2
	echo "  resolve against knowledge/claims/:" >&2
	while read -r token; do
		[ -n "$token" ] || continue
		echo "  $token" >&2
		git grep --untracked -nH -- "$token" '*.go' '*.md' ':(exclude)knowledge' |
			sed 's/^/      /' >&2
	done <<<"$unknown"
	pin="$(git -C knowledge describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD:knowledge 2>/dev/null || echo unknown)"
	echo "  The snapshot is $pin." >&2
	echo "  Either the id is wrong, or its claim family is not published in this" >&2
	echo "  snapshot yet (e.g. it lives in a ledger the snapshot excludes, such as" >&2
	echo "  rom2-asset.md pending its own audit). Report it; do not delete the" >&2
	echo "  citation just to silence this gate." >&2
	exit 1
fi

echo "check-claim-citations: ok ($claim_count claim(s) resolve against knowledge/claims/" \
	"under $prefix_count prefix(es); $exp_count experiment-id citation(s) counted," \
	"not verified -- knowledge carries no experiments/)"

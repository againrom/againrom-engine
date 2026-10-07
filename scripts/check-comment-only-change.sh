#!/bin/sh
# check-comment-only-change.sh — prove that everything this working tree changed
# since a base ref is confined to comments.
#
# Usage:
#   scripts/check-comment-only-change.sh <base-ref> [--allow-added] [--allow-moved <paths>]
#
# It extracts the base ref with `git archive` into a temporary directory, hashes
# both trees with internal/tokenidentity, and fails if any .go file's token hash
# or directive hash moved. The token hash is the go/scanner stream with every
# COMMENT dropped; the directive hash is the //go: and +build lines the token
# half cannot see. Together they say: the code the compiler reads is byte-for-
# byte the same set of tokens, and no build constraint or embed was lost.
#
# --allow-added permits .go files that exist only in the working tree, for a
# change that adds a file on purpose (a new instrument, say). --allow-moved
# takes a comma-separated list of paths whose tokens are expected to move and
# prints each one's two hashes; a comment sweep must rewrite
# internal/storyguard/baseline.go's own counters in the same commit, which is
# the case it exists for. A removed file, or a mover nobody named, always
# fails.
#
# This is the gate a comment-only change runs in place of the release, scenario
# and milestone chains: passing it is what entitles the change to skip them,
# because a tree with the same tokens reaches no screen, no shipped datum and no
# save byte differently than its base did.
set -eu

usage() {
	echo "usage: $0 <base-ref> [--allow-added] [--allow-moved <paths>]" >&2
	exit 2
}

base=${1:-}
[ -n "$base" ] || usage
shift

allow_added=
allow_moved=
while [ $# -gt 0 ]; do
	case "$1" in
		--allow-added) allow_added=-allow-added ;;
		--allow-moved)
			shift
			[ $# -gt 0 ] || usage
			allow_moved="-allow-moved=$1"
			;;
		*) usage ;;
	esac
	shift
done

root=$(git rev-parse --show-toplevel)
cd "$root"

if ! base_sha=$(git rev-parse --verify "$base^{commit}" 2>/dev/null); then
	echo "check-comment-only-change: no such commit: $base" >&2
	exit 2
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT INT TERM

# Both sides are TRACKED .go files only. git archive already excludes submodule
# contents; taking the head side from git ls-files keeps the two symmetric and
# keeps a local scratch .go file or a checked-out knowledge/ snapshot out of the
# comparison.
mkdir "$work/base" "$work/head"
git archive "$base_sha" -- '*.go' | tar -x -C "$work/base"
git ls-files -z -- '*.go' | tar --null -T - -cf - | tar -x -C "$work/head"

# The instrument is built from the WORKING tree and run against both, so the
# base tree needs no copy of it and a base predating this script is comparable.
go build -o "$work/tokenidentity" ./internal/tokenidentity/cmd/tokenidentity

echo "check-comment-only-change: base $base ($base_sha) against the working tree"
"$work/tokenidentity" -compare $allow_added $allow_moved "$work/base" "$work/head"

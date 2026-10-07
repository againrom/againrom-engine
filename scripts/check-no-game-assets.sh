#!/bin/sh
# check-no-game-assets.sh — fail if any game asset is tracked in this repository
# (spec FR-4 / invariant P-1).
#
# Usage:
#   scripts/check-no-game-assets.sh            scan the tracked working tree
#   scripts/check-no-game-assets.sh --history  scan the full git object history
#
# The guard keys on GIT-TRACKED content, never a raw filesystem walk: a local
# game install, extracted sample bytes, or an untracked submodule checkout (e.g.
# knowledge/) that a contributor has on disk are the developer's business and must
# not trip the guard — only tracked content can. Detection is by path
# (case-insensitive, end-anchored game-asset extension), so a document that merely
# mentions these extensions is never flagged. Third-party source has no matchable
# signature; its exclusion is a provenance-review policy (docs/PROVENANCE.md), not
# something this guard claims to detect.
set -eu

mode=tree
case "${1:-}" in
	--history) mode=history ;;
	"") : ;;
	*) echo "usage: $0 [--history]" >&2; exit 2 ;;
esac

if [ "$mode" = history ]; then
	# `git rev-list --all --objects` emits "<sha> <path>" for blobs/trees and a
	# bare "<sha>" for commit/tag objects; -s drops the bare-sha lines and keeps
	# the path (including paths that contain spaces).
	paths=$(git rev-list --all --objects | cut -s -d' ' -f2-)
else
	paths=$(git ls-files)
fi

ext_re='\.(res|reg|alm|16a|256|spr|pal|fnt|fon|wav|snd|ogg|mp3|smk|bik|mpg|avi|exe|dll)$'

offenders=$(printf '%s\n' "$paths" | grep -iE "$ext_re" | grep -v '^$' | sort -u || true)

if [ -n "$offenders" ]; then
	echo "check-no-game-assets: forbidden game-asset paths are tracked ($mode scan):" >&2
	printf '%s\n' "$offenders" | while IFS= read -r p; do
		echo "  $p" >&2
	done
	exit 1
fi

echo "check-no-game-assets: clean ($mode scan)"
exit 0

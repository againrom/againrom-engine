#!/bin/sh
# build-tools.sh — build the engine commands with the revision of this
# checkout stamped in, correct inside a linked worktree.
#
# Usage: scripts/build-tools.sh <output-dir> [package...]   (default ./cmd/...)
#
# The Go toolchain finds a repository by a ".git" directory. A linked worktree
# has a ".git" file, so a plain `go build` there stamps the enclosing
# repository's revision. This script passes the checkout's own HEAD, with
# "+dirty" for uncommitted changes, and turns the toolchain's stamp off.
set -eu
out=${1:?output directory}
shift
[ $# -gt 0 ] || set -- ./cmd/...
root=$(git rev-parse --show-toplevel)
cd "$root"
rev=$(git rev-parse HEAD)
if [ -n "$(git status --porcelain --untracked-files=no)" ]; then rev="$rev+dirty"; fi
mkdir -p "$out"
go build -trimpath -buildvcs=false \
	-ldflags "-X againrom/internal/buildinfo.stamp=$rev" -o "$out/" "$@"

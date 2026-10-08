# Windows release

An immutable `vMAJOR.MINOR.PATCH` tag publishes Windows AMD64, macOS Intel and
macOS Apple Silicon ZIPs and SHA-256 checksums on GitHub Releases. The tag must
match `cmd/againrom/VERSION` and name a
commit on `origin/main`. Create it only after the source commit has passed the
release gates. Never move a published tag. The starter keeps its own VERSION.

The Windows release workflow can also be dispatched from `main` with an existing
version tag. It builds that tagged commit, not the current branch. Both EXEs carry
the tagged source revision and their tracked Windows version resources.

The package contains the game, starter, README, license texts, third-party notices
and `BUILD-INFO.json`. No game data, saves, mods or local settings are copied.
Missing settings files use first-run defaults. The manifest records both program
versions, the complete source revision, tag, target and Go toolchain.

`scripts/package-windows.ps1` takes explicit binary and output directories, a
full revision and a version tag. The output must be new, its parent must exist,
and it cannot be inside a game install or traverse a reparse point. Existing
output is preserved. ZIP entries have fixed timestamps so unchanged inputs
produce unchanged bytes.

The build job has read permission. The publication job alone has write permission.
The macOS builds run through the reusable macOS workflow with CGO enabled on
`macos-15-intel` and `macos-15`. Both Mac packages include the game and starter.
README describes terminal launch and Gatekeeper quarantine removal for these
unsigned, non-notarized executables.

Publication waits for all three builds, creates a draft, uploads the six verified
assets, then publishes it.
A retry may complete a matching draft or confirm a matching published release.
Conflicting identity, notes, unknown assets or different bytes stop publication
without overwriting the release. Failed uploads leave a draft for inspection.

The hosted runner checks version resources, executable stamps, the package
allowlist, private-file exclusion, extraction, checksums and invalid destinations.
It has no game install; these checks do not replace the installed release gates.

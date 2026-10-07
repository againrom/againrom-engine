# Provenance

`againrom` is an independent reimplementation. Its behaviour was established by
reverse engineering the author's lawfully owned original game for
interoperability, in a separate private research process. It must be
reproducible from our own specifications plus a user's own lawful game
installation, and it must never contain game data, original code or third-party
source. This document states the distribution boundary, the evidence hierarchy
behind our format work, and the implementation procedure.

## Distribution boundary

The tracked repository and every release contain **no game data and no
third-party source**. Concretely, none of the following is ever committed or
shipped:

- game archives, maps, registries, sprites, palettes, fonts, audio, video, or
  executables (in original or converted form);
- original program code, in binary form or as a disassembly or decompilation
  listing, and any code copied or adapted from the original program;
- source copied or adapted from any third party.

Code and documents cite the research findings by claim ID and, where a finding
is anchored to the original program, by function address. A citation names a
location and states the behaviour found there in the project's own words; it
does not reproduce the original's instructions. Some closed story records
written earlier quote short instruction fragments inside such citations; they
are historical records and new work does not add to them.

Two mechanisms hold this boundary, and it is important to be honest about which
does what:

- **Game assets — enforced mechanically.** `scripts/check-no-game-assets.sh`
  fails if any game-asset path is tracked, in the working tree or (with
  `--history`) anywhere in git history. It keys on git-tracked content, so a
  contributor's local game install or extracted bytes on disk are irrelevant —
  only tracked content can trip it. This mechanically enforces invariant P-1
  (the tracked repo contains no game data at any commit).
- **Third-party source — enforced by review.** Source has no distinctive
  extension, so no guard can detect it without either matching nothing or
  over-matching. The exclusion of third-party source is therefore a policy
  upheld in code review, not a claimed automated mechanism.

`.gitignore` excludes the game install and extracted asset bytes so they are not
staged by accident; the guard is the backstop for anything force-added past it.

## Evidence hierarchy (game-first)

Facts about the game's file formats and behaviour are established from evidence,
in this order of authority:

1. **The user's own lawful game files**, observed with our developer tools
   (entry counts, sizes, offsets, checksums). This is the primary authority.
2. **The running game's observable behaviour.**
3. **Our own prior analysis**, recorded confidence-tagged in the relevant work
   item.

Every format claim is confidence-tagged and traceable to such evidence, never to
third-party source. The analysis of the original program, which includes its
disassembly, is carried out in the private research process; the supporting analysis and byte-level observations are
maintained separately from this engine repository (see below), so this repo holds
clean specifications and code only.

## Research data (public knowledge snapshot)

Reverse-engineering notes, findings and byte-level evidence are maintained in a private
research repository and published as a one-way public snapshot to
`github.com/againrom/againrom-knowledge`, consumed by format work as a git
submodule mounted at `knowledge/`. Its bytes live in that repository's history;
this repository records only a `.gitmodules` entry and a gitlink, never the
knowledge bytes themselves. The no-game-assets guard keys on git-tracked
content, so a populated `knowledge/` submodule checkout on disk never trips it.
Format facts are drawn from this published evidence, not from any third-party
source. A citation shaped `experiments/...` in a comment or a work item names a
path in the private research repository, not in this checkout; see
`knowledge/SOURCE.md`.

## Implementation procedure

To add or change a format parser or engine behaviour:

1. **Work from evidence, not from third-party source.** Derive the format from
   the evidence hierarchy above; cite the observation for each non-obvious claim.
2. **Write the specification first** (the SDD `spec.md` for the work item),
   detailed enough that a third party could implement a parser without reading
   our code — byte-level layout, endianness, string encodings (CP866,
   represented as bytes/hex, never literal non-ASCII text), and known quirks.
3. **Implement against synthetic fixtures.** Tests build byte streams in code
   from the documented contract; `go test ./...` stays green with no game
   present.
4. **Verify against real files with a developer tool** under `cmd/`, and record
   the evidence (counts, sizes, checksums) in the work item's `verification.md` —
   the evidence ships, the game bytes never do.
5. **Keep the boundary green:** run `scripts/check-no-game-assets.sh` (and
   `--history` before publishing) and keep `THIRD_PARTY_NOTICES.md` in sync with
   the compiled dependency set.

## Own-data-only rebuild policy

The repository is designed to be rebuilt from our own specifications and code
alone. A clean checkout, plus the user supplying their own lawful game install at
runtime via the `-assets` flag or `AGAINROM_ASSETS` (never a compiled-in path),
is sufficient to build and run; nothing in the build depends on redistributed
game data or third-party source.

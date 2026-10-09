# Provenance

`againrom` is an independent reimplementation. Its behaviour was established by
reverse engineering the author's lawfully owned original game for
interoperability, in a separate private research process. It must be
reproducible from our own specifications plus a user's own lawful game
installation. It never contains game data or the original program's code. It
does use third-party open-source components, each under a licence that permits
its use in a GPL-3.0-or-later work and each listed in `THIRD_PARTY_NOTICES.md`.
This document states the distribution boundary, the evidence hierarchy behind
our format work, and the implementation procedure.

## Distribution boundary

The boundary has two sides. The original game and any other material without a
licence for this project stay out. Licensed open-source components come in with
their notices.

### Excluded: the original game and unlicensed material

None of the following is ever committed or shipped:

- game archives, maps, registries, sprites, palettes, fonts, audio, video, or
  executables, in original or converted form;
- original program code, in binary form or as a disassembly or decompilation
  listing, and any code copied or adapted from the original program;
- source copied or adapted from another implementation of the game, or from any
  third party whose licence does not permit inclusion in a GPL-3.0-or-later
  work, or whose licence is unknown.

Code and documents cite the research findings by claim ID and, where a finding
is anchored to the original program, by function address. A citation names a
location and states the behaviour found there in the project's own words; it
does not reproduce the original's instructions. Some closed story records
written earlier quote short instruction fragments inside such citations; they
are historical records and new work does not add to them.

### Admitted: licensed open-source components

Two kinds of third-party code are admitted:

- **Compiled Go modules**, required in `go.mod`, such as Ebitengine and
  `golang.org/x/text`.
- **Source-derived ports** of an open-source library's code into this tree.
  `pkg/video/smacker` is a Go port of libsmacker 1.2.0 (Greg Kennedy,
  LGPL-2.1-or-later), which decodes the Smacker movie container and its audio.

Each admitted component meets four conditions:

1. Its licence permits distribution as part of a GPL-3.0-or-later binary.
2. `THIRD_PARTY_NOTICES.md` lists it with its version and SPDX licence, and its
   full licence text is under `LICENSES/`.
3. A ported file carries a header that names the library, its version, its
   author and its licence.
4. It contains nothing from the original game's program or data.

A third-party component is implementation, not evidence. libsmacker decodes the
general Smacker format; how the original game plays its movies is established by
research and cited by claim ID like any other game behaviour.

### How the boundary is held

- **Game assets: enforced mechanically.** `scripts/check-no-game-assets.sh`
  fails if any game-asset path is tracked, in the working tree or (with
  `--history`) anywhere in git history. It keys on git-tracked content, so a
  contributor's local game install or extracted bytes on disk are irrelevant;
  only tracked content can trip it. This mechanically enforces invariant P-1
  (the tracked repo contains no game data at any commit).
- **Admitted components: enforced mechanically.** `internal/notices` fails
  `go test ./...` when the module table in `THIRD_PARTY_NOTICES.md` disagrees
  with `go.mod` (`TestNoticesMatchGoMod`), or when a registered port lacks its
  notice row or licence text (`TestPortedSourceNoticesPresent`).
- **Original program code and unlicensed source: enforced by review.** Source
  has no distinctive extension, so no guard can detect copied code without
  either matching nothing or over-matching. Their exclusion is a policy upheld
  in code review, not a claimed automated mechanism.

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

Every claim about the game is confidence-tagged and traceable to such evidence,
never to another implementation of the game. The analysis of the original
program, which includes its disassembly, is carried out in the private research
process; the supporting analysis and byte-level observations are maintained
separately from this engine repository (see below), so this repo holds clean
specifications and code only.

## Research data (public knowledge snapshot)

Reverse-engineering notes, findings and byte-level evidence are maintained in a private
research repository and published as a one-way public snapshot to
`github.com/againrom/againrom-knowledge`, consumed by format work as a git
submodule mounted at `knowledge/`. Its bytes live in that repository's history;
this repository records only a `.gitmodules` entry and a gitlink, never the
knowledge bytes themselves. The no-game-assets guard keys on git-tracked
content, so a populated `knowledge/` submodule checkout on disk never trips it.
Facts about the game are drawn from this published evidence. A citation shaped
`experiments/...` in a comment or a work item names a path in the private
research repository, not in this checkout; see `knowledge/SOURCE.md`.

## Implementation procedure

To add or change a format parser or engine behaviour:

1. **Work from evidence.** Derive the game's formats and behaviour from the
   evidence hierarchy above; cite the observation for each non-obvious claim.
   A general-purpose format or service, such as a movie codec, may instead come
   from an admitted open-source component under the conditions above.
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
   `--history` before publishing). When a component is added, follow the
   procedure in `THIRD_PARTY_NOTICES.md`: its notice row, its licence text, and
   for a port its file headers and its `internal/notices` registry entry.

## Own-data-only rebuild policy

The repository is designed to be rebuilt from our own specifications and code
plus the admitted open-source components. A clean checkout, plus the user
supplying their own lawful game install at runtime via the `-assets` flag or
`AGAINROM_ASSETS` (never a compiled-in path), is sufficient to build and run;
nothing in the build depends on redistributed game data or original program
code.

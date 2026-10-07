# Spec — `.res` resource archive container (ROM1)

This story delivers a pure reader for the `.res` archive container and verifies it against a lawful
install (AC-8). Three header/node words carry no meaning the reader needs (header `0x04`, header
`0x0C`, node `0x00`): they are read raw and ignored. This is the base asset-I/O layer every later
story reads through (0002 sprites, the registries, maps). Greenfield.

## Problem / goal

Almost everything the game loads — sprites (`.256`/`.16`/`.16A`), class registries (`.reg`), fonts,
some maps — is bundled inside `.res` archives under path-like names. The engine needs a pure reader
that opens an archive, indexes its entries by path, and returns any entry's bytes by a
case-insensitive, separator-agnostic lookup. Pure `formats` leaf: byte stream in, structs out; no
engine knowledge, no VFS layering (that is 0027).

## Format definition (game-derived, little-endian)

The archive is a 24-byte header, a concatenated data blob, then a **registry** — a flat array of
32-byte nodes forming a directory tree:

`[ header (0x18) ][ data: [0x18, regOffset) ][ registry: nodeCount × 32-byte nodes ][ anything else: ignored ]`.

**Header (0x18 bytes):**

| Offset | Type | Content | Basis (research claim) |
|---:|---|---|---|
| 0x00 | u32 | signature `0x31415926` (`"&YA1"`, little-endian) | RES-MAGIC-001 |
| 0x04 | u32 | **opaque (R-1)** — read & ignored; its value law (∈ {0, non-root-node count}, filled only by graphics/main/sfx) is **one packer's habit**, refuted as format law by a second release, semantics open | RES-HDR-006, RES-HDR-012, RES-HDR-029 |
| 0x08 | u32 | `rootCount` — number of top-level (unowned) nodes | RES-HDR-005 |
| 0x0C | u32 | **opaque (R-1)** — read & ignored; per-file constant ∈ {1, 17} (bit 4 discriminates a format/writer variant), semantics open | RES-HDR-006, RES-HDR-013 |
| 0x10 | u32 | `regOffset` — byte offset where the registry starts | RES-HDR-003, RES-ACCEPT-031 |
| 0x14 | u32 | `nodeCount` — how many 32-byte records the registry holds; the **only** size input | RES-HDR-004, RES-ACCEPT-031 |

**Registry node (0x20 bytes)** — `nodeCount` of them, starting at `regOffset`:

| Offset | Type | Content | Basis (research claim) |
|---:|---|---|---|
| 0x00 | u32 | **opaque (R-1)** — reserved/always-0 across all tail-registry nodes (hash/id refuted); read & ignored | RES-NODE-011, RES-NODE-014 |
| 0x04 | u32 | **file:** payload byte offset · **directory:** index of first child node | RES-NODE-008 |
| 0x08 | u32 | **file:** payload size (bytes) · **directory:** number of children | RES-NODE-008 |
| 0x0C | u32 | type: `0` = file, `1` = directory | RES-NODE-008 |
| 0x10 | char[16] | name — NUL-terminated, `0xCD`-padded; printable ASCII in practice | RES-NODE-007 |

**Where the registry ends.** The registry is the `nodeCount` records that start at `regOffset`, and
nothing else: `[regOffset + nodeCount×32, EOF)` is **not part of the archive** and is ignored. A reader
MUST NOT derive `nodeCount` from the archive's length, and MUST NOT require `(EOF − regOffset)` to be a
multiple of 32. Those two identities describe one packer's output, not the format — the Russian
release's `MAIN.RES` is a valid 504-node archive whose registry region measures `504×32 + 23`, and a
reader asserting either identity rejects it (RES-ACCEPT-031, RES-OPEN-027, RES-GEOM-028).

**What is format law, and what is our hardening.** The signature is the only value the format itself
requires: the original accepts on the magic alone, cannot observe EOF at all, tolerates a short registry
read, and never checks a directory's child range (RES-OPEN-026, RES-OPEN-027). Every rejection FR-3
lists past the signature is therefore **ours, and stricter than the original** — each a corpus clause of
RES-ACCEPT-031, none of them format law. They are kept because a reader that indexes into a byte slice
must know the bytes are there, and because a total tree walk is what makes P-1 and P-2 provable; they are
declared here so no later story mistakes one of them for the format.

**Roots & tree.** Roots are the nodes **never referenced as a child of any directory node** — *not* a
header-pointed range (header `0x04` is opaque). `rootCount` (`0x08`) equals their count, a cross-check.
Walking from the roots and joining directory names with `/` yields every file's full path: a directory
node lists its children as the node range `[off, off+size)`; a file node's payload is
`bytes[off, off+size)` inside `[0x18, regOffset)`.

**Names and lookup.** A name is the bytes of its 16-byte field up to the first NUL. This reader
renders them as **CP866**, and states that as a display convention of its own: the format defines no
character encoding. Lookup **normalizes** a path — `\`→`/`, trim leading/trailing `/`, and fold case
**over ASCII `A`–`Z` alone**, every other byte comparing as itself — so `Units\Units.Reg` and
`units/units.reg` resolve to the same entry. The ASCII bound is normative: a general Unicode
lower-casing also folds Cyrillic, the Kelvin sign and the long s, and MUST NOT be used. The whole-path
ASCII fold is the original's own behaviour and is **required**, not a convenience of ours: every lookup
path is lower-cased in full through a CRT `tolower` that maps `A`–`Z` and nothing else (RES-CASE-036,
RES-LOOKUP-023). Rewriting `\` to `/` is ours: both separators are native terminators there and neither
is rewritten (RES-PATH-025), so the rewrite buys the same equivalence and changes nothing else.

**A record with no name.** A name field empty at its first byte is not a rejection. Its node's path
is its parent's — `""` for a root node, the directory's own path inside one — so the entry keys the
index under that. Two file nodes whose paths normalize alike are both listed by `Entries()` in node
order, and the index binds the key to the **first**; the later one is indexed but unreachable
through `ReadFile`, and a consumer must rule on an empty or repeated `Path` itself. No shipped
archive holds either shape, and what the original does with a zero-length node name is **not
decoded** — no claim covers it and no lawful input reaches it.

**Empty archive.** `nodeCount == 0` is a valid empty archive (e.g. `KIDS.LM` in the corpus, whose
`regOffset` is its EOF): it opens to an empty index, not an error.

**Two `&YA1` flavors.** This spec is the **tail-registry** flavor — `0x10` is a registry *byte offset* —
which occurs **only** as a standalone top-level `.res`/`.LM` (RES-SCOPE-015). The `.reg` stores and save
state use a different **inline** `&YA1` flavor where `0x10` is a *record count* and records start at
`0x18`, immediately after the header, carrying the same 32-byte node shape as the table above plus a
16-byte name; every `&YA1` found *nested inside* an archive or save is that inline flavor, not this one,
and is decoded by its own story (registries: 0011). A consumer must apply this reader only to a standalone
archive stream, never to a nested `&YA1` blob it extracted.

## Functional requirements

- **FR-1 (open & index)** — `Open(path)` reads the header and registry, walks the directory tree from
  the **root nodes** (nodes not referenced as any directory's child), and builds an in-memory index of
  `Entry{Path, Offset, Size}` for every **file** (not directory), keyed by normalized path. `Entries()`
  lists them in registry (node) order.
- **FR-2 (read by path)** — `ReadFile(path)` returns the exact `Size` bytes at the entry's `Offset` via
  a positioned read; lookup accepts `/` or `\` and is case-insensitive **over ASCII `A`–`Z` only**,
  every other byte comparing as itself. A missing path returns a not-exist error, not a panic.
- **FR-3 (atomic rejection, never panic)** — reject with an error and no archive: a bad signature; a
  `regOffset` outside the file; a registry the file does not hold — `regOffset + nodeCount×32 > EOF`; a
  directory child range that spans beyond `nodeCount`; a file whose payload range `[off, off+size)`
  exceeds the data region `[0x18, regOffset)`; a node with an unknown type; a directory nesting or child
  cycle that exceeds a fixed depth guard (defends against a cyclic registry). This list is closed, and
  everything in it past the signature is the disclosed hardening above; **bytes trailing the registry are
  never a rejection**.
- **FR-4 (purity)** — package `formats/res` imports only stdlib + `x/text` (CP866); it takes a file
  path / byte source and returns structs; it knows nothing about archive layering, the VFS, or the
  engine. No floats, no global state.
- **FR-6 (geometry from the header alone)** — the reader locates the registry at `regOffset` and reads
  **exactly** `nodeCount` records from there, computing that bound in 64-bit before it slices anything.
  It never derives `nodeCount` from the archive's length; that length constrains acceptance only by
  having to contain those records (FR-3).
- **FR-5 (dump tool)** — `cmd/restool` exposes `list` (entry names + sizes), `cat <path>` (one entry's
  bytes to stdout) and `extract <dir>` (all entries to a git-ignored folder) for developer-run
  verification against a lawful install — never part of the test suite.

## Acceptance criteria (synthetic archives, no GOG assets)

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a synthetic archive: header + a registry with a root dir, a nested subdir, and files, + packed data | opened | every file is indexed with its full `dir/sub/name` path, correct offset/size; `Entries()` is in node order |
| AC-2 | unit | an indexed archive | `ReadFile` with mixed case and `\` separators | returns the exact entry bytes; a normalized-equal path finds the same entry |
| AC-3 | unit | a path not present | `ReadFile` | a not-exist error, no panic |
| AC-4 | unit | a wrong signature; separately a `regOffset` past EOF | opened | error, no archive, each |
| AC-5 | unit | a directory whose child range runs past `nodeCount`; separately a file whose payload range exceeds `[0x18, regOffset)` | opened | error, no archive, each |
| AC-6 | unit | a node with type ∉ {0,1}; separately a cyclic directory link | opened | error (unknown type / depth guard), no panic |
| AC-7 | unit | a synthetic name with trailing padding and a non-ASCII CP866 byte | opened | the name decodes (CP866) to the expected trimmed string — expected value expressed as a Unicode code point, not literal Cyrillic (synthetic input, not GOG data) |
| AC-8 | manual | a real GOG `graphics.res` (lawful install, developer-run) | `restool list` | thousands of entries; each entry's payload range validates within the data region; results recorded as evidence in `verification.md` — no game bytes committed |
| AC-9 | unit | a valid empty archive (`nodeCount == 0`, `regOffset == EOF`) | opened | opens to an empty index; `Entries()` is empty; no error |
| AC-11 | unit | a valid archive with stale bytes after its registry — the shape a re-packer leaves, a 23-byte suffix of a node record, written as hex; separately one whose `nodeCount` is one short of the records the region holds | opened | both open; the first indexes exactly the entries of the same archive without the residue, the second omits the uncounted record and indexes the rest |
| AC-12 | unit | a `nodeCount` claiming more records than the bytes after `regOffset` hold — the disagreement in the other direction | opened | error, no archive (the disclosed hardening); nothing is read past EOF |
| AC-10 | unit | two entries whose names differ only in a byte `≥ 0x80` that a Unicode fold would collapse (a Cyrillic pair written as hex, never as literal non-ASCII text), plus a path differing only in ASCII case | opened, then `ReadFile` on each | the ASCII-case path hits; the two high-byte names stay **distinct** and each returns its own bytes — no fold is applied outside `A`–`Z` |
| AC-13 | unit | a registry naming a root file whose name field is empty; separately a nameless file inside a directory beside a root file of that directory's name; separately a nameless *directory* whose child shares a root file's name | opened | all three open; the first indexes the nameless record under the empty path and `ReadFile("")` returns its bytes; the other two each list two entries of one path in node order, with the index bound to the first |

## Derived properties

- **P-1** (invariant) Every indexed file's `[Offset, Offset+Size)` lies within the archive's data
  region; `ReadFile` never reads outside it.
- **P-2** (negative-invariant) For any malformed input (FR-3 conditions) the reader returns an error and
  no archive, and never panics or reads out of bounds.
- **P-3** (invariant) Path lookup is a pure function of the normalized path: ASCII case and
  `/`-vs-`\` never change the result — and, conversely, **no byte outside `A`–`Z`/`a`–`z` is ever
  folded**, so two paths differing only in a non-ASCII byte stay distinct.
- **P-4** (invariant) Acceptance and the index are a pure function of the header words and the
  `nodeCount` records at `regOffset`: appending any number of bytes to an archive that opens changes
  neither the outcome nor a single entry.

## Research needed

> **RESEARCH NEEDED — three opaque words [R-1].** Header `0x04`, header `0x0C`, and node `0x00` carry no
> meaning the reader needs. Their value laws were pinned on a widened 60-blob corpus of one packer's
> output and their **semantics stay open** (claims RES-HDR-012, RES-HDR-013, RES-NODE-014): `0x04 ∈ {0,
> non-root-node count}` (populated only by graphics/main/sfx, with no structural predictor for why) —
> **which a second release refutes as format law**, storing an in-table index instead (RES-HDR-029);
> `0x0C` a per-file constant ∈ {1, 17} (bit 4 discriminates a format/writer variant); node `0x00`
> reserved/always-0 across all 4592 of those nodes (a per-node hash/id was refuted). None of this reaches
> the reader, which reads all three raw and ignores them. A successful decode needs none of them,
> so this is **non-blocking**: the reader reads them raw and ignores them, relying on the signature +
> `regOffset` + `nodeCount` + the root-by-exclusion walk (all game-proven). Still a **research-team**
> item: *why* `0x04` is populated for only three archives, and *what* the `0x0C` variant flag and the
> reserved words mean — RE of `rom.exe`'s archive-open/-write routine. *(Header `0x14`, previously
> suspected opaque, is decoded as `nodeCount` — RES-HDR-004 — and is used by the reader.)*

## Out of scope

- Archive **layering** / `patch.res` override / one-namespace VFS and case-insensitive cross-archive
  lookup — story 0027 (`pkg/vfs`) on top of this reader. **Which** archive answers a path belongs there
  too: an archive answers only to paths whose leading segment is its own identity — its filename cut at
  the first `.` — so the shipped archives are disjoint namespaces and their order is a dispatch, not a
  priority (RES-IDENT-034). This reader indexes one archive, is handed paths relative to its own root,
  and has no notion of an identity to match.
- Decoding any *contained* format (`.256`/`.16`/`.reg`/`.alm`/fonts) — their own stories.
- Writing/creating `.res` archives; compression (entries are stored raw).
- Audio/video archives (`sfx.res`, `speech.res`, `movies.res`, …) beyond the shared container shape.

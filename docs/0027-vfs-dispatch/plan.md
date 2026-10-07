# Plan — multi-archive resolution by identity prefix

## Baseline

- `res.Entries()` lists a twice-held path once per record while `ReadFile` serves the first record;
  the reader's lookup key — the folded path, outer separators trimmed — is documented 0001 contract.
- The allow-map already grants `pkg/vfs → pkg/formats/res` and `cmd/restool → pkg/vfs`; no
  allow-map or ARCHITECTURE change in the story.
- The graphics suppliers: `cmd/terraintool` feeds `terrain.LoadTileset`, `game.LoadStatics` and
  `game.LoadUnits` off one `res.Open`; `cmd/mapview` reaches the same three through
  `game.OpenGraphics`. `open <path>: <err>` is frozen wording in both; the statics/units suites
  assert the load error *contains* the registry constant.
- Fixtures key by the consumers' own constants — the menu and terrain suites also pin them as
  literals. `cmd/againrom`'s suite writes a synthetic install tree and runs `NewFrontEnd` end to
  end, naming the container entries it writes by those same constants;
  `pkg/game`'s loader fixtures are `res.OpenBytes` archives handed in directly; `frontend_test`
  assembles `Archives` by hand.
- The picker displays `MapEntry.Source` verbatim: archive rows the folded entry path, loose rows
  the `os.ReadDir` name, real case.

## Design decisions

### DD-1 — a bespoke surface of five names, not `io/fs.FS`

`pkg/vfs` exports `FS`, with `Open(archives, dirs []string) (*FS, error)`, `ReadFile(address
string) ([]byte, error)`, `Entries() []Entry`, `Locate(address string) (Source, bool)` and
`Identity(hostPath string) (string, error)`. `Entry` is the folded address, its `Source` and the
byte size; `Source` names the tier, the identity and the list position — one shape for FR-9's
listing and FR-10's report, so same-source claims compare equal values. `ReadFile` keeps the two
render seams' exact signature — `menu.EntrySource` and `terrain.EntrySource` each declare
`ReadFile(name string) ([]byte, error)`, verified — so `*vfs.FS` satisfies both unchanged.
`Identity` exports FR-1's derivation: AC-2 tests it directly, and `pkg/game` derives its scenario
prefix from the archive-name constant it already owns, not a second literal to drift.

Rejected: `io/fs.FS` — slash-only, case-significant paths against FR-3's folded interchangeable
separators, and a directory walk against FR-9's flat archive-naming listing; no consumer in this
repo asks for it — both seams ask for `ReadFile`.

### DD-2 — one snapshot at open; read, locate and enumerate walk the same data

`Open` opens every listed archive up front — first failure returns `(nil, error)`, retaining
nothing (FR-2) — and records directories untouched. Per archive it snapshots `Entries()` once into
an ordered path→size index, first record winning — the reader's own resolution. A read probes the
snapshot then serves bytes through the winning archive's `ReadFile`; `Locate` and `Entries` walk
the same snapshots minus the byte copy. FR-10's same-source rule and P-3's idempotence hold by
construction: one walk, nothing re-reading a host. The snapshot key applies the reader's
documented rule — outer separators of the remainder trimmed, the fold already address-wide — so
probe and read cannot disagree. The open failure is
`*fs.PathError{Op: "open", Path: host}` wrapping the archive's own error: it
prints `open <host>: <err>`, byte-identical to what `OpenArchives`, `OpenGraphics` and
`terraintool` print today, so their wrappers are dropped, not re-stacked.

Rejected: probing presence with `res.ReadFile` and continuing on `ErrNotExist` — `Locate` answers
without bytes and `Entries` needs sizes, so both need a second mechanism, and two mechanisms is
how the report and the read drift.

### DD-3 — the address grammar lives in one function, and it does not trim

One fold at every entry point: `\` rewritten to `/`, ASCII `A`–`Z` lowered byte-wise, nothing
trimmed — reusing the reader's trim would erase FR-3's refusal, since `\graphics\x` trims to a
resolvable address. Then, in order: an empty leading segment (the empty address, or one beginning
with a separator) is answered absent before any tier; a leading segment equal — never merely
prefix-related — to a registered identity selects the archive walk, the segment consumed and a
non-empty remainder required; a miss everywhere is `*fs.PathError` wrapping `fs.ErrNotExist` and
naming the address (FR-6). A separator-less string is not refused: it is all leading segment, no
archive answers it, and the directory tier is consulted as for every address (FR-8).
FR-1's identity derivation runs once, at open, through this same fold; an empty result refuses
the whole open.

### DD-4 — the directory tier: verbatim segments, and holds means a regular file

The tier looks up the *whole* folded address — identity segment not consumed — as a relative path
under each directory in order. The host path is built from the folded segments verbatim, never
through `filepath.Clean`, and an address holding a `.`, `..` or empty segment is answered absent
by the tier: cleaning would resolve `a/../b` outside the listed root. A directory **holds** an
address iff a regular file stands at that path; the first holder answers, anything else there is
not held and the walk continues —
FR-8's first readable file — while a held file whose read fails non-not-found surfaces unmasked
(FR-7; the loose tier is its only reachable witness, the archive reader failing not-found alone).
`Locate` stats for the same regular file, so both walks stop at the same directory.

Rejected: stopping at the first path where anything exists. It reads FR-7's masking ban into
FR-8's walk — contradicting first-readable-file the moment a directory stands at an earlier
root's path — and turns a bare registered identity into a surfaced host error no contract names.

### DD-5 — enumeration is the read's winner, sorted byte-wise

`Entries` walks archives in list order, each snapshot in registry order, emitting
`identity + "/" + path` on first occurrence and dropping every later one — the same winner every
read resolves to, within an archive (first record) and across same-identity archives (first in
list) — then sorts ascending by plain byte comparison of the folded address. Directories are
never enumerated (FR-9). Rejected: dedup keeping any later record — the listing then names a
source the read does not use (AC-11's exact failure).

### DD-6 — the identity moves into the consumers' constants; the seams stay

Each consumer states its own container in its own path constants:
`menu.EntryPrefix` becomes `main/graphics/mainmenu/`, `terrain.TilePathPrefix` and `DirtPath` take
`graphics/`, `game.ObjectRegistry` and `game.UnitRegistry` become `graphics/…` addresses, and the
sheet cache prefixes the graphics identity onto each registry `File` value at the read.
`LoadStatics`, `LoadUnits` and `sheetCache` re-type from `*res.Archive` to `terrain.EntrySource` —
which `*res.Archive` also satisfies, the property DD-8's staging leans on. The statics/units
errors keep containing the registry constants, now addresses, so both error contracts hold
unedited.

Rejected: a prefixing adapter in `pkg/game` per render package — it leaves every library caller
resolving identityless paths, FR-12's ban verbatim, and rebuilds the out-of-band "who holds what"
knowledge this story exists to delete, one adapter per consumer.

### DD-7 — the map list keeps its rows and loses its second code path

`ArchiveMaps` rebuilds over the container filesystem's enumeration and the scenario identity,
filtered to `.alm`; `Names()` returns the remainders, so every displayed `Source`, every ordering
key and `CheckLine` stay byte-identical (FR-12's rendered-result freeze). `DirMaps` keeps its
`os.ReadDir` scan — FR-9 forbids enumerating the directory tier, so the loose *listing* is
legitimately a scan — but its `Read`, `archiveMaps.Read` and `frontend.mapBytes` all become vfs
reads: an archive row under its scenario address on the container filesystem, a loose row under
its bare name on the loose one (DD-8). `os.ReadFile` and `res.ReadFile` leave both files;
`FromArchive` now only names the address a row listed under.

Rejected: making `Source` carry the full address. It folds harder but changes every picker row the
screen shows, which FR-12 freezes.

### DD-8 — the migration is supplier-first, one consumer per commit, teardown last

Order: vfs first, nothing depending on it; then `Archives` gains its two filesystems — containers
over the three host paths and **no** directories, loose over no archives and the root — beside
the three handles it keeps until the end; then the consumers flip one at a time — menu, the
object/unit loaders, the tileset, the map list — each flip one commit
moving the consumer's constants, suppliers and pinned literals together, so no commit leaves a
constant asking for an address no supplier serves, or the reverse; last, the three handles and
every wrapper DD-2 obsoletes come out — `Archives` keeps `Root` and the two filesystems,
`OpenGraphics` returns the filesystem it opened — and the AC-15 census lands. The census walks
the consumers' own surfaces — `menu.Entries()`, the full `TilePath` grid and `DirtPath`,
the registry constants, sheet addresses built by the cache's own rule from a synthetic registry,
the listed map rows — so an address enters it through the same constant its consumer reads.
Between the loader and tileset flips the two developer front-ends carry archive and filesystem
side by side; the tileset flip retires the archive. The transitional double-read of graphics.res
is the accepted price: every commit builds, vets and tests green, none hollow — a consumer
flipped early would pass its suite on `LoadTileset`'s absence-is-normal contract while resolving
nothing.

Rejected: listing the root beneath the archives on one filesystem. FR-8 then sends every
archive-miss to the host tree: a stray unpacked file under the root silently fills a tile slot
that is nil today — accidental masking the out-of-scope refuses — at a hundred useless host
probes per start. Rejected: one commit flipping suppliers and consumers at once — an unreviewable
diff whose one defect strands the whole migration.

## Risks

- **R-1** The identity comes from the host filename, so a developer flag pointed at a renamed copy
  resolves nothing under the canonical addresses: an empty tileset, a load error naming
  `graphics/objects/objects.reg`. Accepted: errors and the FR-10 report name the address and
  identities in play; a canonical install is unaffected.
- **R-2** Loose map reads now fold: on a case-sensitive host a capitalised loose file
  (`Beast.ALM`) stops resolving — the disclosed C-3 limitation reaching shipped behaviour through
  `DirMaps`. Accepted: shipped installs live on hosts that fold, and the row still lists as
  unreadable rather than vanishing.

## Success criteria

- **SC-1** AC-1, AC-2, AC-8, AC-9 hold in full; P-6 sampled over every failing-open fixture
  (FR-1, FR-2, C-2).
- **SC-2** AC-3, AC-4, AC-6, AC-7 hold in full, every miss error naming the address it refused;
  P-5 sampled by permuting distinct-identity lists and re-reading every address in play (FR-3,
  FR-4, FR-6).
- **SC-3** AC-5, AC-10, AC-13 hold in full — AC-10 on the loose tier, the only one that can fail
  non-not-found: the earlier directory holds the address as a regular file whose read fails,
  injected host-appropriately, and the later copy is not returned (FR-5, FR-7, FR-8).
- **SC-4** AC-11, AC-12 hold in full; P-1 and P-3 sampled: repeated reads and enumerations
  byte-equal, every listed address read back against its listed source (FR-9, FR-10).
- **SC-5** AC-14 holds in full: returned bytes mutated and re-read, the listed directory's tree
  compared unchanged across every read and enumeration; and a `..`-holding address is absent with
  nothing outside the listed root touched (FR-11, P-2).
- **SC-6** Four mutants, each measured and reverted with its failing tests named: the identity
  equality weakened to a prefix accept; the leading-separator refusal deleted; the same-identity
  walk reversed; the enumeration dedup keeping the later record (FR-3, FR-4, FR-5, FR-9).
- **SC-7** The consumer suites pass with fixtures keyed by the flipped constants — `pkg/ui`
  unedited, the menu and terrain suites edited only in their pinned literals, `cmd/againrom`'s
  install builder only to write container entries container-relative — and every expected
  picker-row string, `CheckLine`, and frozen `open <path>:` failure text is byte-identical
  (FR-12).
- **SC-8** AC-15's census holds in full over a synthetic install, every library address carrying
  an identity segment and yielding the container's own bytes; and `pkg/game`'s production files
  import `pkg/formats/res` nowhere (FR-12, C-4).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1, FR-2, C-1, C-2 | DD-1, DD-2, DD-3 | SC-1 |
| FR-3, FR-4, FR-6 | DD-3 | SC-2, SC-6 |
| FR-5 | DD-2, DD-5 | SC-3, SC-6 |
| FR-7, FR-8, C-3 | DD-4 | SC-3 |
| FR-9, FR-10 | DD-1, DD-2, DD-5 | SC-4, SC-6 |
| FR-11 | DD-2, DD-4 | SC-5 |
| FR-12, C-4 | DD-6, DD-7, DD-8 | SC-7, SC-8 |

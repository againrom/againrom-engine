# Tasks — multi-archive resolution by identity prefix

Legend: **files** what the task may change · **fences** what it must not do · **done when** the
observable it leaves behind. Every entry is an implementation task; they land in ascending order,
each depending only on entries before it. A mutant is run by the entry that owns it; a survivor is
reported as a survivor — "killed elsewhere" needs an actual run against that tree.

## T1 — open, the address grammar, and a read through both tiers

**files** ADD `pkg/vfs/fs.go`, `pkg/vfs/fs_test.go`

DD-1, DD-2, DD-3, DD-4 — FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-7, FR-8, FR-11.

**fences** no file outside `pkg/vfs`; imports are `pkg/formats/res` and stdlib alone. Of DD-1's
surface, `Open`, `ReadFile` and `Identity` land here — `Entries`, `Locate` and their
`Entry`/`Source` shapes are T2's — but the open already builds the per-archive snapshot all of
them walk. AC-10's earlier directory makes its file unreadable by whatever the host running the
suite honours; if the host cannot express one, the case skips naming why rather than passing on
nothing.

**done when** SC-1, SC-2 and SC-3 hold over this entry's tests, and SC-6's first three mutants —
the prefix accept, the deleted leading-separator refusal, the reversed same-identity walk — are
each applied, run with the failing tests named on record, and reverted.

## T2 — enumeration and locate over the same snapshots

**files** MODIFY `pkg/vfs/fs.go`; ADD `pkg/vfs/enum_test.go`

DD-1, DD-2, DD-5 — FR-9, FR-10, FR-11.

**fences** no file outside `pkg/vfs`; no second index beside T1's snapshots and no re-read of any
host. AC-14's mutation probe covers enumeration as well as reads. P-4 is sampled here through
`Locate`: every sampled address answers as exactly one of archive, directory, or absent, T1's
fixture shapes reused across both tiers.

**done when** SC-4 and SC-5 hold over this entry's tests, and SC-6's fourth mutant — the dedup
keeping the later record — is applied, run with the failing tests named on record, and reverted.

## T3 — Archives gains the two filesystems, and the menu flips

**files** MODIFY `pkg/game/archives.go`, `pkg/game/frontend.go`, `pkg/render/menu/menu.go`,
`pkg/render/menu/menu_test.go`, `pkg/render/menu/state_test.go`, `cmd/againrom/main_test.go`

DD-6 (the menu constant), DD-8 — FR-12.

**fences** `pkg/ui` untouched. The three `res.Archive` fields stay and every other consumer —
loaders, tileset, map list — keeps reading them; only `menu.Load`'s source and `menu.EntryPrefix`
flip. Both filesystems open in `OpenArchives`, though the loose one gains its first reader in T6.

**done when** the menu suite passes with its pinned prefix literal flipped, `pkg/ui`'s suite
passes unedited, `cmd/againrom`'s install builder writes menu entries container-relative, and
every frozen picker, `CheckLine` and `open <path>:` string is byte-identical (SC-7's slice for
this flip).

## T4 — the object and unit loaders flip; the front-ends carry both

**files** MODIFY `pkg/game/statics.go`, `pkg/game/units.go`, `pkg/game/frontend.go`,
`pkg/game/archives.go`, `pkg/game/statics_test.go`, `pkg/game/units_test.go`,
`cmd/terraintool/main.go`, `cmd/mapview/main.go`, `cmd/terraintool/units_test.go`,
`cmd/terraintool/unitanim_test.go`, `cmd/terraintool/statics_test.go`,
`cmd/mapview/main_test.go`, `cmd/againrom/main_test.go`

DD-6 (the registry constants, the sheet prefix, the `terrain.EntrySource` re-type), DD-8 — FR-12.

**fences** the tileset keeps the archive: `terrain.TilePathPrefix`, `DirtPath` and every
`LoadTileset` call are T5's. `pkg/game`'s loader suites key fixtures by the constants and pass
without fixture edits; a test is edited only where it wrote a host container entry keyed by a
constant that is now an address — the entry goes container-relative, the assertion keeps the
constant.

**done when** both loaders resolve their addresses through the containers filesystem, both
developer front-ends carry archive and filesystem side by side, and every registry-naming load
error and frozen `open <path>:` string is byte-identical (SC-7's slice for this flip).

## T5 — the tileset flips, and the front-ends retire the archive

**files** MODIFY `pkg/render/terrain/tileset.go`, `pkg/render/terrain/tileset_test.go`,
`pkg/game/frontend.go`, `pkg/game/archives.go`, `cmd/terraintool/main.go`,
`cmd/terraintool/main_test.go`, `cmd/terraintool/statics_test.go`, `cmd/mapview/main.go`,
`cmd/againrom/main_test.go`

DD-6 (the tile constants), DD-8 — FR-12.

**fences** the three `Archives` fields, `OpenGraphics`' and `OpenTileset`'s signatures and the
wrappers DD-2 obsoletes stay — teardown is T7's; the tileset inside them is built off the
filesystem. After this commit no front-end holds a `*res.Archive`; the map list still reads the
scenario handle.

**done when** the terrain suite passes edited only in its pinned literals, the full `TilePath`
grid and `DirtPath` resolve under the graphics identity, both front-end suites are green, and
every frozen string is byte-identical (SC-7's slice for this flip).

## T6 — the map list loses its second code path

**files** MODIFY `pkg/game/maplist.go`, `pkg/game/frontend.go`, `pkg/game/maplist_test.go`,
`pkg/game/frontend_test.go`, `pkg/game/world_test.go`, `pkg/game/frontend_statics_test.go`,
`cmd/againrom/main_test.go`

DD-7, DD-8 — FR-8, FR-12.

**fences** `DirMaps` keeps its scan; `BuildMapList`, its ordering keys and `MapEntry`'s fields
stay untouched. A loose row's read is the spec's separator-less address on the loose filesystem —
assert that route, never a host read beside it.

**done when** the maplist and frontend suites pass with every displayed `Source`, ordering and
`CheckLine` byte-identical (SC-7's slice for this flip); archive rows read under scenario
addresses, loose rows under their bare names; `mapBytes`, `archiveMaps.Read` and `dirMaps.Read`
hold no `os.ReadFile` and no `res.ReadFile`.

## T7 — teardown, and the census

**files** MODIFY `pkg/game/archives.go`, `pkg/game/frontend.go`, `pkg/game/archives_test.go`,
`pkg/game/statics_test.go`, `cmd/mapview/main.go`, `docs/ARCHITECTURE.md`; ADD
`pkg/game/address_census_test.go`

DD-2 (the obsoleted wrappers come out), DD-8 — FR-12, AC-15.

**fences** `docs/ARCHITECTURE.md` changes by one clause only — the classdump row's "`pkg/vfs`
being a stub" no longer holds — nothing structural. The census enters every address through the
constant its consumer reads, per DD-8; a census listing addresses of its own is the failure it
exists to catch.

**done when** SC-8 holds: AC-15's census passes over a synthetic install, `Archives` and
`OpenGraphics` carry DD-8's teardown shape with no `*res.Archive` in either surface, `pkg/game`'s
production files import `pkg/formats/res` nowhere, and every remaining frozen string is
byte-identical (SC-7's remainder).

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-7, FR-8, FR-11, AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-7, AC-8, AC-9, AC-10, AC-13, P-5, P-6 | DD-1, DD-2, DD-3, DD-4, SC-1, SC-2, SC-3, SC-6 |
| T2 | FR-9, FR-10, FR-11, AC-11, AC-12, AC-14, P-1, P-2, P-3, P-4 | DD-1, DD-2, DD-5, SC-4, SC-5, SC-6 |
| T3 | FR-12 | DD-6, DD-8, SC-7 |
| T4 | FR-12 | DD-6, DD-8, SC-7 |
| T5 | FR-12 | DD-6, DD-8, SC-7 |
| T6 | FR-8, FR-12 | DD-7, DD-8, SC-7 |
| T7 | FR-12, AC-15 | DD-8, SC-8 |

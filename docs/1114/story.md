# Saved structures and cell links

## Result contract

Both original-SAV LOAD paths restore the saved Building roster, including
source-only, moved, removed and multiply referenced objects. Source archive
identity, authored script identity and native structure handles remain distinct.
Saved cell links, not a replay of rectangle attachment, select area-hit targets.
The saved block overlay supplies static pathing while actor occupancy remains
simulation state. Art and inspection use the restored roster before the first
frame. Ordinary native SAVE, fresh LOAD and the next action retain this state.

## Authority and boundaries

`SAV-DOC-053`, `SAV-BLDG-037`, `SAV-TOKENPOS-074`,
`SAV-CELLLOAD-108` through `111`, `SAV-RECON-268`/`269` and
`SAV-POSTLOAD-221` separate the roster, Position, cell overlay and render sender.
`TERR-STRUCT-070` through `072` separate the blocking and attachment masks and
exclude constructor attachment replay on LOAD. `UNIT-STRUCTDETACH-074` does not
establish the HP-to-destruction schedule.

Base Building operations retain the existing bounded physical/area damage and
ruin policies. Outpost, Tavern and Shop retain their full decoded suffixes and
base geometry, health and cell references; no subclass service is inferred from
an empty tick (`SAV-POSLOAD-140`). `SHOP-SAVE-015` carries only the shop value cap;
`SAV-CLASSSER-173`/`176` and `SAV-LOADRESAVE-208` establish suffix grammar, not
runtime acceptance or subclass service semantics. Unknown base fields and raw
Token references are retained, not assigned invented meaning or behaviour.
In particular no saved owner is inferred from a Token word. The original
HP-to-destruction schedule and unknown subclass operations remain open.

This is a prerequisite of the world SAV writer, not a claim that ordinary
mission SAVE already writes SAV or that ROM1 accepts our output.

## Proof

Independent literal fixtures cover moved and source-only authored-ID-zero
Buildings, missing ALM placements, signed HP and zero maximum, both masks,
duplicate archive references, reminted keys, overlapping/partial registration,
off-rectangle and out-of-map aliases, duplicate cell last-writes and null clears.
Parser, both LOAD doors, first-frame art/hover/card, App physical attack, actual
area-cast dispatch, movement, ordinary SAVE and fresh-process next movement pass.
The pending area cast also survives native continuation and damages the saved
alias target, not the geometric owner. All three subclass suffixes survive.

Form79 is the complete form78 followed by the optional saved-structure payload
and a little-endian uint32 byte span. Zero span is explicitly absent legacy
mode; present empty carries two zero counts. A source record is107 bytes plus
its eight-byte Outpost records; a final cell record is9 bytes. Existing entity
and live22-byte structure strides do not change. Raw base+52 overlaps later
width/height/blocking stores: only the latter runtime values are canonical.

An independently generated synthetic fixture from published1113
`c919f44733c5752a95f78f92af731e5a32f67291` proves genuine form78 Group continuation.
Every older readable form rejects the new reserved static-domain bit. The
literal eight-mask table covers Ground/Air/Ghost and their combinations;
saved Ghost uses its own original static-bit2 domain, while fresh/legacy keeps
its existing policy. Late malformed native/source references are atomic.

The installed witness changes what the previous health-only handoff could show:
authored7 moves to102,47 and former8 becomes a source-only object, with no ALM8
ghost. Installed art, hover/card and native SAVE/LOAD consume the restored roster.
No original ROM1 runtime or physical desktop interaction is claimed.

## Candidate verification

Initial base is public1112 `3c030c75100767540b454224d7bc5bf07678ddad`.
The final code checkpoint `8d6fca0488d0907fd372e64a25d933da0cfc6c40` composes
corrected, live-origin-verified1113 `75a07d1b2214eb24f77eb187ea216b878f8b8911`;
research stays at promoted `dde13d1d75c192a14250dec948eeb388af4275e0`.

At that checkpoint, changed Go files are gofmt-clean;
`go test -trimpath -count=1 ./...`, no-game-assets and divergence-claim guards pass.
The claim guard's partial-retraction notices for DIV-666/795 were checked against
SAV-RECON-268: its retracted universal Human-row selector is not their cited
Building/cell or hybrid-load clause. No retracted selector is used here.

Own rebuilt binaries pass headless0152-save666 and1005-doll-carry-over-worn on
both EN and RU. The28-map census is unchanged against
`pipeline/milestone-baseline.txt`: mission10 and20 have0 unsupported nodes before
and after; both unattended drives lose at304 with4/36 moved and1 fallen. This is
not a campaign-passability claim. The player result is the installed roster and
spatial continuation above, not a reduction of that already-zero script count.

One final paired invocation passes EN150/150 and RU150/150, with0 tests lacking
a subject on either root, including the new installed roster witness and the
existing1098 health/ruin,1100 dead-weapon and multi-cell area-spell controls.
The approved test-owned legacy-city copy supplies the frozen-city witness;
inherited `GOFLAGS=-buildvcs=false` also applies to its nested386 video helper.
Earlier attempts found and corrected the readable-form78 census entry and the
old dead-control test's native footer walker. Invocation-only failures were a
relative install path, an omitted approved legacy-city fixture, and nested
helper VCS stamping; none is counted as a passing release run.

Gate logs remain ignored under `builds/1114/`: `full-go-composed.log`,
`release-verified.log`, `milestone-composed.log`, the four
`scenario-composed-*.log`, `noassets-composed.log` and `div-claims-composed.log`.
After all install-capable commands the preservation guard passes181 files,
both roots unchanged. Allocation sweeps bracketed ledger edits with32 ledgers,
0 missing answers. The research pin is a descendant of the initial base pin;
there are no file deletions or Co-Authored-By trailers. Subsequent commit changes
only this proof document, not the measured code.

DIV-666 narrows to remaining continuation/output debt; DIV-730 narrows its cell
payload gap, and DIV-552 keeps the unknown destruction schedule. DIV-794 records
retained-but-unimplemented subclass/base semantics; DIV-795 records atomic
refusal instead of the original unresolved raw-key fallback. Reserved796..801
are unused. No research experiment was started.

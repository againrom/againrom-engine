# SAV structure health

## Result and scope

Restore current and maximum health for a present, uniquely matched map-placed
base Building before the mission's first frame or tick. The two saved words
reach structure inspection, signed ruin selection, combat and native SAVE/LOAD.
This slice does not complete original-SAV world writing or original round trips.

## Contract

- FR-1: read Buildings through the exact counted document walk, including
  intervening classes and the final endpoint. No class-tag search or partial
  result on malformed input.
- FR-2: bind the full nonzero Token authored ID to exactly one ALM type-4 ID.
  Reject ambiguous source or target bindings before any health write. Runtime
  IDs and source order never select a target.
- FR-3: restore both raw health words only for a base Building whose class,
  position, dimensions and masks agree with the constructed placement. Count
  unsupported subclasses, unbound/unmatched objects, changed topology and
  map-only placements. Absence does not mean destruction.
- FR-4: use the same handoff in diagnostic and ordinary LOAD. Native SAVE/LOAD
  preserves the resulting hash, health, inspection/ruin and combat continuation.

## Design and authority

DD-1: `sav.Buildings` shares `groundDocument` with GroundSacks. The parser
retains typed values, not source bytes. `SAV-BLDG-037` and `SAV-TOKEN-034`
define its members; `SAV-ID-015` separates authored and runtime identity.

DD-2: `applyOriginalStructures` stages a unique join and requires agreement
with `mapload.Footprints`. `ALM-CLS-053` binds the fresh health pair and table
row; `TERR-STRUCT-070`, `TERR-STRUCT-071` and `TERR-STRUCT-072` bind dimensions,
blocking, attachment and load ordering. `sim.ImportOriginalStructureHealth`
changes only the two existing hashed words. No byte-form change.

DD-3: zero and signed-negative health are retained, not clamped or converted
into a deletion. `TERR-STRUCT-102` supplies the existing ruin selector.
`SAV-CELLLOAD-109`, `SAV-CELLLOAD-110`, `SAV-CELLLOAD-111` and
`SAV-CELLLOAD-113` establish a separate saved-cell identity overlay that this
slice does not import. `UNIT-STRUCTDETACH-074` and `UNIT-STRUCTSTOP-066` leave
the whole-world health-to-destructor transition Unknown; existing DIV-547
retains ruined objects and their obstruction.

## Proof and open debt

`verification.md` records independent fixture checks and lawful EN/RU controls.
Full structure population, saved cell attachments, changed shapes, runtime
spawn/removal, subclass state and original-SAV world output remain open.

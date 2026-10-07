# Story `1033` — claim provenance

`contract.md` names the claim table before the work; this file crosswalks each cited row to the
as-built line that implements it. Read a row whole with `go run ./tools/claim <ID>` from
`research/`; this file does not restate a row's own text, only where it lands.

Pin: `753034d`, unchanged from the worktree's creation (no bump during this story).

| Claim | Confidence | Implements |
|---|---|---|
| `TRIG-CHECK-051` | High | `ScriptCheckHealth` case, `pkg/sim/script.go`: the `scriptCheckHealthGate` (6) test, `Entity.HP` read on the gate, nothing written off it. |
| `TRIG-CHECK-052` | High for the mechanism (the found branch's only use of the container-search return is the null test; the distance source is the unit, re-loaded from `e.X`/`e.Y`), Medium for the corpus counts | `ScriptCheckItemDistance` case: `containerHolds` over `w.carried[i]`, `0xff` on a miss, `scriptDistance(e.X, e.Y, c.Args[0], c.Args[1])` on a hit — the unit's own position, not the item's. |
| `TRIG-CHECK-053` | High for the mechanical facts (offset, 16-bit width, unconditional access, no selector or notification call on instant 26), Unknown for the field's semantic label | `Structure.Field42` (`pkg/sim/structure.go`), `ScriptCheckStructField` and `ScriptInstantStructField` cases. The field is named for its offset (`structure+0x42`) and not for a meaning; the code comment on `Structure` states this and cites `DIV-267` for the initial-value question the claim leaves fully open. |
| `TRIG-CHECK-054` | High for the widths and the gate literal, Medium for the authored/reachable/maps counts | The corpus figures this row gives (10/7/16 per root, 33 total, 11 maps) are the `DIV-239` baseline this story's `closure.md` measures against, and the limit split (check 4's gate and check 16's 8-bit width are engine constants; check 21's width is a stored save-record field) is carried into `DIVERGENCES.md`'s closed `DIV-239` row under G2. |
| `SAV-BLDG-037` | High | Cross-confirms `TRIG-CHECK-053`'s 16-bit width from a third site (`Building::Serialize`) sharing no code with either script arm, and measures the field's own value in the owner's original saves. This story adds no save-record `Building` type; it seeds the simulation-side `Structure.Field42` the two script arms read and write. See the seed table below. |

## Claims consulted and not used as a source of a value

`EXP-0080`'s uncited vocabulary note (`evidence/rom-vocabulary.md`) reads check 16's distance as
`dist(item,(p0,p1))`. `TRIG-CHECK-052` refutes it by reading both branches of the arm to their end;
this story's `ScriptCheckItemDistance` case follows `TRIG-CHECK-052`, not the note. The note is
recorded here only as the trap `contract.md` and the brief both named, not as a source consulted for
any value in the implementation.

## Claims for the ledger rows this story writes or closes

`DIV-267` ("Authored where research is silent") cited `TRIG-CHECK-053` for the field's mechanics and
stated, following the same claim's own search, that `ALM-OBJ-019`'s candidate fields on the type-4
record are each already assigned a different role and none is cited as `Field42`'s source. **That
statement was wrong when it was written, and the row is closed** (`DIVERGENCES-CLOSED.md`, adversarial
pass 1, 2026-08-23). The value does not come from the map record at all, so a search over the type-4
record's own fields could not have found it; the search that answers is
`go run ./tools/claim -k` over the SUBJECT across every ledger.

Three rows in this story's own pin give the seed, and each was read whole with its evidence file at
`753034d` before the field was seeded:

| Claim | Confidence | What it gives | Where it lands |
|---|---|---|---|
| `ALM-CLS-053` | High | The type-4 spawn's own parameter consumption: `sizeX`/`sizeY` to the footprint, `scanRange` to `obj+0x48`, `healthMax` to `obj+0x44`/`+0x42`. `+0x42` is the word both script arms read. | `mapload.Structures` (`pkg/mapload/fromalm.go`) seeds `Field42` from that position; the 16-bit store is the destination width the same row names. |
| `SAV-BLDG-037` | High | `+0x42`/`+0x44` measured as equal non-zero pairs in the owner's own original saves: 1000/1000, 30000/30000, 100/100, 2000/2000. Reads the shape as a current/maximum pair and states that the meaning stays Unknown. | Corroborates the seeded value independently of the spawn listing. The row's Unknown is why the field stays offset-named. |
| `DAT-BLD-005` | — | The shipped per-kind `healthMax` figures, which are the four values `SAV-BLDG-037` measured. | Joins the two above: the saved value is the table's value. |
| `DAT-SCHEMA-004` | — | `healthMax` is Buildings column 4. | A column is 1-based and a parameter position is 0-based, so the position is 3. `structures_test.go`'s own `slotBlocking = 4` and `slotAttach = 5` for columns 5 and 6 agree with that indexing independently. |

`DIV-264` (FIDELITY-DEBT, OPEN) is what remains after the seed, and it cites `ALM-CLS-053` and
`SAV-BLDG-037` for what is known and records that no claim in this repository gives what writes
`+0x42` after the spawn: no damage path, no destruction test and no consumer of the pair is decoded.

`DIV-239`'s closure cites all five rows above; no new claim was needed to close it, since `EXP-0215`
(the experiment publishing all four `TRIG-CHECK-05x` rows) had already answered the row's own open
question before this story began.

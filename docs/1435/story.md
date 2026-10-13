# Players and sides in the map editor

Initial base: `77c9e77506bce11ae5bcc9718c0444522dc61c3a`.
Merged base: `62ca3ea47e18b7b0048e6d3ee37b40dda90d3a38`, including the
objects brush. The player window extends that version's key and mode tables.

## Intent and builders

The author edits the type-5 roster, saves the map and plays it as a mod mission.
The owner requires safe deletion: a referenced player cannot be removed, and
later references retain their player identity after an unreferenced row is removed.

`pkg/mapedit/section.go` owns the one record-section builder, `sections`.
`KindPlayer` is its type-5 row: 76-byte records, metadata count at `+0x1c`,
no map cell, placement validation, absent-section creation and a deletion hook. `Place`, `Delete`,
`Record`, `Fields` and the existing splice history consume that row.
`playerFields` supplies the name, colour, control and directed relation fields.
The common field builder supports fixed-width text through `TextFieldValue`
and `SetTextField`. There is no separate player byte model.

`mapInspectionEdits.change` remains the transaction and inspection builder.
It adopts the edited model only after its projection succeeds. The existing
save writers consume `SaveBytes`; their path fences and report checks apply.
The report reuses `mapRules`: `owner-slot` already covers units and structures;
`script-reference` also checks every typed player parameter in both node arrays.

The window reuses `MapEditor.rows`, `Select` and `detailLayout` for the roster
and its scrollable details. P opens the player filter and returns to all records.
A calls `InspectionPlayers.AddPlayer` and selects the appended row. Up/Down
select players; the record list scrolls with selection. E opens the common
field prompt and Tab cycles the section table's fields. Names use
`InspectionField.Text` and `TextValue`, then `InspectionTextFields.SetTextField`;
numeric fields retain `InspectionEdits.SetField`. `modalLayout` renders entered
UTF-8 through the existing host font encoder. A refusal keeps the selected
field and entered text. Delete uses the common record deletion route; a Player
has no clone, drag or map-pick route. Escape cancels a prompt before leaving the
roster. All keys and help lines belong to `editorKeyRows`; no second mode state
or player form was added. The keys screen's fourth help line reads
`N: new map / U: unit / P / F5`: the installed font leaves no room in 300
pixels to name what P and F5 do.

## Authority and values

Knowledge pin: k305, `c57c885ed78838fa6bfc8bcee47f7ecc3aa0bfa8`.
The claims were read with `knowledge/tools/claim`; the type-5 grammar is
`knowledge/formats/alm/placements.md`.

| Value | Wire bytes | ROM1 authority and grade | As built |
|---|---|---|---|
| Player identity | File-order ordinal plus one; not a stored identity | `ALM-OWN-039`, `ALM-PLAYER-069`: High for owner/ordinal and colour stores; older composed owner chain Medium. `ALM-SAVE-256`: High for named editor stores, Medium for bounded order conclusions | Unit and structure owners and script player parameters address this ordinal |
| Colour | `+0x00`, four bytes | `PAL-LIMIT-017`: High, safe team range 0..15. `PAL-SHADE-012`: High within constructor family, Medium for absence of other aliased stores. `PAL-SHADE-013` and `PAL-RULE-021`: High local sender rules, Medium shipped single-player composition | Mod missions use the authored slot for unit bodies and minimap marks. Loose maps use it for unit bodies; their minimap retains its existing green/red marks. Shipped campaign missions retain the ordinal shade rule. This authoring policy is DIV-3295 |
| Control | `+0x04`, four bytes | `ALM-139`: High four-byte transfer, native I/O Unknown. `SESS-072`: High local transfer, Medium composed identity, native ordering Unknown. `SESS-073` and `SESS-074`: High local matrix/projection/persistence operations, native lifecycle Unknown | The editor offers 0 human and 1 computer. Loading preserves all 32 bits. A unique current Player matching a nonzero owner Slot supplies the whole-word zero/nonzero test to the local readers below; absent or ambiguous identity retains the legacy fallback. Owner-zero handling remains each consumer's existing rule. Control is separate from local command ownership |
| Preserved scalar | `+0x08`, four bytes | `ALM-CPLAYERC-314`: High bounded call/body/count observations, Medium no-reader conclusion; gameplay effect Unknown | No setting or stated gameplay effect. Existing bytes survive. New player value is 5000 |
| Name | `+0x0c`, 32 bytes | `ALM-GRP-020`: High stride/name bounds, Medium diplomacy semantics. `ALM-DLGPLAYERS-287`: High validation arguments, Medium enforcement and overflow inference | At most 32 encoded Windows-1251 bytes; embedded NUL and unencodable text are refused. Decode is bounded to those 32 bytes |
| Directed relation | `+0x2c+2*j`, two bytes | `AI-DIPLO-004`: High bit meanings; `AI-DIPLO-005`: High row loading and forced diagonal, Medium complete value-space inference. `TRIG-DIPLO-019` and `020`: High directed write and low-two-bit script read. `TRIG-DIPLO-021`: Medium author-label interpretation, exact census | 0 neutral, 1 hostile, 2 allied with the ordinary hostility-flip lock. Each edit changes one direction only. The game forces the loaded self-relation to 2 even if a different diagonal is stored |

Far-route budgets and escort idle acquisition use the control predicate in
`MOVE-TERM-003` and `AI-ROUTE-045`. The same lookup supplies the short-reach
mage veto at reacquisition (`AI-GUARD-007`, High), attack-notice stamping and
scan consumption (`AI-GROUPSEE-068`, High for the local routine), spell-damage
victim exclusion (`UNIT-OWNER-009` and `ALM-140`, High for the local zero tests),
and the full-book mage selector's nonzero-control gate (`MAGIC-AI-012`, High
for that gate). The creature slot draw suppresses only a zero-control mage
with reach below 2; a non-mage or a mage with reach at least 2 is not excluded
by zero control (`AI-377`, High for the gate, Medium for its historical
participant meaning). Owner-zero creatures retain their explicit refusal.
Ordinary engage admission is unchanged; the shared attack writer also carries
explicit commands and is not given a blanket control veto (DIV-1518).

Saved-group activity coverage comes from current living on-map nonzero-owner
actors taking the zero-control branch; represented nonzero-control members
rebuild the activity byte (`AI-ACTIVITY-324`, High for the local rebuild).
Owner-zero members retain the previous consumer rule. Native and saved group
decisions use each member's control for their existing no-candidate branches,
including the Stand Ground zero-versus-nonzero split (`AI-517`, High for the
dispatch and arm targets, Medium for full composition). This changes the
predicate, not command protection, the missing native heal branch or complete
original scheduling.

`ALM-CPLAYER-090` (High) distinguishes the game and editor class surfaces;
their matching scalar offsets do not establish matching methods.
`ALM-ROSTERIO-292` is High for the reader/writer windows and installed counts,
Medium for an image-wide absence of normalisation. Arbitrary existing roster
words and name padding are carried until their own field is explicitly edited.

`ALM-DLGPLAYERS-286` is High for the original dialog routing and read bodies,
Medium for the bounded reference census. `ALM-DLGPLAYERS-288` is High for Add,
Delete and owner emission; deleted-unit reload composition is Medium and the
structure reload arm is Unknown. New rows take control 1, scalar 5000,
`New Player N`, own relation 2 and every other relation 1. Existing rows are
unchanged by Add. The original default colour N becomes 15 for player 16;
the original Add body has no 16-player check. The editor caps the roster at
16 and applies the safe colour range (DIV-3296). A runtime roster-count cap
is Unknown. `ALM-DLGPLAYERS-289` is High for the directed row mappings and
Medium for library exchange; the editor restricts relations to its three
offered values because a stricter game validation bound is Unknown.

## Bytes and deletion

Add appends 76 bytes to type 5, increases its payload length by 76 and
increments type-0 `+0x1c`. When type 5 is absent, the same row permits
`Place` to insert its 20-byte section header and first 76-byte player after
type 0, increment file recordCount and set Count5 to 1. Stale absent counts
and uncounted trailing records are preserved outside those framing words.
Name edits replace only the 32-byte name field.
Colour and control edit their own four bytes. A relation edits its own two
bytes. Each accepted edit is one existing history entry; Undo restores its
exact preimage and Redo restores its exact result.

Delete refuses the first player and any player named by a unit owner,
structure owner or `Target_Player` parameter. The error identifies the
record and parameter where applicable. Before applying anything, it walks
every present owner section and both script arrays; a malformed present
section refuses the operation. The deletion then removes one row, updates
its framing and decrements references to later existing players at type-6
`+0x14`, type-4 `+0x0e` and type-3 parameter values at node `+0x4c+4*j`.
Unit and script references use the full word; the current structure route
uses its low 16 bits, and renumbering preserves the upper bits. References
to no existing player remain unchanged.
Parameter types are read at `+0x74+4*j`, including unknown opcodes and every
one of the ten slots. Surviving relation columns shift with the removed
player and the final column becomes zero. All changes share one undo step.
DIV-3294 records this deliberate difference from original editor deletion.

## Proof

Focused model tests independently construct expected byte images for each
mutation, round-trip Undo/Redo, exercise both script arrays, and reject
referenced or first-player deletion. A corrupt roster above 16 can be
reduced without indexing beyond its fixed 16 relation columns.

`TestReleaseMapEditorPlayerEditsChangeOnlyRosterAndFraming` compares a saved
installed map against its existing `SaveBytes` result, names the pre-existing
save differences, and excludes changes outside type 5 and its count/length.
`TestReleaseMapEditorPlayersPlayWithAuthoredColoursAndRelations` loads an
edited map through a mod mission, compares a neutral unedited control,
checks an independently decoded installed palette pixel and live hostile
acquisition. It includes legacy AGS cold restoration and the production SAV
`ExportCurrentSave` / `RestoreOriginal` route. Both routes pass on EN and RU
in `players-final-en.log` and `players-final-ru.log` in the task's notes.

The window's focused UI and host tests pass in `window-focused.log`.
The recorded mode table passes in `window-golden.log` (10.384 seconds).

`TestAuthoredControlFarMovementAndColdRestore` distinguishes control by a
long detour under production simulation steps and repeats it after cold
restore. Before the fix, `human-other-slot` and `computer-self-slot` fail.
`TestAuthoredPlayerControlSurvivesMissionAndColdRestore` fails before the fix
because an authored first-player control 1 becomes 0 on mission start.
`TestAuthoredControlEscortIdleAndColdRestore` exercises the existing escort
no-candidate acquisition branch for control 0, 1 and 0x100.

The added tests in `authoredcontroldamage_test.go` assert the remembered cell,
stamp, 21-scan expiry and exact spell-damage XP. `authoredcontrolacquire_test.go`
asserts reacquisition, full-book choice, the distinct mage/reach slot gate,
retained approach and native no-candidate response. `authoredcontrolgroups_test.go`
asserts activity and saved Stand Ground decisions using the member's control
when its Group belongs to a different Player. `authoredcontrolrestore_test.go`
asserts that resume repair preserves eligible retained casts. These tests use
cold serialization and continuation hashes. The focused candidate run passes
in `control-focused.log` (0.804 seconds). Ordinary `pkg/sim` and storyguard
pass in `control-ordinary-final.log` (16.963 and 4.721 seconds); the touched
control packages' vet exits 0 in `control-vet.status`. Behavioral loss control
reproduces failures on `ec94605a4d4dcadc22ed8f955c52f9ce6c5ea05b` in
`control-loss-validated.log`. These receipts are separate from the earlier
EN/RU player-release receipts above.

## Open boundaries

The scalar's gameplay effect, a native enforced roster-count bound and native
arbitrary-word control lifecycle remain Unknown at the grades above.
Control does not add multiple local command participants or a network lobby.
Local command admission, selection, manual-cast policy and existing command
protection retain their ownership tests. Native duplicate-slot Player
association retains its prior fallback rather than inventing an identity
from an ordinal. These local predicates do not establish every control
consumer or original AI scheduling. Full-book target, re-entry and id-28 debt
remains DIV-3268; activity population and session-override debt remains DIV-351;
group-order, ordinary engage and healing debt remains DIV-1518 and DIV-1581.
Creating a roster in a map without type 5 is byte-preserving outside its
new section and count words. Native playability of such newly populated
installed maps is not established by the synthetic creation tests.

EN and RU focused release receipts are PASS.
For each installed `scenario/10.alm`,
Save As grows 67,554 bytes to 67,630; the pre-existing SaveForm changes zero
bytes. The roster edit changes no section besides type 5 and the type-0 count.
The mod mission's computer owner 3 uses colour 9. Its installed palette index
10 produces RGBA (33,54,37,255), compared with ordinal colour 3's
(78,68,47,255). Legacy AGS and production SAV cold restoration retain that
pixel, owner 3, control 1 and the directed relation: 3 to 1 is hostile,
1 to 3 remains neutral. The hostile computer acquires owner 1 at tick 7;
the neutral control has no target.

`TestReleaseModMapRuleCensusRefusesNoInstalledMap` checks 28 maps in each
install with zero errors. EN reports 1 `drop-cell`, 4 `mod-message-text` and
5 `unit-terrain` warnings; RU reports 1, 3 and 5 respectively.

The baseline-compatible `TestPlayerLossControls` probe fails all eight
subtests on exact base `77c9e77506bce11ae5bcc9718c0444522dc61c3a` and passes
all eight on the candidate model. The subtests cover add, rename, colour,
control, relation, deletion, first-player refusal and undo. The probe and
both receipts are in the task's notes directory.

Normal selected-player and name-prompt panels were rendered headlessly on EN
and RU into the notes directory and visually checked.

Final full-tree and release receipts are recorded in the lane notes
`PROGRESS.md`.

# Extended map scripts in the map editor

## Intent

Let a map maker write the extended map script in the map editor. This is
step 3 of the extended map scripts: the editor opens `.alm` and `.almx`
maps, edits a trigger of the extended model, asks once before an original
map becomes extended, saves an extended map as `.almx`, and offers to save
one that fits the original records again as `.alm`. An original map edited
within its records saves byte for byte as before.

## Authority

Owner decisions for the extended map scripts (`docs/1608/story.md`): the
`.almx` is the ALM layout plus the Againrom script record; the switch comes
at the first construct the original record cannot hold (a fourth pair, OR,
a fifth action, a variable past the free registers, a trigger past 1000);
OR of AND groups; 4096 variables, 10000 triggers, 64 conditions and 64
actions per trigger; extended variables are their own array.

Owner direction for this step: the first extension asks once, then the map
uses the extended script machine. An extended map whose script fits the
original record again may be saved as `.alm` when the editor offers it.
Owner rulings: one builder per kind; the editor is Againrom's own; the 1608
review's note 1 is settled by one rule. Rows DIV-4615 and DIV-4616 in
`docs/divergences/mods-maps.md`; DIV-4603 holds the loose `.alm` rule;
DIV-3394 points at DIV-4615 for an original map's limits. The modder's form
is `docs/MODS.md`, "The map editor" under "Extended map scripts (.almx)".

## As-built behaviour

### Kinds and builders

| Kind | Builder | Where |
| --- | --- | --- |
| extended record codec | `alm.DecodeScriptRecord`, `alm.EncodeScriptRecord`; one trigger's bytes are `DecodeScriptRecordTrigger`, `EncodeScriptRecordTrigger`, which the whole-record pair now calls; `FrameScriptRecord` frames the encoder's payload | `pkg/formats/alm/scriptrecord.go` |
| script limits | `scriptprofile.First`, `Extended`; `Of(extended)` picks one; `GroupKey` names a condition's group field | `pkg/scriptprofile/profile.go` |
| map model sections | `mapedit` `section`: on a map carrying the record, `KindTrigger` is the record's trigger array (`recordTriggers`) through the one section machinery | `pkg/mapedit/scriptform.go`, `section.go`, `editor.go` |
| 184-byte trigger writer | `mapedit.NewScriptTrigger`, `SetTriggerSlots`, `ScriptLimitError` | `pkg/mapedit/script.go` |
| extended trigger writer | `(*Editor).NewTrigger`, `setRecordTrigger`, through `EncodeScriptRecordTrigger` | `pkg/mapedit/scriptform.go` |
| form moves | `(*Editor).ExtendScript`, `FoldScript` (fold writes through `NewScriptTrigger`) | `pkg/mapedit/scriptform.go` |
| script host | `mapInspectionEdits.change` turns a lifted `ScriptLimitError` on an original map into `*ui.ScriptFormNeeded` (`formNeeded`); host ops `ExtendScript`, `FoldScript`; `setTriggerSlots`; `raiseVariables` | `pkg/game/mapinspectionscriptform.go`, `mapinspectionedit.go` |
| editor's map open and save | `writeEditedMap`; `scriptSaveForm` decides `.almx`, the fold question or the refusal; `extendedName` refuses the record under a non-`.almx` name in every map writer (editor save, text pages, mod save); `refuseExtendedMap` removed | `pkg/game/mapinspectionedit.go`, `mapinspectionscriptform.go`, `maptextpages.go`, `modmapsave.go` |
| editor's script model | `InspectionTrigger` slices `Left`, `Right`, `Cmp`, `Acts`, `Group`; `InspectionScriptModel.Profile` | `pkg/ui/mapeditorscriptmodel.go` |
| trigger window | the details view: rows per condition and action held, Add condition pair, Add OR group, Add action, Remove | `pkg/ui/mapeditortrigger.go` |
| trigger wizard | its rows carry the group; "Or when" starts a group | `pkg/ui/mapeditorwizard.go` |
| confirmation dialog | the editor's `questionSpec` popup, prompt kind `promptScriptForm`; `askScriptForm` holds the edit and runs it again | `pkg/ui/mapeditorscriptform.go` |
| map list load gate | `CheckModMap`, unchanged | `pkg/game/mapcheck.go` |

No second builder was added: one trigger model, one record codec, one save
path, one limits table, one dialog builder. No second-game path is reached;
the record is read only in the first-game dialect.

### The question

On an original map, an edit the 184-byte record or the original counts
cannot hold is held. The edits are a fourth condition pair, an OR group, a
fifth action, a variable at 100 or above, and a trigger past 1000. They come
from the trigger window, the trigger wizard, a node's window, a New button
of the Scenario palette, a drawn area box, or a condition's Add to trigger.
The host answers `*ui.ScriptFormNeeded`; the editor opens the question in
EN or RU: "Make this an extended map?", "The original trigger record cannot
hold this edit. Applied, the map becomes an extended map: it is saved as an
.almx file, and the original game will not open it." Buttons: "Extend and
apply", "Cancel". Cancel changes nothing. Yes runs `ExtendScript` as one
history entry, then the held edit. After a Yes the question is not asked
again for that opened map.

`ExtendScript` moves the type-7 triggers into the record in map order, with
the hundred registers as its first variables; the type-7 record keeps the
nodes. Each trigger fires as the original fires it: a pair with an empty
side and an empty action slot go; a trigger the original never fires keeps
one condition that never holds (a check compared with itself by "differs").

### The trigger window on an extended map

A trigger shows its conditions in groups and every action it holds. Add
condition pair adds a condition to the last group; Add OR group starts a
group; Remove on a condition's comparison row or on an action's row takes it
out. A trigger holds up to 64 conditions and 64 actions. The variable
pickers and the variable table offer variables 0..4095; the variable count
rises with the highest variable a node addresses. The heading
"Conditions: all must hold" stays above the groups.

### Save

- An extended map saves through `writeEditedMap` as `.almx`, the record
  written by `alm.EncodeScriptRecord` and no type-7 trigger.
- Save over the map's own `.alm` writes `<name>.almx` beside it; the `.alm`
  stays as it was.
- Save As to an `.alm` name of a map that does not fit is refused with a
  message and writes nothing.
- The `.alm` offer: when the script fits the original records again (one
  group per trigger, at most 3 pairs and 4 actions, variables below 100 and
  none in a check's register, at most 1000 triggers and 100 checks), Save or
  Save As to an `.alm` name asks "Save as an original map?" with "Save as
  .alm" and "Cancel". Yes runs `FoldScript`, each trigger written by
  `mapedit.NewScriptTrigger`, the record removed, as its own history entry,
  then saves. Cancel writes nothing.
- An original map with no extension saves byte for byte as at the base.

`NewScriptTrigger` encodes decoded UTF-8 names through `ScriptLabelText`
as Windows-1251, with fewer than 64 encoded bytes; an unrepresentable or
longer name refuses the write. Cyrillic names survive the `.almx` to `.alm`
round trip. The trigger window and wizard already write authored names
through that codec. The node window, palette New buttons, area box and
Add to trigger keep names or create ASCII names; no pre-existing corrupt
Cyrillic-name route was found among those callers.

The 184-byte writer alone handles a model with no condition pairs and at
least one nonzero action: it writes first pair `(1, 0, ==)`. `FoldScript`
passes the shown model unchanged to that writer. `TRIG-BIND-010` establishes
that the original drops a trigger whose first left ID is zero and binds
only pairs with both IDs nonzero; `TRIG-CMP-006` establishes that evaluation
starts true. Thus the written trigger runs without a comparison, subject
to its once flag. Previously a direct construction of that model wrote
`(0, 0, ==)` and the original dropped it. Existing records and explicit
empty pair slots are preserved; editing slots in an existing original
trigger still uses `SetTriggerSlots`.
Drop-zone deletion recognizes the conditionless pair as well as legacy
empty pairs. A trigger with a comparison keeps its record and loses only
the deleted zone's action reference.

### A loose `.alm` carrying the record

The map list reads a loose map by its bytes: a loose `.alm` that carries the
record lists and plays as an `.almx` (DIV-4603). A mod mission's map named
`.alm` that carries it stays refused by `CheckModMap`. The rule is stated,
not changed: a refusal at the map list would be a second gate on a non-mod
path, and no Againrom writer makes such a file.

### `scriptprofile.First` sites kept

- `mapedit/arm.go` `var script = scriptprofile.First`: an arm writes into
  the type-7 trigger layout; on an extended map the section machinery
  routes it to the record.
- `triggerWords` and `NewScriptTrigger`: the 184-byte writer is the original
  record's only.
- `FoldScript`: folding targets the original record.
- `errChecksFull`: the 100 registers shared with the checks exist on an
  original map only.
- `scriptRegisterAllowed`'s original branch: the free registers of an
  original map.
- `mapinspectiontriggers` `RecordPairs`, `RecordActions`: the original
  layout's offsets.
- `currentscriptprogram.go`: the runtime session block, not the editor.

## Proof

- Census at base `963d9ea20` and on the candidate, over every map of the
  EN (38) and RU (34) first-game installs, read through the engine's
  loaders: per map the editor's open, its script listing, the trigger
  popups and the digest of a save without edits. Open, listing and save
  digests are byte-equal. The popup lines differ on 18 maps in each
  language by 158 added empty add-row lines and no other line: they fall in
  the 77 triggers of 421 (EN) that hold three pairs or four actions, which
  now show Add condition pair and Add action. The instrument was a
  temporary test, not committed.
- `pkg/mapedit/scriptform_test.go`: ExtendScript as one edit and one undo;
  extended triggers through the section machinery; FoldScript equals
  `NewScriptTrigger`'s bytes and refuses what the original cannot hold; an
  original map's writes name the limit the record lifts.
- `pkg/game/mapeditorscriptform_test.go`, on a synthetic map with no install
  bytes: each edit past the record asks and changes nothing until extended;
  the trigger window's fourth pair asks (EN and RU), Cancel changes nothing,
  Yes extends, then an OR group, a fifth action and a variable past 100 ask
  nothing, and Save writes an `.almx` that reopens as the saved model and
  compiles to the extended program; Save As to `.alm` refused when the map
  does not fit and asked when it fits, with type-7 records equal to
  `NewScriptTrigger`'s; the wizard's OR group asks; an original map saves
  without the record, as the base writes it.
- `TestScriptRoundTripPreservesTriggerName` carries the review's ASCII and
  Cyrillic save/reopen/fold/save probe into the story tests.
  `TestConditionlessFoldUsesTriggerWriter` asserts byte equality with the
  direct writer for zero conditions and four actions.
  `TestNewScriptTriggerEncodesTheLabel` checks the Windows-1251 bytes,
  a 63-byte Cyrillic name and refusals for 64 bytes, unrepresentable text
  and embedded NUL. The writer layout test pins the conditionless pair and
  preservation of explicit empty slots.
- `pkg/game/mapeditorareaedit_test.go`, `mapeditorscriptadd_test.go`: a
  full original trigger asks to extend the map from the area box and from
  Add to trigger; Yes puts the condition in its fourth pair.
- `pkg/game/extendedscript_test.go`: `TestMapEditorOpensAnExtendedScriptMap`
  replaces the refusal test.
- The map editor modes walk ran once, ordinary; no golden changed.

## Open debt

- The EN and RU release tests of the script window ran in no lane.
- A mod package names its map `mission/map.alm`, so `CheckModMap` refuses an
  extended map saved into a mod or test-played as a mod mission (inference
  from the code; not run).
- ExtendScript drops a trigger's 64 unnamed bytes the record has no place
  for, and refuses a node id past 5000.
- The heading "Conditions: all must hold" stays above OR groups.
- The 1608 changelog line saying the editor does not open such a map is
  followed, not rewritten.

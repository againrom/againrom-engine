# One builder for map scripts

## Intent

Make the map script one builder per kind before map scripts are extended
for mods: one walk of the type-7 framing, one writer of a trigger record,
and one profile of the script limits per game. No player change: every
shipped map decodes, compiles and lays out as before, every editor
operation writes the same bytes, and the World, its hash and its byte form
are unchanged. Architecture work counts zero toward a release. No DIV row
is reserved or changed.

## Authority

Owner rulings: one builder per kind; extended map scripts are taken with
the five recommended decisions, of which this is step 1 (the builders) and
step 3 later turns the typed limit error into the owner's confirmation.
The limits are the engine's values at base `30dd2af67`: ALM-TRIG-047 (the
184-byte trigger record, three pairs, four actions), TRIG-CMP-006 (codes
0..5), TRIG-STORE-002 (100 registers, 1000 latches), MISSION-SLOT-008 (checks
and variables share the register array), TRIG-BIND-010 and R2-SESSION-019
(the second game's 1000 registers).

## As-built behaviour

### Kinds and builders

| Kind | Builder | Where |
| --- | --- | --- |
| type-7 walk | `alm.FrameScript`, `frameScript`; `Triggers.Decode` decodes through it | `pkg/formats/alm/script.go` |
| trigger record writer | `mapedit.NewScriptTrigger`; field setter `(*Editor).SetTriggerSlots` | `pkg/mapedit/script.go` |
| node record writer | `mapedit.NewScriptNode`, unchanged | `pkg/mapedit/script.go` |
| script limits | `scriptprofile.First`, `scriptprofile.Second` and the constants | `pkg/scriptprofile/profile.go` |
| runtime compile | `mapload.compileScriptFrom`, unchanged in shape | `pkg/mapload/script.go` |
| editor trigger fields | `triggerFields` table; ui `scriptFields` keys from `scriptprofile` | `pkg/mapedit/script.go`, `pkg/ui/mapeditortrigger.go` |

### The walk

`FrameScript(payload)` locates the three counted arrays (actions and
conditions of 796-byte nodes, triggers of 184-byte records) as payload
offsets: each array's count word, first record, count and record size. A
payload the arrays do not tile exactly is a `*ScriptFrameError` naming the
array and the shortfall. `Map.Script` and `Triggers.Decode` read the arrays
at its offsets; their error text is the base's. `mapload.DropCells` reads
the decoded actions through `Triggers.Decode` instead of its own walk of
`Body`. The map editor's `scriptLayout` takes the frame's offsets and maps
the typed error back to its own base wording. `cmd/almtool` had no walk of
its own: it prints `EntryCount` and `len(Body)` and decodes through
`Map.Script`.

### The trigger value and its writer

`alm.ScriptTrigger` holds `Left`, `Right`, `Cmp` and `Acts` as slices.
Decoding fills every slot of the record (three pairs, four actions);
`Pairs`, `Pair(k)` and `Act(k)` read a value that spells fewer, zero past
its slices.

`NewScriptTrigger(t)` builds a zero record with the name, the slot words
and the fire-once word, through the offsets of the trigger field table.
`SetTriggerSlots(i, t)` writes `t`'s slot words and fire-once word over
trigger `i` as one `SetFields` edit, only the words that change; the name
and the 64 unnamed bytes at +0x40 stay. A word the record already holds is
not checked, so a shipped code outside 0..5 survives a write of the other
slots. Both refuse a value the record cannot hold with a
`*mapedit.ScriptLimitError{Limit, Max, Got}` raised in `triggerWords`:
`LimitPairs` (more than 3 pairs), `LimitActions` (more than 4 actions),
`LimitCompare` (a code above 5), `LimitID` (an id above 5000). Nothing is
written and no history entry is made.

Writers moved onto it: `mapedit.AddArm`, `mapedit.AddDropZone`,
`game.addScriptRecord` (the new empty trigger), `game.addBoxArea` (the
box pair), `game.writeWizardTrigger` (new and rewritten wizard triggers)
and the wizard's drop-zone delete. The ui's field-key writes (the trigger
window and `listedToTrigger`) reach `mapedit.SetFields` over the same
trigger field table with keys from `scriptprofile`; `pkg/ui` may not import
`mapedit`.

### The profile

`pkg/scriptprofile` is a leaf package beside the schema rather than inside
`pkg/formats/alm`: `pkg/sim` may not import a format package (the
determinism wall) and `pkg/ui` may not import `alm` (the DAG). `alm`'s
`ScriptPairs` and `ScriptActionRefs` are its constants.

| Value | First game | Second game |
| --- | --- | --- |
| pairs per trigger | 3 | 3 |
| actions per trigger | 4 | 4 |
| highest comparison code | 5 | 5 |
| registers | 100 | 1000 |
| latches | 1000 | 1000 |
| triggers | 1000 | 1000 |
| highest node id | 5000 | 5000 |

The trigger field keys (`left0`, `cmp0`, `right0`, `action0`, ...) are its
`LeftKey`, `CmpKey`, `RightKey` and `ActionKey`. Sites that read it: `alm`
schema; `sim` (`scriptRegisters`, `scriptLatches`, the
`ScriptTrigger.Pairs` and `Instants` arrays); `mapedit` (`arm.go`'s
`script`, the node id cap, the trigger field table, `AddDropZone`'s trigger
cap); `mapload.compileScriptFrom` (pair and action loops over the sim
arrays); `game` (`addScriptRecord`, `scriptRegisterAllowed`,
`writeWizardTrigger`, `wizardNode`, `addBoxArea`, the trigger inspection,
`mapinspectionworkspace.go`, `maptextpages.go`, `currentscriptprogram.go`,
`errChecksFull`); `ui` (`InspectionTrigger` arrays, `objectEdit.added`, the
trigger window's loops and field order, the wizard's loops, refusal,
comparison choices and `mirroredCmps`, the Add window's comparison bound
and `listedToTrigger`). The sim arrays keep their sizes, so the World, its
hash and `scriptbinary.go` are unchanged. The runtime World still holds 100
registers for both games.

### Guard

`internal/archtest/scriptframe.go` `CheckScriptFraming`, over the
type-checked module: outside `pkg/formats/alm` no production file reads
`alm.Triggers.Body` except as the argument of `len`, and outside
`pkg/formats/alm` and `pkg/mapedit` none names a trigger slot offset
(`ScriptTriggerPairs`, `ScriptTriggerActions`, `ScriptTriggerCompare`,
`ScriptTriggerOnce`). No static check states the profile: a bare 3, 4 or 5
has too many other meanings in the listed files.

## Proof

- Census over every map of the EN, RU and second-game RU installs (155:
  EN 38, RU 34, second-game RU 83), read through `vfs`, `alm.Open` or
  `alm.OpenROM2`, `mapload.CompileScriptFrom` or `CompileROM2ScriptFrom`,
  `mapload.DropCells` and `mapedit.New`: per map the decoded script, the
  compiled script, the drop cells, the editor's three array layouts and the
  drop zones. Base and candidate outputs are byte-equal. 154 maps carry a
  script; the editor opens the 71 first-game maps with one (one RU map has
  no type-7 record).
- The same census rebuilt each shipped trigger through `NewScriptTrigger`
  on the 72 maps the editor opens: 717 records equal their bytes from +0x80
  to the end, none differ, and 122 are refused with the typed error because
  they hold a code outside 0..5 (114) or an id 0xffffffff (8). The highest
  shipped node id is 124 (actions) and 74 (conditions).
- `pkg/game/scripttriggerwrite_test.go` pins every trigger writer's bytes
  and its alm decode against digests taken at the base: four arms, a drop
  zone, a new record, a box area on a trigger and on a new one, three
  wizard writes (one keeping code 0xffffffff) and a field set.
- `pkg/mapedit/scripttrigger_test.go`: the record layout by hand, a decoded
  trigger rebuilt to its record, the typed errors for 4 pairs, 4 codes, 5
  actions, code 6 and ids 5001 (nothing written, no history), and a held
  code kept across a slot write and undone.
- `pkg/formats/alm/script_test.go`: the frame's offsets and its error
  wording.
- Release witnesses for EN and RU (first game, `AGAINROM_ASSETS`):
  `TestReleaseTriggerWizardOpensShippedTriggers`,
  `TestReleaseDropZoneMatchesTheShippedShape`,
  `TestReleaseMapEditorTriggerPopupWritesTheFieldCallBytes`,
  `TestReleaseMapEditorTriggerAreaCensus`,
  `TestReleaseEveryInstalledMapListsItsScriptArmsAndWritesItsBytes`,
  `TestReleaseScriptUnbuiltNodeCensus` in `pkg/game`. Not run in the lane.

## Open debt

- The generic `SetFields` route over the trigger field table refuses an
  over-limit value with the field table's own message, not
  `ScriptLimitError`; its limits come from the profile.
- `AddDropZone` now refuses an action id above 5000, which the base wrote;
  no shipped map comes near it.
- `scriptprofile.Second` is data only: the second game's runtime World
  holds 100 registers, and its session's 1000 live in `pkg/formats/sav`.
- Game code that ranges over `Left` and indexes `Right` and `Cmp` relies on
  a decoded trigger's full slots.

# Quality-of-life keys

## Intent

Three always-on owner keys on the mission map: W takes back the weapon a unit
held before, Ctrl+A turns autocast on or off for the whole selection, Home
resets the zoom and End resets the game speed. The F1 help lists them in
English and Russian. Every key acts only through existing commands: the World
and the SAV after the keys equal those after the same ordinary orders. Base:
public main `e44519c6`; knowledge pin unchanged (k238).

## Authority

Owner direction (the brief): a weapon swap, a selection-wide Ctrl+A, a zoom
reset and a speed reset, each on a key the original does not use, listed in
the F1 help. No ROM1 behaviour is claimed. The original's key surface is read
from `AI-KEY-125` (High, the complete default mission key surface), `MENU-057`
(High, the W arm) and `MENU-078` (High, Home and End in the help text).
Divergence rows DIV-2948..2951 in `docs/divergences/client-input-and-keys.md`
record each difference.

## Keys

| Key | Action | Why the original does not use it | Holder |
|---|---|---|---|
| W, no Ctrl, no Alt | weapon swap | `AI-KEY-125` lists W only as Ctrl+W; `MENU-057` shows the W arm skips everything while the Ctrl latch is clear. Ctrl+W stays the engine's retreat key. | `readAppInput` field `WeaponSwap`; `Viewer.swapWeapons` (`pkg/ui/qolkeys.go`); `mapWorld.swapWeapons` (`pkg/game/weaponswap.go`) |
| Ctrl+A | autocast for the selection | `AI-KEY-125` names no Ctrl+A on the map. The engine already bound Ctrl+A to the primary unit's autocast; the key keeps its binding and widens to the selection. | `Viewer.toggleAutocast` (`pkg/ui/spellbook.go`) |
| Home, no Ctrl or Shift | zoom reset | `AI-KEY-125` does not name Home on the map; inside the help text Home does nothing (`MENU-078`). Shift+Home stays the text-field key. | `readAppInput` field `ZoomReset`; `Viewer.resetZoom` (`pkg/ui/qolkeys.go`) |
| End, no Ctrl or Shift | speed reset | As Home (`AI-KEY-125`, `MENU-078`). Shift+End stays the text-field key. | `readAppInput` field `SpeedReset`; `flow.resetCadence` (`pkg/ui/qolkeys.go`) |

V was the first choice for the swap. It is excluded: the engine pins V
unbound since the owner freed it (`TestMissionVHasNoBinding`). Alt+W stays
in the debug letter range; Alt suppresses the swap.

## As built

### Weapon swap

The game side keeps, per persistent party member whose inventory the player
can open (not hired, a class that composes a figure), the pair it held at the
last observation and the pair it held before slot 1 last changed from a
non-empty weapon. Items match by `sim.ItemEqual`. The observation runs after
each tick and before each swap; it reads the World and writes nothing.

W names every living selected unit the local player owns. A unit whose
previous weapon is in its own pack starts a swap. The swap is one ordinary
order per applied tick. A step is queued only after the unit's previous step
has applied and while no earlier inventory order of that unit is pending
(equip, unequip, drop, pickup, book, potion or scroll), and its container
index is then read, by item equality, from the pack the World holds. The
index therefore names the remembered item when the order applies: a queued
order earlier in the same tick cannot shift another item under it. When an
earlier order removed the remembered weapon the swap ends with no order, as
for a unit with no previous weapon. The steps:

1. The previous weapon goes to slot 1 (`sim.EquipDisplacing`). The current
   shield goes to the pack unless it is the previous shield and the weapon
   leaves a hand free.
2. The previous shield goes to slot 2 (`sim.Equip`) when it is in the pack
   and the weapon leaves a hand free; when the previous pair had no shield,
   slot 2 is emptied (`sim.Unequip`).

Pairing rule: the shield worn with a weapon comes back with that weapon. A
two-handed weapon or a bow (`mapload.WeaponAllowsShield` false) is a slot-1
weapon worn with slot 2 empty, so swapping to it sends the shield to the pack
and swapping back wears the shield again. The current pair becomes the
previous pair, so W twice returns to the start.

`rearm` and `refreshAppearance` follow only the inventory subject. Before
each tick the driver records the worn items of every party member a pending
equip, unequip or worn-item drop order names; after the tick each recorded
unit whose worn items changed and that is not the subject becomes the subject
for one forced pass of both, and the player's subject is restored. The result
equals the equip from that unit's own inventory, World hash for World hash.
The rule reads only the pending orders and the World, which the SAV holds, so
a SAV written before a queued order applies reopens to the same continuation.
It covers every pending order, not only the swap's: an inventory equip queued
while paused and followed by a selection change before the tick now re-derives
the equipped unit as well.

The memory and the unfinished swaps are App state. They live on the running
`mapWorld` (`weaponSwap`, ruled cosmetic in FR-5 of
`docs/0143-save-and-load/spec.md`), are not hashed, are not saved, and are lost
at LOAD and at the next mission; no hashed value reads them. A step already
queued is an ordinary pending order and rides in the SAV. The steps a LOAD
loses are those not yet queued: the shield step of a swap whose weapon step is
queued or applied, and a step waiting on an earlier inventory order. Such a
unit stays in the ordinary state the applied orders left: the previous weapon
in slot 1 and slot 2 holding what the weapon step left there.

### Ctrl+A

`MapEntity` carries each unit's `AutoSpell`. Ctrl+A takes the living
selected units the local player owns. With a spell selected in the book, the
casters are those that know it; when any of them is not autocasting it, each
is set to it, otherwise each is cleared. With no spell selected the key clears
every caster when all of them autocast some spell and does nothing otherwise.
Each change is the autocast command the spell cell's right-click sends.

### Home and End

Home returns the zoom to the zoom a mission opens at (`defaultMapZoom`, the
one `applyStartView` uses) and keeps the view's centre to within the camera's
one-cell clamp. End sets the default cadence rung, leaves the unlimited loop
with its phase reset, stores the rung as the speed preference and posts the
speed line. Neither acts under a popup. Neither changes World state.

### F1 help

The help body is the install's `text/help.txt`, then five engine words in the
install's language (`help.keys`, `help.swap`, `help.autocast`, `help.zoom`,
`help.speed`), each converted to the install font's alphabet and joined by
the install text's own line break. An install with no help text opens no
panel, as before.

## Scope: both games

ROM2 missions run on the same `mapWorld`, viewer and key reader, and ROM2
reads `help.txt` through the same path, so the keys and the F1 lines reach
both games. The CHANGELOG lines are `[BASE]`. No ROM2 claim describes the
second game's key map; the ROM2 collision check is open debt.

## Proof

Focused tests, each with its loss control:

| Test | Loss control |
|---|---|
| `TestWeaponSwapRestoresThePreviousPair` (three pairings) | a swap before any weapon change queues nothing |
| `TestWeaponSwapNeedsThePreviousWeaponInThePack` | the same unit with the weapon still packed swaps |
| `TestWeaponSwapOfSeveralUnitsMatchesTheInventoryEquip`: the swap, the inventory equip, the same orders queued bare and the inventory equip with the selection moved before the tick give one World | the same orders stepped on the World alone, without the re-derivation, leave a different World |
| `TestWeaponSwapQueuedBeforeASaveAppliesAfterTheLoad`: World `0x1483130294306173`, damage 10/10, before and after a reopen | the same order stepped on the World alone: `0x7b756756671c2364`, damage 14/12 |
| `TestWeaponSwapWaitsForAQueuedDrop`: W after a queued drop of the remembered weapon queues nothing and slot 1 keeps the sword | a queued drop of another element: W still takes the remembered weapon back |
| `TestWeaponSwapMemoryIsNotCarriedIntoANewMission` | the running mission does queue the swap |
| `TestCtrlATogglesAutocastForTheWholeSelection` (seven cases) | the primary-only toggle sent one command |
| `TestCtrlALeavesADeadUnitAlone` | the same selection alive takes the command |
| `TestWeaponSwapKeyNamesTheOwnedSelection` | a frame without the key, and Alt+W, name nothing |
| `TestHomeRestoresTheOpeningZoomAboutTheCentre` | the wheel left another zoom |
| `TestEndRestoresTheDefaultGameSpeed` | + and the unlimited loop left another rung; the help panel refuses End |
| `TestHelpBodyListsTheAddedKeys` | an empty install text opens no panel |
| `TestQualityOfLifeKeyBindings` | each key is bound without Ctrl |

Installed witness `TestReleaseQualityOfLifeKeysSaveOnlyOrdinaryOrders`, EN and
RU. Mission 10 with the default mage hero, who wears the installed Fire Arrow
staff and carries a plain staff of the same code. One run wears the plain
staff from the pack, then, during the player pause, takes the Fire Arrow staff
back with W, selects Fire Arrow and presses Ctrl+A, moves the zoom with two
wheel notches and the speed with two presses of +, and resets both with Home
and End. A second run gives the same orders through a pack double-click and a
right-click on the spell cell. Results, EN and RU alike: both Worlds are byte
for byte equal (World hash `0xddd53554681fc5b4` at tick 51); autocast goes
from none to Fire Arrow; the weapon spell returns to Fire Arrow; zoom and
period return to the opening values. The two SAVs are byte for byte equal
(EN 218469 bytes, RU 217958 bytes); the SAV written before the orders
differs (loss control). A cold LOAD gives the saved World hash and equals the
uninterrupted run on each of 64 further ticks. W after LOAD takes nothing
back; W on the running mission takes the plain staff back (loss control).

Release tests, RU then EN, one root at a time, run directly with
`AGAINROM_ASSETS`, all passing on both: the help family
(`TestReleaseHelpPanelOverTheMap`, `TestReleaseHelpKeyDoesNothingInTown`,
`TestReleaseHelpScrollBarDragAndWheel`, `TestReleaseCastKeyOnTheMap`), the
mission keyboard (`TestReleaseMapKeyboardRemainder`,
`TestReleaseSettingKeysPostInstalledStateLines`,
`TestReleaseFunctionKeysSaveLoadAndDiplomacy`,
`TestReleaseDialogueKeysOnEveryPage`, `TestReleaseActivePauseOrderedSAVAndResume`),
the inventory (`TestReleasePackItemDraggedOntoThePaneIsWornInBothModes`,
`TestReleaseShieldWornAloneAndOverTwoHandedWeaponsThroughApp`,
`TestReleaseHeroSwingClassRederivesOnEquipmentChange`,
`TestReleaseWornBodyItemRaisesThePanelBody`,
`TestReleaseHiredHumanMercenaryHasAReadOnlyMissionDoll`,
`TestReleaseModLayersInTheMissionInventory`, `TestReleaseEqualBowsShareOnePackCell`),
autocast (`TestReleaseAMageKeepsItsOrderedEnemyAndResumesAfterASelfCast`) and
the witness. The F1 panel now wraps to 42 lines on EN (35 install, 7 added)
and 54 on RU (48 and 6); `TestReleaseHelpPanelOverTheMap` and the scroll-bar
style witness state both parts.

Installed witness `TestReleaseWeaponSwapQueuedBeforeASaveAppliesAfterTheLoad`,
EN and RU. Mission 10 with a generated companion that wears the short sword
(`0x0103`) and then the two-handed sword (`0x1106`) from its pack. With the
primary hero the inventory subject, the whole selection takes W and the game
is saved before the companion's order applies. The cold LOAD holds the same
World and the same one pending order; both runs resume through the App and
stay equal on each of 8 ticks, World `0x7e74e4a5cbbb1415` at tick 50, the
companion back on the short sword with damage 10/6 (SAV EN 220798 bytes, RU
220830). Loss control: the same order stepped on the reopened World without
the re-derivation leaves damage 11/8 and a different World.

Installed witness `TestReleaseWeaponSwapAfterAQueuedDropEquipsNothingElse`, EN
and RU. Mission 10, the hero wears the short sword and carries the remembered
two-handed sword ahead of the bow (`0x8134`). While paused the inventory drops
the two-handed sword and W is pressed. Slot 1 keeps the short sword and the
World equals the drop-only run (loss control), `0xeb22abc26b75c36d` at tick
48. A run with W and no drop wears the two-handed sword.

No golden, trace or digest moves.

## Open debt

- A hired unit has no swap; its inventory is read-only.
- A swap's steps not yet queued and the previous-weapon memory are lost at
  LOAD, by owner direction.
- The second game's key map has no claim; whether ROM2 uses W, Home or End
  on its map is Unknown.

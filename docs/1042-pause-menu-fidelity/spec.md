# Story `1042` — as-built specification

This is the canonical specification at landing. Claim provenance, the five-behaviour limit and the
review stopping condition are in `contract.md`.

## Root surfaces

**FR-1.** `ScreenGameMenu` has three root populations:

| Session | Rows, in order | Gates |
|---|---|---|
| Campaign mission | Save, Load, Game Options, Sound Options, Quest Objectives, End Quest, Return | Save needs a save writer; Load needs at least one listed save; Sound needs an opened audio player; the other four are enabled |
| Standalone map | Save, Diplomacy, Game Options, Sound Options, Quest Objectives, End Quest, Return | Save and Quest are disabled; Sound needs an opened audio player; the other four are enabled |
| Town | Save, Load, Sound Options, Abort Game, Return | all five are enabled |

Campaign is the implementation meaning assigned to researched mode 2. A production mission viewer
installs `Campaign:true`; a loose-map viewer installs `Campaign:false`. A hand-built viewer with no
context seam retains the campaign root for compatibility with the pre-story client tests.

**FR-2.** Save, Load, Diplomacy, Game Options, Sound Options, Quest Objectives, End Quest, Abort Game,
Return, Change Map, Victory, Exit to Main Menu and Exit to Windows use the install's `dialogs.txt`
labels. Diplomacy is table-local index `0x4c`; Abort Game is `0x4d`; the four confirmation labels are
`0x2a` through `0x2d`; the other seven are `0x22` through `0x28`. The first character after a single
`~` is the accelerator; when no mark exists the row's owner-authored fallback is used. In
particular, Change Map's unmarked EN label uses authored `C`; no cited claim decodes that immediate.
The install selector applies the existing CP866 fold. A disabled row remains drawn and focusable but cannot
dispatch through Enter, pointer or accelerator.

## Destinations

**FR-3.** Game Options opens a two-row nested page. `Tips: On/Off` toggles the existing inverse
`FrontEnd.TipsOff` value through `SetTipsOff`; Return goes to the root. The existing option store is
therefore the only persistence writer. Sound Options shows mute and master volume, permits toggling
mute and moving volume in steps of 25 within 0..100, and applies the new settings to subsequent
plays on the opened device. Sound state lives on this `FrontEnd` and is neither an options-file nor
a save field. An unavailable callback produces an informational row and a usable Return.

**FR-4.** Quest Objectives shows the current ALM map description, wrapped into literal rows, or
`NO QUEST OBJECTIVE IS RECORDED FOR THIS MISSION.` when it is empty. Diplomacy enumerates non-zero,
non-player owner slots currently present in the world, ascending. The relation from `sim.SelfSlot`
to each is shown as Hostile when bit 0 is set, Allied when bit 1 alone is set, and Neutral otherwise.
The projection is read when the root opens. It never writes a relation or holds a world pointer in
the client.

**FR-5.** Return on the root and Escape on the root restore the screen that opened the menu. Return
or Escape on any nested page rebuilds the same root. A Load window opened by GameMenu lowers
`menuUp`; refusal followed by Escape centrally rebuilds the campaign or town root and restores
`menuUp` before returning `ScreenGameMenu`. The first visible campaign-menu frame therefore consumes
the hidden span under stopped cadence, with no hashed-world or ambient-animation catch-up. A Load
window opened by the main menu returns there without a GameMenu rebuild, and a successful load keeps
its normal replacement destination. Long information pages use the existing Picker seven-row
window; paint and pointer dispatch use the same `Visible()` top and count.

**FR-6.** End Quest and Abort Game first open separate confirmation pages in the decoded
`(100,100)-(440,340)` panel. The mission page lists Change Map, Victory, Exit to Main Menu, Exit to
Windows and Return to Game. The town page lists Exit to Main Menu, Exit to Windows and Return to
Game. Every confirmation row is enabled. Neither entry action changes the session. Change Map and
Victory release the current map and reach the map picker; Victory does not manufacture a campaign
win. Exit to Main Menu reaches the main-menu screen but retains the current viewer and session seams;
the next entry or reset owns their replacement. Exit to Windows requests application exit. Return
or Escape cancels either page and rebuilds its root.

**FR-7.** Menu navigation and information reads do not change simulation hash, save byte form,
campaign progress, inventory or equipment. Tips Mode persists at the existing application-options
boundary. Sound changes survive map and town transitions within this `FrontEnd` only. Every other
menu field is discarded when the menu closes and recomputed at the next open.

## Design decisions

**DD-1.** A row carries a `gameMenuAction` value, not a callback. `menuRows` derives every page from
the current surface, page, context and gates; one switch is the action dispatcher. Order and action
therefore cannot drift through parallel tables.

**DD-2.** `GameMenuContextSource` is installed on each production viewer. It returns presentation
values and copies the relation slice. This is the only Sim Core interface the story adds.

**DD-3.** Nested destinations remain states of `ScreenGameMenu`, not new application screens. This
preserves the existing overlay composer, held background and cursor transition. Confirmation pages
select the town-sized panel independently of whether the held background is a mission or town.

**DD-4.** `audio.Player` remains the minimal playback interface. The concrete UI device implements
an additional `SetSettings(audio.Settings)` capability behind a private type assertion; a mutex
makes a setting update and a concurrent play observe one complete value.

**DD-5.** Install labels and objective prose are byte strings by the time the font draws them. The
objective path uses the same install rune encoder as typed menu accelerators and substitutes `?` for
an unrepresentable rune. Literal information rows neither strip `~` nor advertise accelerators.

**DD-6.** No simulation package file, hash input, persistence form or save version changes. The only
read is `World.Entities()` plus a copied `World.Relations()` value.

## Research reconciliation

The three root populations, their order, install bindings, accelerator walk, decoded gates,
confirmation populations, confirmation labels and confirmation rectangle follow `MENU-ITEM-011`,
`MENU-ITEM-012` and `MENU-KEY-013`. `MENU-KEY-013` does not decode Change Map's unmarked fallback.
Research does not establish the meaning of the mode word, the nested page layouts or the
confirmation targets. The owner-directed campaign mapping and coherent destinations remain recorded
together in `DIV-099`. Pass-1 witness debts `DIV-404` through `DIV-406` are closed by the correction.

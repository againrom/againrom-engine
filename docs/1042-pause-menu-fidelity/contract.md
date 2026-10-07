# Story `1042` — pause-menu fidelity and destinations

Base `0ec4798358ace4b67d021f6577fa6660c1e2ba9b`. Research pin
`d7ee0c62cfa4a16083d356f24b1870035e1f0209`.

## Result

The in-game menu has the original mission and town root rows in their exact order. The rows use the
install labels and label-derived accelerators. A campaign mission shows Load Game; a standalone map
shows Diplomacy in the same position. Every enabled root row reaches a working screen or action.
End Quest and Abort Game require confirmation before leaving the current session.

The pointable result is a headless EN and RU campaign witness that opens the mission and town menus,
checks their install-specific rows and accelerators, visits every submenu, returns without advancing
the world, and confirms each exit separately.

## Research state

- `MENU-ITEM-011` (High for the mission rows, order, messages, install-label binding and exclusive
  Load/Diplomacy pair; Medium for disable predicates) establishes the seven-row mission surface.
  Save is disabled outside mode 2. Load is shown in mode 2 and enabled when a matching save exists.
  Diplomacy replaces Load outside mode 2. Sound Options is disabled when the audio state reports no
  device. Quest Objectives is enabled only in mode 2.
- `MENU-ITEM-012` (High for the town rows, order, messages, absence of disables and confirmation-row
  populations; Unknown for the two exit targets) establishes the five-row town surface and the
  Abort Game confirmation.
- `MENU-KEY-013` (High) establishes that the first single `~` mark in the install label supplies
  the effective, language-dependent accelerator. It does not establish the constructor's immediate
  for an unmarked label; Change Map's `C` is this story's owner-authored fallback.

The semantic name of `campaign+0x6bc`, the submenu contents, and the confirmation exit targets are
not established by these claims. The owner directed this story to build coherent destinations
without waiting for more research. This story maps mode 2 to an active campaign mission and the
other mode to a standalone map. It updates existing `DIV-099`. Pass-1 witness debts are recorded
and closed as `DIV-404` through `DIV-406`.

## Behaviors

**B1 — root-menu population and gates.** A campaign mission lists Save Game, Load Game, Game
Options, Sound Options, Quest Objectives, End Quest, Return to Game. A standalone map replaces Load
Game with Diplomacy in the same position. The town lists Save Game, Load Game, Sound Options, Abort
Game, Return to Game. Every row uses its install label and the `MENU-KEY-013` accelerator walk.
Mission Save is enabled only for a campaign mission with a save seam. Mission Load is enabled only
when the combined save list is non-empty. Mission Sound Options is enabled only when an audio
configuration is available. Mission Quest Objectives is enabled only for a campaign mission. Town
rows are all enabled. Disabled rows remain visible and cannot be activated by pointer, Enter or an
accelerator.

**B2 — settings destinations.** Game Options opens a page that shows and toggles the existing Tips
Mode preference through its existing persistent store. Sound Options opens a page that shows and
changes the current master mute and volume. Sound changes affect subsequent playback in the current
process. The sound preference is session state and is not written into a campaign save or the
options file.

**B3 — mission-information destinations.** Quest Objectives opens a page showing the current map's
shipped description, or an explicit no-objective sentence when it is empty. Diplomacy opens a page
showing the current directional relation of the local roster slot to every roster slot present in
the live standalone map. The page reads the current world when the menu opens, so script changes are
not replaced by map-load values. These two page presentations are owner-directed interpretations,
not claims about ROM1 submenu layout.

**B4 — return navigation.** Return to Game closes the root menu. Escape on the root menu also closes
it. A submenu Return row or Escape goes back to the same root menu with its population and enabled
states rebuilt. This includes Escape after a refused Load: campaign and town roots are rebuilt,
and a campaign return restores `menuUp` before the first visible menu frame. No submenu input
reaches the map or town behind the menu. The world, hashed cadence state, camera and ambient clocks
remain held for the whole nested visit. A main-menu Load return and a successful load keep their
existing destinations.

**B5 — confirmed exits.** End Quest opens the researched five-row confirmation and Abort Game opens
the researched three-row confirmation, both with their install labels. Change Map and Victory
release the mission and go to the map list without manufacturing a victory. Exit to Main Menu
selects the main-menu screen while retaining the current viewer and session seams until a later
entry or reset replaces them; Exit to Windows requests application exit. Return or
Escape cancels either confirmation and restores the root menu. Merely opening a confirmation changes
no campaign, world or persistence state. These destinations are owner-directed because research
establishes the row population and labels but grades the two exit targets Unknown.

The five behaviors are one slice because every destination is selected by the same root-row model,
uses the same held background, and returns through the same menu state. Splitting destinations from
root composition would leave enabled rows with no destination or destinations with no production
entry point.

## Acceptance

- Synthetic composition tests compare the complete ordered production paint stream for the three
  roots, four submenus and two confirmations against independently frozen digests. A one-pixel
  mutation at the production panel or row draw call changes the corresponding digest.
- Mutation-sensitive input tests activate every root row through Enter, pointer release and its
  resolved accelerator. They prove disabled rows do not dispatch and repeated accelerators select
  the first row in screen order.
- Session tests distinguish campaign mission from standalone map, exercise the exclusive row,
  verify current diplomacy is read at open time, and prove every submenu keeps the map held.
- Game Options round-trips Tips Mode through the existing store. Sound Options changes the settings
  used by the next play and does not change a save snapshot.
- End Quest and Abort Game change no destination before confirmation. Confirm and cancel paths are
  tested independently on both source surfaces.
- Install-gated EN and RU tests compare each resolved word to its exact source-table lookup, compute
  accelerators from those bytes, and drive every root, nested and terminal action through production
  `FrontEnd` and `App` wiring.

## Domains and aspect scope

- **Client:** menu composition, row models, pointer and keyboard input, navigation, and live audio
  settings.
- **Campaign & Scripts:** campaign-versus-standalone classification, current map description,
  current diplomacy projection, and confirmed session destinations.
- **Persistence:** existing Tips Mode storage is reused. Sound settings and menu navigation are
  explicitly session-only. Save payloads and the simulation byte form do not change.
- **Sim Core:** the Client receives a read-only projection of the live directional relation. No sim
  field, command, digest or byte-form version changes.

The closure matrix covers all twelve aspects. AI, triggers, inventory/equipment and combat are
expected `N/A`; their adjacent interfaces are checked for unchanged world advance and input
isolation.

## Out of scope

- Reconstructing ROM1 submenu art or undocumented controls.
- Multiplayer or editing diplomacy.
- Treating the map description as researched proof of ROM1's Quest Objectives contents.
- Persisting sound settings or adding new option keys.
- Typed save names, overwrite, delete, autosave, or changes to save-file discovery.
- Any simulation or AI change owned by concurrent story `1037`.

## Divergence reconciliation

`DIV-099` is amended in place. The root omission debt closes. The same row records the authored
campaign/standalone mapping, settings and information-page contents, confirmation targets,
retained Exit-to-Main seams, and session-only sound preference. Pass 1 opened three W debts:
`DIV-404` for a complete paint oracle, `DIV-405` for per-field install lookup discrimination and
`DIV-406` for the full action/destination population. This correction closes all three.

## Review ceiling and stopping condition

The absolute ceiling is three adversarial passes. The chain stops at the first independent pass
with no P finding after the remaining surface list is empty. Pass 1 exhausted the original surface.
The fresh-pass remaining list is only the correction delta: campaign and town Load refusal/Escape
return, main-menu and successful-load controls, first-visible-frame cadence/hash/ambient hold,
rebuilt root gates, all nine paint states, all sixteen per-field install lookups, the complete
root/nested/terminal action population, and the Change Map and Exit-to-Main truth corrections.
Only P returns the story. Repeated findings require a full population sweep of that finding class.
W is recorded and D is fixed without another pass.

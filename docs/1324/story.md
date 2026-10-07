# Purse gold drop

## Intent and authority

Result: the player drops gold from the inventory purse. The gold leaves the
purse through a player-addressed command, becomes a ground Sack, survives a
SAVE and a cold LOAD, and returns through the ordinary pickup.

Authority: knowledge k163, `MENU-088`, `MENU-090` to `MENU-095`, and the modal
rectangle of `MENU-COMBAT-017`. Measured:
the purse double-click opens an amount editor with text `0`; its action
queues server command 0x23 (Player, positive affordable amount, cell); the
server resolves the Player's first actor, debits Player+38 before the Sack is
created or merged, and a cell more than 2 away on either axis falls back to
the first actor's cell; pickup credits the looter's Player. Unknown to
research: the pointer route to the amount, the parse of malformed text
(`MENU-092`), the editor's reopened caret and strings, a SAVE taken between
request and application, first-actor ties and a refusal after the debit.
Each is a bounded engine policy in DIV-2239 to DIV-2244.

## As built

- `pkg/sim`: `KindPlayerDropGold` (0x23) with constructor `DropGold`. Player is
  the roster slot, Group the full 32-bit amount, X and Y the cell. It is
  applied before entity lookup like `KindPlayerParameter`. `playerFirstActor`
  reads the saved Player container and member order, then the actor
  traversal, and refuses a slot named by two containers. A refused request
  changes nothing. The gold merges into a Sack already on the cell; over a
  saved-objects world it mints a generated Sack root and recomputes its value.
- `pkg/ui`: the purse cell drags to the ground (1000, or the available
  balance; shift takes all) or opens the editor on a double-click. The editor
  is a bounded modal with edit, action and cancel focus, caret and selection
  keys, and the modal's own rectangle. It captures keys and pointer while open.
  Escape, F-keys and SAVE cancel a held share or open editor.
- `pkg/game`: `enqueueGoldDrop` queues the command with the amount clamped to
  the previewed purse (world purse less queued requests less the held share).
  The simulation debits only when the command is applied, so an active pause,
  a SAVE and a LOAD show and save the unspent server purse. The pending queue
  projects the command into SAV and restores it with its full-width amount;
  the player-scoped validators bind no issuer.
  `appendInventoryGold` rebuilds the purse mark with the pack, so an item cell
  left by an unequip is never taken for the purse.

## Proof

- `pkg/sim`: `TestPlayerDropGold*`, `TestPlayerFirstActorFollowsPlayerContainerOrderAndRefusesAmbiguity`.
- `pkg/ui`: `TestPurse*`, `TestGoldEditSelectionSemantics`, `TestParseGoldAmount`.
- `pkg/game`: `TestEnqueueGoldDrop*`, `TestPendingGoldCommandSurvivesValidationAndProjection`,
  `TestAppendInventoryGoldMarksOnlyTheCurrentPurseCell`,
  `TestPurseDragDropsGoldThroughPauseSaveColdLoadAndPickup` (App drag under
  active pause, SAVE holding the unspent purse and the request, cold LOAD,
  one application, second SAVE and cold LOAD, pickup, 20 continuation ticks,
  an altered `Money` control).
- Release: `TestReleaseEnginePurseGoldDropSurvivesQueuedSaveLoadAndPickup`
  on the installed EN and RU missions.

## Open debt

- Native queue-time and held SAVE, refusal after the debit, first-actor ties,
  the pointer route and the dialog strings stay Unknown (DIV-2239 to
  DIV-2244).
- The editor's labels are authored English.

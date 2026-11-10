# Story 1630: the starter shows the real mod load order and which mods fit the base

## Intent

For the selected base, the starter's mod list shows the enabled mods in the order the game loads
them, and every refusal the game would raise for the set stands on the row of the mod it names,
before Play. A mod that does not fit the selected base says so on its row whether it is ticked or
not. Before, the list showed the enabled mods in the order they were ticked and only the game
refused a set, at Check or Play.

## Authority

Owner direction: one mod workbench, built without workarounds; the owner accepted the plan of
`review/mod-workbench/ESTIMATE.md` in the seat, of which this story is row 1 ("load order" and "mod
discovery" in its section 3). The mod format is Againrom's own (`docs/MODS.md`); nothing here
differs from the original game, so no DIV row is involved.

## As-built behaviour

- `mod.Order(mods, base)` is the one builder of the load order and its refusals. It returns
  `mod.Ordering`: `Mods` in load order and `Refusals`, each a `mod.Refusal` with its kind (api,
  base, conflict, unreadable requirement, missing requirement, requirement version, cycle), the
  refused mod and the ids and values it names. It checks every mod by id (its api and applies-to
  through `mod.Fit`, then each conflict, then each requirement) and the cycle last; the order of the
  checks is the order the earlier function returned on, so `Ordering.Err`, the first refusal, has
  the same text the launch printed. Under refusals the set is still ordered: a missing requirement
  adds no edge, and a cycle's mods follow the others by group and id. `Ordering.Of(id)` lists the
  refusals that stand on a mod, a cycle standing on each of its mods.
- `mod.Fit(manifest, base)` is the part of `Order` a mod raises alone: an api this build does not
  run, then an applies-to that does not name the base.
- `modrt.Load`, the game's launch, reads `Order(...).Err()` and then `Mods`.
- `installinfo.Info.BaseID` names an install's base for mods; `game.BaseID` forwards to it. The
  starter orders for the base id the game orders for on the same root.
- `app.modRows` (`cmd/starter/app.go`) stays the one producer of rows. With a base selected it calls
  `mod.Order` on the readable enabled mods and numbers them in its order; the enabled mods whose
  folder is missing or whose manifest cannot be read follow, by name; then the mods that are not
  enabled, by folder name, each with `mod.Fit` for the base. Without a selected base nothing is
  ordered or refused and the enabled rows are unnumbered.
- A refusal is shown by `app.refusalText` through the caption table (`refused-api`,
  `refused-base`, `refused-conflict`, `refused-missing`, `refused-version`, `refused-unreadable`,
  `refused-cycle`, and `refused-unusable` for a requirement on an enabled mod that cannot be read or
  whose folder is missing) in English and Russian, with the mod ids and values as data. The row is drawn in
  the refusal colour as `id version: refusal  title  (applies-to)`. The row of a mod whose manifest
  cannot be read keeps the manifest reader's text, as before.
- A refused mod can be ticked and unticked like any other. Play and Check are refused while an
  enabled row carries a refusal, with the `mod-unusable` caption naming the first such row in the
  list; an enabled row that cannot be read or is missing is named first, with its own error, before
  any refusal of an ordered row (`app.launchPlan`).
- Selecting a base or ticking a mod refreshes the rows: `modRows` is computed on every layout.
- `starter.ini` is unchanged: `settings.store` writes the enabled list as before, and its field
  comment now calls it a set. `-mods` keeps the ticked order and does not order the mods.
- The tick hint reads "Tick to enable. Enabled mods load in the order shown." (Russian alike) with a
  base selected, and "Tick to enable." without one, where the rows carry no number.

## Proof

- `pkg/mod`: `TestOrderNamesEveryRefusalAndKeepsTheFirstText` (every refusal of a seven-mod set,
  its kind and data, the refusals on each mod, the first text, the order under refusals),
  `TestFitIsTheModsOwnRefusalsOnTheBase`; `TestOrderRefusalsNameTheMods` keeps its strings and
  reads `Err()`.
- `pkg/modrt`: `TestOrderRefusalsReachTheCaller` unchanged: the launch prints the same first error.
- `pkg/installinfo`: `TestBaseIDPrefersTheDetectedProfile`, moved from `pkg/game`, with a rom2-ru row.
- `cmd/starter`, manifests written into `t.TempDir()`:
  `TestEnabledModsStandInTheGamesLoadOrderOnEveryBase` (rom1-en, rom1-ru, rom2-en, rom2-ru: the
  enabled rows equal `mod.Order` over `mod.Resolve`, common mods first, ticked in reverse; each
  base refuses its own rows; the mod that is not enabled follows),
  `TestRequirementOnAnEnabledUnusableModIsNotCalledNotEnabled` (the required mod's manifest cannot be
  read, or its folder is missing: both rows, Play and Check, English and Russian),
  `TestEveryRefusalStandsOnTheRowOfItsMod` (api, applies-to, conflict, missing requirement,
  requirement version, cycle on both mods, a mod that is not enabled, Play refused, changing the
  base clears rows), `TestModListOrderAndBrokenFolders` (tick order no longer counts).
- Frames: `TestStarterModListFrames` draws the main view in the states ordered, refused and misfit,
  English and Russian, and checks each row's colour; with `AGAINROM_STARTER_MOD_FRAMES` set it
  writes them as PNGs through `internal/witness`. No image is committed.
- Guard: `TestStarterLinksNoGamePackage` (`internal/archtest`); the starter gained no import of
  `pkg/game`, `pkg/ui` or `pkg/modrt`.

## Open debt

- One mods folder with rescan on focus, the mission list and Start, and keeping the enabled list at
  Play are later rows of the plan.
- A row cut by its width hides the later refusals of its mod, while the Play and Check line states
  all of them.
- A mod that only loads after a cycle is labelled as in the cycle, as the game's own text does.
- `starter.ini` naming one id twice is one clean row while the game refuses it, as before this story.
- The starter's `-base` argument still comes from `Base.ID()` with `base.Nameable`; it is a launch
  flag, not the mod base, and is unchanged.

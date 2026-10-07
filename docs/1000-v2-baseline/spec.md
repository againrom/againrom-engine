# 1000-v2-baseline — spec

The game as built at the v2 epoch (2026-08-15, master `9ab5a86`, research pin `7d41fa4`,
`formatVersion` 50). One section per domain of `docs/DOMAINS.md`. Known gaps are cited as
`DIV-NNN` rows of `docs/DIVERGENCES.md`; this file does not restate them.

This spec describes what exists, at domain level. Per-feature behaviour lives in each story's
own `spec.md`; this file is the map, not the detail.

## 1. Assets

All shipped formats load from a lawful install on both roots (en, ru): RES/LM archives, REG
class registries, ALM maps, SPR256/SPR16A sprites, palettes, `Data.bin` class tables, and the
install's text files (CP866 decoded). Access is through `pkg/vfs` dispatch over the install
root, taken from `AGAINROM_ASSETS` or `-assets`. No asset ships in the repo; all test fixtures
are synthetic.

## 2. Sim Core

`pkg/sim` advances by `Step(state, commands) → state` on a fixed integer tick, stdlib-only, no
floats. State serializes through the sim's own binary writer (`formatVersion` 50) and hashes to
a digest used for determinism comparison across the shipped campaign. Occupancy, terrain
passability and cost, two-tier pathfinding, movement cadence, the world clock and the day/night
cycle are in. `internal/archtest` enforces the import DAG and the determinism wall's source
scan.

## 3. Combat & Magic

Melee and ranged combat resolve with to-hit, damage, armour and weapon reach; units die, leave
corpses, corpses decay and are lootable. Spells: the spellbook, casting (including staff casts
and script-commanded casts), damage and healing application. Point-effect arms land no state
(DIV-001) and a placed area effect harms nothing (DIV-002); both are queued for story
0170-spell-effects on a complete research ground.

## 4. AI & Orders

Units acquire targets by sight and hostility, engage, chase and give up routes; guard radius
and guard acquisition hold; group commands and group orders (including patrol) execute; escort
closes and holds formation (residues in DIV-007).

## 5. Party, Items & Heroes

Character generation (point-buy), the derived-stat graph, experience from use, training and
skill moves. Inventory, ground sacks, containers, stacks; equipment with the wear rule; item
semantics from `Data.bin` columns. Party slots, joins, and hero recompute on equipment change.

## 6. Campaign & Scripts

Mission scripts run: check and instant opcodes over authored trigger nodes, mission start and
outcome, campaign advance, and the campaign drive from mission to mission through the town. The
progress instrument is `pipeline/check-milestone.sh`'s census of shipped script nodes this
build cannot run: 313 at this baseline (story 0169, in review at the epoch, takes it to 59).
Dialogue panels use a live speaker's currently dressed figure. When no live actor matches, the
synthetic portrait follows `DLG-SPEAKER-022` and carries twelve empty equipment slots.

## 7. Town & Economy

The town surface with its buildings; the shop sells and buys with original stock and pricing
rules (rounds 0157-r1/r2). The shop screen's composition debts are DIV-003 and DIV-004; round
0157-r3 owes them. Start money and the money cycle match research.

## 8. Client

One windowed client (ebiten): main menu (install's own words, 0168), character generator, town,
shop, in-game view with HUD panels (unit info, compact panel, hero sheet), in-game menu,
save/load screens. Rendering: displaced lit terrain, sprites with palettes and shadows, fog of
war, night darkness. Input: mouse, keyboard, wheel (DIV-005), box select, group hotkeys. Text
draws in both roots' encodings. Sound plays. A headless scenario layer (0155) drives the same
game for tests and evidence; the code is the instrument.

## 9. Persistence

The build saves and loads its own campaign+sim state (`formatVersion` 50, slots) and reads the
original's `.sav` far enough to resume: party, walkability, explored map (0144, 0147–0150). A
field that does not survive save/load is a Persistence defect wherever the feature lives — the
standing aspect-matrix check.

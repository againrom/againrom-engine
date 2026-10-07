# Plan — 0137 every unit states its own character sheet

## Shape

Four seams, in dependency order: the placement sheet in the map-loading tier, the band in the
window tier, the join in the driver, the report in the tool.

```
pkg/mapload/sheet.go   PlacedSheets(map, table) -> per-placement sheet     (new)
pkg/ui/panel.go        CharacterBand on UnitCharacter; the skill row       (edit)
pkg/game/panelchars.go placed -> ui.UnitCharacter, merged with the party   (edit)
pkg/game/world.go      two constructor arguments, one overlay guard        (edit)
cmd/classdump/sheet.go the sheet block, from both map-bearing verbs        (new)
```

## Decisions

**DD-1 — the derivation lives in `pkg/mapload`, not in `pkg/game`.** Two consumers need it: the
driver, which turns it into a window value, and the tool, which prints it. The tool's row in the
import DAG reaches the map-loading tier and stops there; giving it the window tier instead would
drag the graphics dependency into a developer tool and widen an allow-map entry for a readout.
Placing it one tier below both consumers keeps one statement of the rule (FR-1, P-1).

**DD-2 — the sheet type is the tier's own, not the window's.** `pkg/mapload` cannot name a window
type and must not: the window tier sits above it. So the tier states a plain value of integers and
the driver copies it field by field into the window value — the same mechanical copy the party
path already performs out of the recompute's derived set. The copy is the only place the two
vocabularies meet (FR-1, FR-5).

**DD-3 — the band is an enumeration, not a boolean, and it carries the unknown case.** Three
states exist and two of them are already distinguished by the `Known` flag the window value
carries; a boolean beside that flag would make one state expressible twice and let the two
disagree. The window's own enumeration is separate from the map-loading tier's resolution arm even
though the two agree today, because the arm has four values for two bands and the window must not
learn the resolution ladder (FR-2).

**DD-4 — a creature's five weapon-kind columns are copied into its skill positions at build time,
not resolved at draw time.** The alternative is a reader that consults a different field of the
same value depending on the band; that puts the band rule in every consumer instead of in the one
builder. Copying makes the window's formatter a formatter, and it leaves the window's own
weapon-kind family field exactly where it was — which is what makes FR-4a's repetition a copy
rather than two competing sources (FR-4, FR-4a).

**DD-5 — the General position of a creature is expressed by the row printing five numbers, not
six.** The window value's skill array stays six wide and comparable; what changes is the
formatter, which is handed the band and prints slots 1 to 5 for a creature. No sentinel integer is
invented for "unstated", because a sentinel is a number and this contract forbids showing a number
for an unstated position (FR-4, FR-7, P-4).

**DD-6 — a person's sheet is derived with an empty loadout, which is exact today and is fenced.**
Of the six things the sheet takes from the derived-stat graph — four statistics and the two
families — none reads the loadout: the weapon reaches only the active-skill slot and the weapon
fold, and the equipment modifier's two family terms are filled by nothing in this build. So
passing no equipment yields the same six values as passing the placement's own, and the sheet does
not have to resolve equipment at all. That equivalence is a fact about today's tree and not a
property of the graph, so it is written into the contract as FR-5a and pinned by a test that
asserts an equipped placement and a bare one agree — the test is what will fail the day the
equipment seam is filled, instead of the panel quietly going wrong (FR-5, FR-5a).

**DD-7 — the skill levels come from the row, not from the derived set.** The graph restores and
clamps slots 1 to 5 before anything reads them; the party path already states the member's own
levels rather than the restored ones, and a placed person is stated the same way, so one rule
covers both bands of person. The two readings agree only while the graph's skill bonus is zero,
which is why FR-5 names the row rather than the graph (FR-5, FR-6).

**DD-8 — the two lookups are merged, not carried side by side.** The driver keeps one character
map, as it does today, filled with the placements first and the party second. The party's ids are
minted after the placements', so the two cannot collide; ordering them this way makes that an
observation rather than something to defend. Every existing reader of that map — the draw path,
the re-arm write — is untouched (FR-1, FR-6).

**DD-9 — the live overlay is guarded on the band, at the one line that writes a skill.** The draw
path already replaces a known character's skill positions and experience from the entity's own
per-slot experience each tick. The experience total stays unconditional, because it is state every
entity carries; the per-slot level write is guarded to the person band, because a creature's
positions are columns (FR-8).

**DD-10 — the tool prints from the same two sources the report already uses.** Everything a built
world carries — the pools, the damage pair, absorption, to-hit, defence, sight, rate — is read off
the entity, exactly as the existing combat row is; only the four statistics, the two families and
the experience columns come from the sheet. Nothing is recomputed in the tool (P-3, FR-9).

**DD-11 — one report function, two call sites.** The map-dump verb reaches loose maps and the
campaign verb reaches the container's. A single function takes a map, a table and the built
world's entities, so the two verbs cannot come to print different sheets (FR-10).

**DD-12 — the campaign verb builds its world at the ordinary difficulty and with the equipment
collections named.** FR-10's own rule for a caller that states no setting; the verb takes no
difficulty argument and this story adds none, so the ordinary setting is what it gets. The two
equipment collections are added to that verb's table because the verb already claims its numbers
are a player's, and a person built without them is bare (FR-10, FR-11, AC-12).

**DD-13 — unstated columns are printed as a dash, and the legend says why once.** Weight on every
row, and a person's XP value. A dash is not a value and cannot be mistaken for one; a zero can
(FR-9, P-4).

## Risks

**R-1 — the two five-wide families could be crossed.** The units row stores the elemental family
first and the weapon-kind family second, and our two field names follow that order; the window's
two rows follow the same names. A silent transposition would look entirely plausible on screen.
Mitigated by pinning the slot-to-family binding in a test that builds a row with ten distinct
values, asserts which array each of the two blocks lands in, **and** asserts which of them reaches
the skill positions — a test that pins only the arrays would enshrine whichever binding was
written rather than measure it.

**R-2 — a creature's weapon positions could be overwritten by the draw path.** The overlay writes
every slot of a known character's skill array on every tick; if the band guard is missed the
columns are replaced by levels derived from an experience nothing seeded, and the row reads
plausible zeroes. Mitigated by a test that steps a world holding a creature and reads the readout.

**R-3 — the placement-to-entity id rule could drift.** The sheet map is keyed by the placement
index because the world builder mints ids in that order. Stated by the tier that mints them, and
pinned by asserting the built world's entity count against the map's placement count where the
tool already does.

## Success criteria

- **SC-1** — the local gate is green: build, vet, gofmt over our own files, the whole test suite,
  and the asset, doc-budget, audit and hotfix-ledger checks.
- **SC-2** — a test shows a creature placement's four statistics and both families equal to its
  row's columns, and its General position not drawn.
- **SC-3** — a test shows a person placement's statistics capped, his six levels the row's, and
  his two families the derived ones; and shows an equipped placement's two families equal to a
  bare one's.
- **SC-4** — a test shows the party's characters unchanged in every field but the band.
- **SC-5** — a test steps a world and shows a creature's weapon positions unmoved while a party
  member's move with his experience.
- **SC-6** — the tool prints a full sheet block for every placement of campaign missions 10 and
  20 on both lawful roots, and the two roots agree.
- **SC-7** — the sheets of the placements of the row the owner named are recorded, with their
  unstated positions shown as unstated.
- **SC-8** — the mission-10 headless run is compared before and after, and any movement in its
  outcome is reported with its numbers.

## Traceability

| FR | Decisions | Criteria |
|---|---|---|
| FR-1 | DD-1, DD-2, DD-8 | SC-1, SC-2 |
| FR-2 | DD-3 | SC-2, SC-3 |
| FR-3 | DD-1, DD-2 | SC-2 |
| FR-4 | DD-4, DD-5 | SC-2 |
| FR-4a | DD-4 | SC-2 |
| FR-5 | DD-2, DD-6, DD-7 | SC-3 |
| FR-5a | DD-6 | SC-3 |
| FR-6 | DD-7, DD-8 | SC-4 |
| FR-7 | DD-3, DD-5 | SC-2, SC-3 |
| FR-8 | DD-9 | SC-5 |
| FR-9 | DD-10, DD-13 | SC-6, SC-7 |
| FR-10 | DD-11, DD-12 | SC-6 |
| FR-11 | DD-12 | SC-6 |

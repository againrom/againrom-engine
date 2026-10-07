# 0137 — every unit states its own character sheet

**Intensity:** spec-first / static.
**Terrain:** brownfield for `pkg/game`, `pkg/ui` and `cmd/classdump` (each already draws or prints a
readout and this story changes what it says); greenfield for the placement-sheet contract itself,
which nothing states today.

## Why

The unit panel states a character — four statistics, the skill row, experience, and the two
five-wide families — for a party member and for nobody else. Every other unit on the field gets a
zero character and the panel omits those rows entirely, so the same window is full for one unit and
nearly empty for the unit standing beside it. The numbers are not missing: the definition row every
placement resolves to states them, and the loader already reads that row to build the entity. What
is missing is the second half of the readout, the half that says which statistics the entity's
numbers came from.

Nothing in this seat can report unit data either. The tool that walks a map's placements prints a
health maximum, a rate, an owner and a combat row; the statistics, the pools, the two families and
the experience columns are all absent from it, so a question about a unit cannot be answered
without writing a new program each time.

## Vocabulary

- **Placement** — one unit record of a map, in the map's own order.
- **Band** — which definition collection a placement resolved against. A **creature** resolved
  against the units collection; a **person** resolved against the humans collection, by any of the
  three routes that reach it. A party member is a person for every purpose of this document.
- **Sheet** — everything a readout states about one unit: the four statistics, the two pools, what
  a blow throws and meets, the **skill positions**, the two **families**, the experience, the sight
  and the rate.
- **Skill positions** — the six positions of the readout's existing skill row: General first, then
  the five keyed by the weapon kinds Blade, Axe, Bludgeon, Pike, Shooting.
- **Elemental family** — the five-wide row keyed by Fire, Water, Air, Earth, Astral. The readout
  already draws it; this story changes only who it is drawn for.
- **Weapon-kind family** — the second five-wide row, keyed by the same five weapon kinds as the
  skill positions 1 to 5. Also already drawn.
- **Stated** / **unstated** — a value the readout asserts, against a position for which nothing in
  the definition data says anything. An unstated position is shown as unstated; it is never shown
  as a zero, because a zero is a measurement.

## Current behavior — the baseline this changes

- A character is built for the members a mission was started with, keyed by the entity ids the
  start minted, and for nothing else. A map's own placements are absent from that lookup.
- The panel's four statistic rows, its skill row, its experience row and its two family rows are
  each gated on the character being known, so all seven are omitted for every unit a map placed.
- The eight combat numbers and the rate ARE stated for every unit, off the entity.
- The tool's map dump prints, per placement, the resolution arm, the entry index, the owner, the
  health maximum, the rate and the movement domain, then a combat row of charge, relax, to-hit,
  defence, absorption, damage base, damage spread, always-hits and sight. It prints no statistic,
  no pool pair, no family and no experience.
- The tool's campaign census reaches the maps inside the campaign container, which the map dump
  cannot; its own table omits the two equipment collections a later story added, so a person it
  builds a world for wears no armour and no shield.

## Functional requirements

**FR-1 — every placement states a character.** A character is produced for every placement that
resolves to a definition entry, keyed by the entity id the world builder mints for it. A placement
that resolves to no entry, and a map with no table to resolve against, state no character — an
absent entry, not a character of zeroes.

**FR-2 — a character names its band.** Every character says whether it was stated by the creature
band, by the person band, or by nothing at all. Every reader that must treat the two differently
reads that, and no reader infers the band from a value.

**FR-3 — what a creature states.** Its four statistics, its elemental family and its weapon-kind
family, each from the row's own columns and none of them derived. Its experience is unstated: the
units collection has no per-slot experience column.

**FR-4 — a creature's skill positions carry its weapon-kind columns.** Positions 1 to 5 hold the
same five numbers its weapon-kind family holds, and not a skill level: a creature has no skill
level of any kind and the collection states none. Position 0, General, is **unstated** — no column
of that collection fills it, and it is shown as unstated rather than as a zero.

**FR-4a — that repetition is deliberate and is disclosed.** For a creature, and only for a
creature, the readout states those five numbers twice: once in the skill positions and once in the
weapon-kind family row. The weapon-kind family row keeps its existing source on both bands, so a
person's is not touched; the skill positions are what the owner ruled a creature's sheet shows.
Neither row is suppressed to hide the repetition, because both are correctly sourced and a
suppressed row is an absence this contract would then owe an explanation for.

**FR-5 — what a person states.** His four statistics, capped as everything downstream reads them;
his six skill levels in his skill positions, General included, **exactly as his row states them
and not as the derived-stat graph restores and clamps them** — the same rule the party path
already follows; his elemental family and weapon-kind family as the derived-stat graph produces
them; and that graph's experience.

**FR-5a — a person's two families are stated without resolving his equipment.** They are the same
values his equipment would produce, because nothing in this build fills the equipment modifier's
two family terms. If that ever changes, this requirement is what has to change with it.

**FR-6 — the party's own path is unchanged.** A party member's character is still derived through
the recompute, off the member the mission was started with, and states exactly the values it
stated before. This story adds the band to it and changes no number.

**FR-7 — the panel states a character for every unit it can.** Every row the panel already draws
for a party member is drawn for any unit a character was produced for. The skill row states six
numbers for a person and five for a creature — position 0 is not drawn for a creature, because
FR-4 leaves it unstated. No other row's condition changes on either band.

**FR-8 — a creature's skill positions are not moved by experience.** The readout replaces a
person's skill positions with levels derived from the entity's own per-slot experience on every
tick it is built. A creature's are not replaced: they are columns, and no experience moves them.
The experience total itself is still read off the entity for every character, on both bands.

**FR-9 — the tool prints the whole sheet.** One block per placement, in this order: Body, Agility,
Mind, Spirit; Health as current and maximum; Mana as current and maximum; Dmg as a range; Absorb;
Attack; Defense; the five weapon-keyed skill positions; the elemental family; Weight; XP; Sight;
Speed. Each block names the placement, its band, and the entry it resolved to. Weight is
**unstated** on every row: no definition row states one. XP here is the row's own experience
**value** — what felling the unit is worth — which only the units collection carries, so it is
unstated for a person. It is a different quantity from the experience the panel's own XP row
states, which is what the entity has earned.

**FR-10 — the tool reaches campaign maps.** The sheet block is printed both for a map named as a
file and for a map named inside the campaign container, so a campaign mission's units can be
reported without extracting anything. Where the caller states no difficulty setting the ordinary
one is used, which is the setting the tool's other map-bearing verb already defaults to.

**FR-11 — the campaign verb resolves equipment.** The table that verb builds names the same
collections a map's placements are actually armed from, so the worlds it builds are the worlds a
player gets.

## Acceptance criteria

| # | GIVEN | WHEN | THEN |
|---|---|---|---|
| AC-1 | a map placing units of both bands, and a table | characters are produced for it | every placement that resolved to an entry has one, and every placement that did not has none |
| AC-2 | a creature placement | its character is read | the four statistics and both families equal its row's own columns, unchanged, with the elemental family from the elemental columns and the weapon-kind family from the weapon-kind ones |
| AC-3 | a creature placement | its skill row is read | positions 1 to 5 hold the row's five weapon-kind columns, in Blade…Shooting order — the same five numbers its weapon-kind family row states (FR-4a) — and position 0 is unstated |
| AC-4 | a person placement | its character is read | the four statistics are the capped ones, the six skill positions are his row's own levels rather than the graph's restored ones, and both families are the derived ones |
| AC-5 | a mission started with a party | each member's character is read | it is byte-for-byte what it was before this story, plus the person band |
| AC-6 | a running world holding a creature and a party member | a readout is built | the creature's skill positions still hold its columns, and the member's still hold levels derived from his entity's experience |
| AC-6a | a person placement carrying equipment | its two families are read | they equal the values the derived-stat graph gives him with no equipment at all (FR-5a) |
| AC-7 | a subject of each band | the panel resolves its skill row | six numbers for the person, five for the creature |
| AC-8 | a subject whose character is unknown | the panel resolves any character row | the row states nothing, exactly as before |
| AC-9 | a map with no table, and a placement resolving to nothing | characters are produced | the lookup is empty or the entry is absent; nothing panics and no zero character is minted |
| AC-10 | either lawful root, campaign mission 10 and 20 | the tool prints the sheets | a block appears for every placement, in FR-9's order, and the two roots' outputs agree |
| AC-11 | either lawful root, the placements of the row the owner is looking at | their blocks are read | the four statistics, both families, the pools and the combat numbers are stated, and Weight and the person XP value are unstated |
| AC-12 | the campaign verb over a lawful root | it runs | it still reports every map, and the worlds it builds resolve armour and shields |

## Properties

**P-1 — a character is a pure function of the map and the table.** Produced twice from unchanged
inputs it is equal, so a readout built twice cannot state two sheets for one unit.

**P-2 — nothing here reaches simulation state.** No statistic, no skill position, no family and no
band is written to an entity, to the byte form or to any digest. The serialized form version is
untouched.

**P-3 — the tool derives nothing it can read.** Every value a built world carries is read off the
world; only the values no entity holds are read off the definition row.

**P-4 — an unstated position is never a zero.** Every position the readout shows as a number has a
column or a derivation behind it.

## Out of scope

- Hiring mercenaries, the campaign chain, and how a skill level moves.
- Equipping armour and the armour fold; the drawn body.
- Naming a placed unit's weapon in the panel's weapon row. A placed unit's weapon name stays
  empty and that row stays absent for it.
- Deciding where the original draws each value. The layout is not decoded; only the owner's
  ruling for the weapon positions is followed, and it is named as authored.
- Any change to what the eight combat numbers, the rate or the pools state.

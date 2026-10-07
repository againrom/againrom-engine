# 1209 — the remaining special tooltips

## Result

The map-list hover hint and the spellbook popup — the mission book and the
shop's Book toggle, `DIV-118` — are now bound, composed from spells.txt names
and installed labels exactly as `TEXT-HOVERTEXT-052` binds them, gated on the
same availability bit the catalog already carries, with every existing
disclosure, actor-kind, ownership and rectangle gate kept. The monster-spell
hint is built and gated the same way, but it is not reachable in this build:
no placed creature in the loaded mission set carries a nonzero `KnownSpells`,
because the class-level spellbook setup `UNIT-SPELL-007` describes is not
implemented for a creature spawn (only a human/party roster row gets one, see
Open debt). Every other target the four hover claims name was already bound
before this story, and a full re-check against the census in
`TEXT-HOVERSET-049`, `TEXT-HOVERCHAR-050`, `TEXT-HOVERROOM-051` and
`TEXT-HOVERTEXT-052` finds five gaps that remain, each for a reason the census
states. No simulation, save or hashed state changed.

| | count |
|---|---|
| getter/index groups the census table names | 25 |
| bound before this story | 17 |
| newly bound this story | 2 (map-list hint, spellbook popup) |
| built and gated this story, not yet reachable | 1 (monster-spell hint, `UNIT-SPELL-007`) |
| left, with a stated reason | 4 (inherited getters, chargen live-value join, six local zero-return bodies, `dialogs.txt[135]`/`[136]`) |
| left, no new gap by design | 1 (the dispatch bodies TEXT-HOVERSET-049 names) |

## Intent

Owner's standing request, carried from the original `DIV-1270` row: complete
the remaining special-tooltip targets and their composition, using the
existing text research, the shared delay and the normal font, without
changing the delay, the font or the owner-directed placement `DIV-1270`
already records.

## As built

### The map-list hint

`TEXT-HOVERTEXT-052`'s map-list getter (`L11131`) returns row-owned text or
`dialogs.txt[134..136]`. `mapListHoverLines` (`pkg/ui/tooltip_targets.go:428`)
builds the hint from a `PickerRow`'s own `Description` (`pkg/ui/picker.go:43`,
carried unchanged from `game.MapEntry`, `pkg/game/maplist.go:65`) and, when a
decoded size is present, a caption line reading `<MapListSize>: W x H`.
`Words.MapListSize` (`pkg/ui/words.go:118`) is the installed `dialogs.txt`
local slot 134 caption ("Size of map" in the shipped root), decoded through
`EncodeInstallText`'s new dialogs slot (`pkg/game/installtext.go`) and wired
through `frontend.go`'s row-building re-encode. `dialogs.txt[135]` and `[136]`
("recommended players", "minimum level") are the claim's other two named
captions; the decoded map record (`pkg/formats/alm`) carries no per-map
column for either field, so there is no value to pair either caption with —
a format gap, not an unread caption (`pkg/ui/words.go:114-117`).

`pickerTooltip` (`pkg/ui/tooltip_targets.go:409`) reaches the hint through the
`RowAt` hit test the row's own click already shares with its draw, so the
hint appears only over a row the picker itself would accept a click on.

The original hover-help screen this claim describes sits among strings the
engine does not implement (the multiplayer lobby); the engine's own
map-selection picker is the nearest reachable screen carrying the same
per-row size and description metadata, so the hint binds there. That choice
is recorded in `DIV-1270` (see below), not left implicit in code.

### The spellbook popup

`TEXT-HOVERTEXT-052`'s spellbook getter (`R1207`) binds four captions —
mana cost (main117), damage (main118), range (main123), duration (main124) —
to a spell's own characteristics; the claim's other seven named indices
(182-187/217: speed, resistance, sight, the two damage-probability captions,
absorption) name no live field this engine computes, so they are never read.
`spellInfoLines` (`pkg/game/spell.go:240`) builds the popup: the spell's own
name (from spells.txt, `Words.SpellBookNames`, not the internal `spell.txt`
identifier data the card helper reads), then each bound caption joined to its
live value as `"label: value"` — an authored join the claim does not give
(`DIV-1270`). Mana cost and range are unconditional; damage line only when the
rule is `Damaging`; duration only when positive. The four fields come from
`sim.SpellCharacteristicsFor` on the selected entity, rebuilt on every display
push, so equipment, skill and Mind changes move the fields the simulation
says depend on them while the flat installed values (mana cost, range) stay
put.

The composed entry is gated on the catalog's own availability bit
(`SpellEntry.Unavailable`, set by `selectedSpellbook`'s fixed 24-ID catalog
for any ID the selected entity does not know): an unavailable cell answers no
popup on both call sites — `tooltipTarget` (`pkg/ui/tooltip_runtime.go:61`)
for the mission book and `ShopHoverLines` (`pkg/ui/shopscreen.go:770`) for the
shop's Book toggle (`DIV-118`) — while the cell itself stays hoverable and
clickable in both, since `spellbookEntryAt`/`shopSpellEntryAt` are also how a
player selects a cell at all.

### The monster-spell hint

`TEXT-HOVERTEXT-052`'s card helper builds a monster spell list from `main192`
plus the `spell.txt` object. `monsterSpellHoverLines`
(`pkg/ui/tooltip_targets.go:27`) joins `Words.Hover[192]` with every
`Words.ItemSpellNames[id]` a subject's `KnownSpells` bitmask has set, in
ascending spell ID order, joined `", "` — an authored separator the claim does
not give — skipping any id with no installed name. It reports false — no
hint, not an empty one — for a subject with no known spell or no installed
heading.

`PanelSubject.KnownSpells` (`pkg/ui/panel.go:602`) is a read-only `uint32`
copy of `MapEntity.KnownSpells`, carried into the panel at `pkg/ui/panel.go:2121`.
`characterStatsTooltip` (`pkg/ui/tooltip_targets.go:162`) calls the monster
hint only when the hit field is `PanelFieldSpellcaster` and the subject's
`Char.Band` is `CharacterBandCreature`; a person's own spellcaster row keeps
its existing plain hint, because the claim binds this composed list to a
monster's card, not to a hero's spellbook, which the mission spell bar
already states.

The hint is built and gated, but not reachable in this build: no placed
creature entity in the loaded mission set ever carries a nonzero
`KnownSpells`. `UNIT-SPELL-007` describes the class-level setup routine that
builds a real spellbook from a unit class's `Spell 1..3`/`Probability 1..3`
columns; `pkg/mapload`'s own writer for `KnownSpells`
(`pkg/mapload/spawn.go`'s `rosterTemplate`, via `definitionForHuman`) applies
only to a human/party roster row, never a creature placement, so a monster's
card never has a nonzero bitmask to hand this hint. This is out of this
story's scope (binding a hint's composition, not populating creature
spellbooks) and is not counted as delivered in the table above.

### The census, re-checked on the merged tree

Measured by reading `pkg/ui/tooltip_targets.go`, `pkg/ui/panel.go`,
`pkg/ui/picker.go`, `pkg/game/maplist.go`, `pkg/game/spell.go`,
`pkg/ui/shopscreen.go` and `pkg/formats/alm/alm.go` against the four claims,
after merging engine main `817e648` (story1210's merge touched no file this
census reads; its own sim/combat and SAV-graft work is outside this story).

| claim | getter / index | engine binding | disposition |
|---|---|---|---|
| TEXT-HOVERCHAR-050 | card helper `R0916`, main155-158 (BODY/REACTION/MIND/SPIRIT) | `pkg/ui/tooltip_targets.go:85-92` (`panelTooltipSlot`) | bound (pre-existing) |
| TEXT-HOVERCHAR-050 | main159/160 (health/mana) | `pkg/ui/tooltip_targets.go:93-96` | bound (pre-existing) |
| TEXT-HOVERCHAR-050 | main161-164 (damage/tohit/absorb/defence) | `pkg/ui/tooltip_targets.go:97-104` | bound (pre-existing) |
| TEXT-HOVERCHAR-050 | main165-170 (weight/sight/speed/skills/resist/xp) | `pkg/ui/tooltip_targets.go:105-122` | bound (pre-existing) |
| TEXT-HOVERCHAR-050 | main171-180/188 (weapon/magic skills, monster resistance) | `pkg/ui/tooltip_targets.go:113-117,124-130` | bound (pre-existing) |
| TEXT-HOVERCHAR-050 | generator getters, free points main273 | `pkg/ui/tooltip_targets.go:393-395` (`chargenTooltip`) | bound (pre-existing) |
| TEXT-HOVERCHAR-050 | generator getters, difficulty/template/continue/back main247-255, name main256 | `pkg/ui/tooltip_targets.go:363-373,384-386` | bound (pre-existing) |
| TEXT-HOVERCHAR-050 | attribute +/- and value rectangles, live-value join | none | left: the claim states the rectangles format a current value in addition to prose, but does not give the join format; the rectangles keep their plain `Hover[155+i]` prose (`pkg/ui/tooltip_targets.go:380-392`) |
| TEXT-HOVERROOM-051 | `R0739`, town mask -> main233/234/235/236/237 | `pkg/ui/tooltip_targets.go:295-303` | bound (pre-existing) |
| TEXT-HOVERROOM-051 | `L11143`, school icons main171-180 via helper permutation | `pkg/ui/tooltip_targets.go:287-292` | bound (pre-existing) |
| TEXT-HOVERROOM-051 | `R1861`, shopkeeper main61, stock groups main62-65 | `pkg/ui/tooltip_targets.go:324-337` | bound (pre-existing) |
| TEXT-HOVERROOM-051 | shop/inventory getters, labels main54-60,74 and the item formatter for an occupied cell | `pkg/ui/tooltip_targets.go:307-351` | bound (pre-existing) |
| TEXT-HOVERROOM-051 | character getter `R0734`, book/backpack/doll/menu main8-14 | `pkg/ui/tooltip_targets.go:44-81` | bound (pre-existing) |
| TEXT-HOVERROOM-051 | hero/portrait arrows main52/53 or 121/122 | `pkg/ui/tooltip_targets.go:69-78` | bound (pre-existing) |
| TEXT-HOVERROOM-051 | tavern getter `R1018`, candidate card and worn-item mask | `pkg/ui/tooltip_targets.go:274-286` | bound (pre-existing) |
| TEXT-HOVERTEXT-052 | map-list getter `L11131`, row-owned text or `dialogs.txt[134]` | `pkg/ui/tooltip_targets.go:409-446`, `pkg/game/maplist.go`, `pkg/game/installtext.go`, `pkg/ui/words.go:118` | **newly bound this story** |
| TEXT-HOVERTEXT-052 | map-list getter, `dialogs.txt[135]`/`[136]` ("recommended players"/"minimum level") | none | left: the decoded map record (`pkg/formats/alm`) carries no recommended-players or minimum-level column at all; the gap is a missing per-map data column, not an unread caption |
| TEXT-HOVERTEXT-052 | card helper monster spell list, main192 + `spell.txt` | `pkg/ui/tooltip_targets.go:18-42,162-166`, `pkg/ui/panel.go:593-602,2121` | **built and gated this story, not yet reachable**: no placed creature carries a known spell (`UNIT-SPELL-007`) |
| TEXT-HOVERTEXT-052 | shop/equipment getters -> item formatter `R0964` | existing `ShopHoverLines`/`TownCandidateHoverLines` | bound (pre-existing, unchanged by this story) |
| TEXT-HOVERTEXT-052 | world-map getter `R1863`, `sites.txt` discovered/offered marker | `pkg/ui/tooltip_targets.go:236-247` | bound (pre-existing) |
| TEXT-HOVERTEXT-052 | spellbook getter `R1207`, main117/118/123/124/182-187/217, gated on an availability bit | `pkg/game/spell.go:206-266`, `pkg/ui/tooltip_runtime.go:56-63`, `pkg/ui/shopscreen.go:761-772` | **newly bound this story** |
| TEXT-HOVERSET-049 | 21 specialized text-return bodies | covered by the rows above, where a claim names the same body | bound where a specialized claim names it; no residual body left unaccounted |
| TEXT-HOVERSET-049 | six local zero-return bodies | none | left: the claim names no engine-reachable screen for these |
| TEXT-HOVERSET-049 | inherited getter `L06345` (69 tables), vslot18 hint-copy `L11132` | none | left: the engine has no widget tree to bind an inherited getter to, and the claim itself states setter/constructor bindings for inherited hints were not recovered |
| TEXT-HOVERSET-049 | dispatch bodies `R0791` (recursive child search) and `R0370` (single-widget ask) | none, by design | the engine's per-screen dispatch chain (`additionalTooltipAt`, `tooltipTargetWithSurface`) plays the same role directly; the four specialized-getter claims above give control -> slot mappings that make a searched widget tree unnecessary |

Everything `TEXT-HOVERROOM-051` and `TEXT-HOVERCHAR-050` name that this
census lists as "bound (pre-existing)" was verified again by reading the
merged tree, not assumed from the prior lane's report: town doors 233-237,
school icons 171-180, shop 54-65/74, character corner 8-14/52-53/121-122,
the tavern card and candidate slot mask, attributes 155-170/188, chargen
247-256/273, world-map sites, and the item formatter.

### Oversized or corrupt descriptions

Left with their current bounded handling. `alm.EncodeDescription`
(`pkg/formats/alm/encode.go:49`) and the decode counterpart
(`pkg/formats/alm/alm.go:43,647`) fix the field at 64 bytes; the tooltip
wrapper's width is clamped to `tooltipMaxWidth` (`pkg/ui/tooltip.go:13`, 320
design pixels) and wrapping always progresses. No reachable crash or
unbounded allocation was found, so no fix was made.

## Proof

### Focused tests

- `TestTooltipPickerStatesRowOwnedTextAndDecodedSize`
  (`pkg/ui/tooltip_test.go:71`): a picker row with no decoded metadata gets no
  hint; a row with a decoded description and size gets both lines, the second
  reading `Size of map: 256 x 256`.
- `TestTooltipMonsterSpellListJoinsHeadingAndKnownSpellNames`
  (`pkg/ui/tooltip_test.go:388`): a creature subject with two known spells
  gets `"Spells: Wall of Fire, Heal"`; a person subject on the same row gets
  no hint; a creature with no known spell gets no hint.
- `TestReleaseSpellbookPopupComposesFromSpellsTxtAndInstalledLabels`
  (`pkg/game/spellpopup_release_test.go`): reads a spell's name from the
  installed spells.txt row (not the binary rule identifier), reads all four
  bound captions from the installed main.txt row the claim names, and checks
  the popup's own `"label: value"` join and order against those installed
  values, live on both roots — a mismatched RU install would fail this test
  exactly as an English regression would.
- `TestMissionSpellPopupGatesOnAvailability` and
  `TestShopBookPopupGatesOnAvailability` (`pkg/ui/spellpopupgate_test.go`): an
  unavailable spell cell answers no popup on either call site, while staying
  hoverable and clickable so a player can still see the catalog entry.

All four pass on the merged tree. `TestReleaseSpellbookPopupComposesFromSpellsTxtAndInstalledLabels`
is registered in `internal/gatedtests/testdata/population.txt`, so the release
chain runs it against both installs; the two gate tests are fixture-based and
run under the normal `go test` chain.

### Screenshots

Neither the map-list nor the monster-spell hint's own screen has any CPU
composite path in this build: `composeScreen` (`pkg/ui/app.go`) refuses
`ScreenPicker` outright, and `HeadlessFrame` (`pkg/ui/headless.go`), the only
place `paintTooltip` runs, refuses the mission screen the monster-spell
hint's card belongs to. Neither refusal is this story's to lift, so
`cmd/screenshot` cannot reach either hint. A witness, `cmd/tooltipshot`,
composes each hint's own popup box through three thin exported wrappers
around already-tested production code (`ui.MapListHint`, `ui.MonsterSpellHint`,
`ui.ComposeTooltipHint`, the same "exported only for the witness" idiom
`pkg/game/panelchars.go` already established) and writes it as a PNG, using a
real install's own `FrontEnd.Maps` row and a real campaign mission scan for
its inputs.

The spellbook popup's own screen (the mission book) does composite in
`HeadlessFrame`, so its capture is a direct screenshot of the running popup:
a controlled mage party member with a known damaging spell is placed in
mission 10, the world is paused so no scripted notice can reopen and block
the popup, the entity is selected, the book bar is confirmed shown, and the
spell cell is hovered with the tooltip delay set to zero before the frame is
read back.

One EN and one RU capture for each of the three hints, written to
`review/story1209/` (untracked):

- `review/story1209/maplist-hint-en.png`, `review/story1209/maplist-hint-ru.png`
- `review/story1209/monsterspell-hint-en.png`, `review/story1209/monsterspell-hint-ru.png`
- `review/story1209/spellbook-popup-en.png`, `review/story1209/spellbook-popup-ru.png`

The map-list hint's and the spellbook popup's captures are real decoded data
end to end. The monster-spell hint's captures are a labeled authored
fallback: searching missions 1-120 of the EN install found 1015 creature-band
entities and not one carries a nonzero `KnownSpells` bitmask, for the reason
stated above. `cmd/tooltipshot` falls back to two real installed spell names
under an authored bitmask and says so in its own printed line, so the
fallback is never mistaken for a found monster. What each image shows is
described in the return report, not restated here, per `PROSE.md`'s rule
against repeating the same fact in two places.

### Gates

- `gofmt -l` clean.
- `go test -trimpath -count=1 ./...` exit 0.
- `bash scripts/check-no-game-assets.sh` clean.
- `bash pipeline/check-release-tests.sh <EN root> <RU root>`.
- `bash pipeline/check-div-claims.sh`.
- `internal/storyguard`: `CommentBytes` rises from the pre-merge 7664887 to
  7715897 on the final merged tree. Every rise is named by file, in the order
  it landed, in `internal/storyguard/baseline.go`'s own comment: this story's
  hint-binding note (5463), two merges of independent main work reaching
  7679471 then 7687012, the `cmd/tooltipshot` witness and its
  `internal/archtest/dag.go` row (5512 + 549), a further main merge (mission
  20 town SAVE, reaching 7700697), the correction pass's own spellbook-popup
  getter and its two new tests (3822, named by file in the same commit that
  merges main a final time), and main's own attack-cycle-latch and
  dying-pursuer's frozen-order work. All six comment forms stay at zero.
- `internal/archtest`: `cmd/tooltipshot` is registered in the DAG allow-map
  (`internal/archtest/dag.go`) naming `pkg/ui`, `pkg/game`, `pkg/mapload` and
  `pkg/render/text`, the same two tiers `cmd/screenshot` already names plus
  the two single-value imports the tool's production call signatures require.
  `TestLiveTreeClean` passes with no ratchet change to `CommandLiteral` or
  `Coordinators`.
- Milestone census (`missionrun -mission N -trace -ticks1 | grep -c
  UNSUPPORTED`) for mission 10 and mission 20: unchanged from
  `pipeline/milestone-baseline.txt`. This story touches no simulation or
  script code.

Exact verdict lines and counts are in the return report.

## Open debt

- **Inherited getters** (`TEXT-HOVERSET-049`). The engine has no widget tree
  to bind them to, and the claim's own setter/constructor bindings for
  inherited hints were never recovered.
- **Chargen live attribute values** (`TEXT-HOVERCHAR-050`). The claim
  confirms the +/- and value rectangles format a current value in addition to
  prose, but not the join format, so the rectangles keep their plain prose
  hint.
- **`dialogs.txt[135]`/`[136]`** ("recommended players"/"minimum level"). The
  decoded map record (`pkg/formats/alm`) carries no per-map column for
  either field; the gap is the format, not an undecoded caption.
- **Creature spellbooks** (`UNIT-SPELL-007`). The monster-spell hint is bound
  and gated correctly for any `KnownSpells` bitmask it is handed, but no
  placed creature in this build's loaded missions carries a nonzero one:
  `pkg/mapload/sheet.go`'s `creatureSheet` has no spell field, and the only
  `KnownSpells` writer (`pkg/mapload/spawn.go`'s `rosterTemplate`) runs off a
  human/party roster row, never a creature placement. Populating a creature's
  spellbook from its class's `Spell 1..3`/`Probability 1..3` columns, the way
  `UNIT-SPELL-007` describes the original setup routine doing it, is a
  `pkg/mapload`/spawn-population question, out of this story's scope (binding
  a hint's composition, not populating creature data), and left for whichever
  story next touches creature spawn data.

None of these four is a new gap this story introduces; each was already
outside `DIV-1270`'s covered slice and remains there. The spellbook getter
(`TEXT-HOVERTEXT-052`) is no longer open debt: the mission popup and the
shop's Book toggle are now bound. The oversized/corrupt-description clause is
also retired from this list: this story found no reachable crash or unbounded
allocation to fix.

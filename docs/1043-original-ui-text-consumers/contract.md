# Story `1043` — original UI text consumers

**Dispatched 2026-08-24.** The exact implementation base is
`d5b76f4c6c519098f28c924b1291dce318c181a4`. The required research pin is
`40ac9aff7bb716602d3b89960f8fc562d7ff9f81`. The base uses save form 61 and pins research
`49b154de5a0d81cde33313cd03c86ee57211bc25`; the lane advances only the research gitlink before
production code. The seat reread `TEXT-UI-032` through `TEXT-UI-047` at the required pin. The lane
copies this file to `implementation/docs/1043-original-ui-text-consumers/contract.md` and commits
the contract and pin advance before production code.

Story `1040` is present in the exact base. Its two item-description boundaries remain
`itemInstanceInfoLines` and `itemInstanceInfoLinesWithWeaponDamage`. The later seat hotfixes retain
those boundaries. They add fixed character-card columns, shared shop spellbook cells, doll drag
preview and compact tavern cells. This story localizes text without changing those layouts or
interactions.

## Result

The enumerated character panels, item descriptions, save acknowledgement, mission outcomes,
mission and town menus, world-map cards and selection status select their program-chosen strings
from the active lawful install at the indices published by `TEXT-UI-032` through `TEXT-UI-047`.
Each changed consumer retains the punctuation, numeric grammar and visibility gate established at
that row's confidence. The Russian install no longer falls back to English on those surfaces.
Existing localized command, pause, outcome and shop-control paths remain intact.

This is a finite current-production consumer story, not a general translation system. Original shop
category hover and item identification have no production interaction or modal in this build. Their
indices remain published inputs, but their mechanics are deferred to a separate Town & Economy
story. This story does not invent those interactions, infer a string for an unclassified callback,
or translate authored prose that the original reads from another file.

## Authority

- `TEXT-UI-032` fixes the sixteen-table order and the root-specific total: EN 1,568 lines, RU
  1,527, with only `credits.txt` differing in count.
- `TEXT-UI-033` bounds the static consumer population and its computed-target and bulk-copy blind
  spots.
- `TEXT-UI-034` through `TEXT-UI-036` give the shared unit/character panel's name source, fixed
  captions, four numeric formats and visibility gates.
- `TEXT-UI-037` and `TEXT-UI-038` give the item formatter's three `stats.txt` accessor sites and
  ten executable-authored formats.
- `TEXT-UI-039` and `TEXT-UI-040` give shop hover and identify strings whose consumers are absent
  here; they bound the deferred mechanics story and are not implementation authority for this one.
- `TEXT-UI-041` gives the four shop-control strings already consumed by production.
- `TEXT-UI-042` and `TEXT-UI-043` give the save acknowledgement and mission outcomes.
- `TEXT-UI-044` and `TEXT-UI-045` give the mission and town Esc-menu populations already built by
  story `1042`; this story retains them and folds them into the complete witness.
- `TEXT-UI-046` and `TEXT-UI-047` give the world-map and selection-status grammar.

Research confidence belongs to each claim. `TEXT-UI-033` is Medium for image-wide absence because
a runtime-computed target or unmaterialized bulk copy remains outside its static census. This story
does not promote that negative.

## Current consumer boundary

| Published surface | Current implementation surface | Story action |
|---|---|---|
| sixteen text tables | `InstallWords` retains only `main.txt` and `dialogs.txt` | extend one loader to retain the five requested tables; test all sixteen counts |
| character/unit panel | `RenderCharacterPanel` through town/shop, mission pane and chargen routes | carry one presentation-only name index and detail level to the shared composer |
| item formatter | `itemInstanceInfoLinesWithWeaponDamage` for mission and `itemInstanceInfoLines` for shop doll/shelf/table/pack | localize both final story-`1040` boundaries without changing their resolvers |
| shop hover and identify | no category-hover consumer and no identify state/modal | excluded; separate mechanics story |
| shop controls | four existing controls | preserve and include in the census |
| save acknowledgement | hardcoded `saved as ...` notice | replace its text with slot 203; keep this build's notice lifetime |
| mission outcomes and Esc menus | existing consumers | preserve and include in the census |
| world map | hardcoded two-line home card and `Reward` prefix; no marker-site label painter | localize the existing card and exact `%s: %d c` payment form; ledger the unproved join/separator |
| selection status | zero hides the panel; two or more still use the unit pane | render from `presentSelected`, key zero/plural state, and keep one actor on the full pane |

## Behaviours

### B1 — one install-text source, five requested tables

The current `InstallWords` path retains only `main.txt` and `dialogs.txt`. Extend that path, rather
than adding a parallel localization service, so its target state also retains exact
`unitname.txt`, `stats.txt` and `sites.txt` tables from the active install. Use the already-decoded
CR walk and per-field authored fallback. Global and table-local indices remain different typed
access paths. Missing, short or empty input falls back field by field and never blanks an otherwise
visible control.

The implementation does not load `credits.txt` merely to make EN and RU totals agree. The loader
witness enumerates all sixteen original table counts, while production retains only the five
tables consumed by this contract.

### B2 — shared character and unit panel

All three production populations converge on `RenderCharacterPanel`:

- town and shop through the `DrawTownCharacterRegion` wrapper;
- the mission character pane's selected-entity adapter;
- chargen's current-Human definition adapter.

The presentation seam gains a non-persistent `UnitNameIndex`; `ui.MapEntity` has no `TypeID`, and
the story may not infer one from its current selection identity. Before code, the lane enumerates
and wires every producer: selected mission entities from their resolved unit definition,
party/town records from their resolved definition, and chargen previews from the current Human
definition. The painter resolves `unitname.txt[UnitNameIndex]`. It resolves every fixed caption in
`TEXT-UI-035` from its stated global `main.txt` slot and applies `TEXT-UI-036`'s exact group gates.
Numeric values keep `%d`, `%d/%d`, `%d.%d` and `#%s: %d-%d`. Actor kind chooses the weapon-skill or
magic-school family; positive pools gate Health and Mana.

No current producer carries the original controlling level. Add one presentation-only detail level
to the panel subject. The existing party, town and chargen producers write full detail, value 7,
because every current route presents an owner-controlled or preview character. Independent
composer tests cover levels 0 through 7 and make every decoded threshold load-bearing. A divergence
records the Unknown original source and this build's full-detail policy. The value is not canonical
simulation state and is not derived from health, fog or combat fields.

### B3 — item text and retained shop controls

The item-description formatter resolves its three `stats.txt` labels and uses the ten literal forms
in `TEXT-UI-038`. Story `1040` owns item identity, effects, price and distinct-cell behaviour; this
story changes only the words and formatting with which those facts are shown. The mission item
popup keeps `itemInstanceInfoLinesWithWeaponDamage` and its live weapon-damage resolver. Shop doll,
shelf, table and pack descriptions keep `itemInstanceInfoLines` and the shop's nil resolver. The
lane enumerates every call site and localizes below those two boundaries rather than inventing a
new “final item-view API”. Headless and CLI diagnostics are not painters and remain explicitly
outside this story.

Undo, Buy, Sell and Exit retain the already-built slots 72, 70, 71 and 73 and join the complete
population witness. Category/shopkeeper hover and identify remain excluded because adding their
state, hit regions, price flow and modal transitions would change Town & Economy mechanics.

### B4 — notices, world map and selection status

The existing successful-save notice replaces hardcoded `saved as ...` with global slot 203. It keeps
this build's existing notice lifetime: `TEXT-UI-042` does not establish the unit of ROM1's numeric
`0x7530` argument, so this story does not reinterpret it. Main-menu, failure and cancellation paths
remain unchanged. Mission outcome text retains slots 140 and 141.

The world map replaces the existing two-line home card's hardcoded title and detail with local
`sites.txt[0]` and global slot 261 in the published order. Positive mission payment uses global slot
262 with `TEXT-UI-046`'s High `%s: %d c` form. The story does not invent site labels for mission
markers; their original consumer waits for a painter that exposes that surface. Exact card joining,
spacing and the Medium separator fact remain bounded in a divergence rather than being described as
original punctuation.

Selection status gains a named two-line renderer driven by the current `presentSelected`
population, not raw selection storage. Zero selected draws slots 47 and 48. Two or more draw slots
49 and 50 plus the count. Exactly one presented actor continues through the full panel path. The
mission pane's render/cache key includes the zero/one/plural presentation state, so a count change
cannot reuse a stale pane. Town and chargen preserve their existing empty-pane behaviour.

### B5 — complete production witness

A committed census names every production read and paint use this contract includes: loader transit,
all three panel adapters and the shared composer, both item-description boundaries, the four retained
shop controls, save notice, outcomes, both Esc menus, the world-map home/payment card and all three
`presentSelected` count arms. It also names the absent hover, identify and marker-label consumers as
exclusions. Tests do not derive expected text from `InstallWords` or from the same mapping table as
production. Each changed painter has a literal-slot or independently constructed expected-frame
witness, and a use-site mutation must fail before the production change is accepted.

Both preserved roots run the same headless production population without launching ROM1. The report
states selected, executed and skipped counts separately and prints the exact EN/RU differences for
every field. Concrete routes are: a real campaign mission with zero, one and multiple presented
actors; town and chargen panel adapters; one enchanted mission item popup; shop doll, shelf, table
and pack placements; all four retained shop controls; a successful save notice; Victory/Defeat and
both Esc menus; and a home plus positive-payment world-map card. The owner-visible prediction is the
same strings and numeric forms on those screens; any exact layout question not established by a
claim is reported to the owner rather than answered by launching the game.

## State and domains

Touched domains:

- **Assets & Formats** — indexed text-table loading and raw install bytes;
- **Client & Presentation** — word transport, consumer formatting and painters.

No simulation field, canonical hash, save form, trigger, AI order, inventory rule, price rule or
campaign rule changes. The story does not consume a byte-form version.

## Twelve aspects

| Aspect | Contract verdict |
|---|---|
| Data | PASS — five requested table sources and the sixteen-table census |
| Runtime state | PASS — one immutable word set per active install/viewer |
| Simulation | N/A — no simulation state changes |
| Player input | N/A — no input behaviour changes; retained controls are interaction witnesses |
| AI | N/A |
| UI/HUD | PASS — every named consumer reaches its painter |
| Triggers/scripts | N/A |
| Inventory/equipment | PASS — story `1040` facts are formatted, not changed |
| Persistence/save-load | N/A — the save acknowledgement is presentation only |
| Campaign/session | PASS — mission, town, map and selection roots retain lifecycle; no new transition |
| Shipped content | PASS — both lawful roots and all selected positions |
| Existing interactions | PASS — command, pause, outcome and shop text stay localized |

The N/A verdicts follow from the contract's absence of simulation, AI, trigger and persistence
changes. A discovered in-scope GAP fails the story.

## Exclusions

- Script dialogue, briefings, tips and other authored prose already read as content.
- Credits presentation and the 41 EN-only trailing credits positions.
- Text painted into bitmaps.
- A general runtime translation API or language switch after an install is open.
- Any static accessor target outside `TEXT-UI-033`'s named population unless a concrete production
  path is found during the required sweep. Such a path is either added to this finite population or
  recorded as a separately bounded remainder; it is never covered by an absence headline.
- Item mechanics, enchantment generation and price semantics owned by story `1040`.
- Shop category/shopkeeper hover, item identification and marker-site labels. Their strings are
  decoded, but their production interactions or painters are absent and require separate scope.

## Divergence discipline

The seat allocated `DIV-409` through `DIV-416`. The lane stops and asks if it spends that range.
Existing divergence rows for hardcoded English are amended or closed only
for the exact surfaces this story lands. `DIV-165` remains open for school/tavern text not covered
here. `DIV-191` remains open for layout, order and font questions the new rows do not settle. A new
row is not opened merely to duplicate an old one.

The literal `%d` in the original identify prompt is not silently “fixed” or painted here. The later
identify story decides whether to reproduce that consumer or record an owner-directed deviation.

## Review ceiling and stopping condition

At most three fresh adversarial passes. Only P — player-visible wrong behaviour or wrong hashed
state — returns the story. W is fixed and ledgered without another pass; D is corrected in place.
The chain stops at the first pass with no P and an empty named remaining surface. The review brief
must carry the complete consumer population already covered and only the correction delta forward.

## Dispatch preconditions

1. Story `1040` and the seat-owned town UI hotfixes are merged at exact implementation base
   `d5b76f4c6c519098f28c924b1291dce318c181a4`. This contract is reconciled against their actual
   item-description, character-card, shop and doll boundaries.
2. The lane advances the research gitlink from `49b154de5a0d81cde33313cd03c86ee57211bc25`
   to exact `40ac9aff7bb716602d3b89960f8fc562d7ff9f81` before production code. The target contains
   `TEXT-UI-032` through `TEXT-UI-047` and does not move any claim backward.
3. The story owns `DIV-409` through `DIV-416`. The current save form is 61. This story does not
   consume another form.
4. A new `wt-story-1043` starts from the exact implementation base above. No rebase or stash is
   used. Any later master change is integrated by a normal merge after a coherent green commit.

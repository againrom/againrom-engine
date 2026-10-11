# The second game's druid town built by the town composer

## Intent

On a second-game root, town 3 (the druid town) is a picture: the square with
its shop and inn highlights, the woman, the man, the lizard and the bug, the
two keepers who speak, and the inn as town 1's inn page with the druid centre,
its water, its taverner and its lower-right crop. The town composer
(`pkg/town`) builds both from one description,
`pkg/game/towns/rom2-town3.json`, beside `rom2-town1.json` and
`rom2-town2.json`; the second game's adapter (`pkg/game/secondtown.go`)
answers its hooks. The first game's town, town 1 and Kaarg are unchanged.

Base: `8a053839`, merged with main `b61fc0a6`. Knowledge pin: k253
(`db6fc8eb`).

## Authority

- Owner: one town composer; one builder per kind; a second-game screen is
  its own screen, with second-game evidence only.
- Square: `R2-ENGINE-231` (view, keys, music b15), `R2-ENGINE-251` and
  `R2-ENGINE-257` (shared town and room bases),
  `R2-ENGINE-253` (painter order and destinations), `R2-ENGINE-254` (mask
  sampler, hover, subset 2 at once), `R2-ENGINE-255` (clicks, keeper
  dialogue, tips, no gate frames), `R2-ENGINE-256` (clock, waits, bug routes,
  lizard), `R2-ASSET-064` through `R2-ASSET-066` (mask census, art,
  sprites).
- Inn: `R2-ENGINE-258` (water, taverner), `R2-ENGINE-368` and `R2-ASSET-089`
  (lower-right crop), `R2-ENGINE-260` (shared inn sides); town 1's inn claims `R2-ENGINE-246` through
  `R2-ENGINE-248` for the shared page. Shop: `R2-ENGINE-259`,
  `R2-ENGINE-369`. Bank slot 770: `R2-ENGINE-367`, `R2-ENGINE-331`.
- Campaign: `R2-SESSION-123` (town 3 by Leave50), `R2-SESSION-154` and
  `R2-SESSION-155` (SAV of the current town).
- Every cite list in the description names at least one `R2-` claim; no
  first-game claim is cited.
- Rows: DIV-2982 through DIV-2986 are new. DIV-2963, DIV-2964, DIV-2965,
  DIV-2966, DIV-2968, DIV-2971 and DIV-2973 extend to town 3; DIV-2390,
  DIV-2631, DIV-2635 and DIV-2639 are narrowed. DIV-2987 through DIV-2989
  are unused.

## Claims that differ from the brief

- The keeper dialogues read Scenario variable 0x300, bank slot 768, the stage
  (`R2-ENGINE-255`), not bank slot 770. Bank slot 770 gates the shop's fourth
  category (`R2-ENGINE-259`) and the inn's NPC 675 topic 78 offer
  (`R2-ENGINE-331`). The description follows the claims: `druidinnkeeper%d`
  and `druidshopkeeper%d` take `stage`.
- `R2-ENGINE-369` states that ordinary entry gets 0 from EnterShop and so
  bypasses the pending `shop\npc31m%d` dialogue. It does not state that no
  trading page opens; the engine has no second-game shop page (DIV-2963), so
  the shop click opens nothing in every second-game town.
- The four lizard vectors' values (`R2-ENGINE-256`) are in the claim's
  evidence, not in the public snapshot. The lizard is a still picture
  (DIV-2984).
- `R2-ENGINE-254` names Ddruid1 and Ddruid2 for the persons. The installed EN
  sfx.res holds no `town_druid/ddruid1` or `ddruid2` member; no claim names
  the keys of the other 18 slots (DIV-2983).

## Campaign

Town 3 is added by the ordinary departure after mission 50
(`R2-ENGINE-146`, `R2-SESSION-123`); before this story it was the row screen
TAVERN and GATES. Now the player sees the druid square, its inn and the two
keepers' rows. Its gate always leaves and keeps town 3 available
(`R2-ENGINE-144`, `R2-ENGINE-255`). The SAV records the room, square or inn,
in the existing `currentSecondCampaign.Room`; cold LOAD lands in it. The druid
inn's kind 0 talk to NPC 675 topic 78 now stores 2 in bank slot 770
(`R2-ENGINE-367`), so the inn stops offering it; Leave70 already stored 1.
The release witnesses reach town 3 by a constructed campaign position: a new
game, then the stage given, town 3 and mission 70 the destinations, entered
from the destination rows. The missions before it are not played.

## As built

- `pkg/town`, new parameters of existing kinds:
  - Hotspot (kind 4): `no-tip` gives a hotspot no tip, as the keeper
    selectors have none. An op `changed` runs only on the update that
    changed the hovered hotspot. The existing verb `arm` also starts a
    families member on its hover sheet at once, at cursor 0, its clock
    stamped.
  - Scheduled family (kind 10): `hover-sheet` names the hotspot whose hover
    picks a member's sheet; `start-when: idle` consumes an elapsed wait but
    starts the member only while it is idle; `sheet-positions` place a member
    by its drawn sheet.
  - Layer condition: `any` holds when any alternative holds; `running` with
    `sheet` holds while a families member runs that sheet. The building
    highlights use both: hover, or the person's subset 2.
  - Alternating page actor: `modulus` and per-state `residues` choose the
    state by the drawn wait's residue; motion `hold` shows frame 0 for
    `steps` admitted steps at the state's own `at`.
- New kind: the `route` program (verdict kind 11). It draws frame
  `frames*route+cursor`, starts after a first wait, picks a route on each
  later wait and keeps its cursor across starts. The flock carries a group
  and prefix per member and resets its cursor, so it cannot carry the bug's
  retained cursor (`R2-ENGINE-256`).
- `pkg/game/secondtown.go`: town 3 names `rom2-town3`; row texts INNKEEPER
  and SHOPKEEPER. Both answers of `TipAt` take `TipSlot`.
- `pkg/game/secondinn.go`: the kind 0 talk to NPC 675 topic 78 stores
  bank slot 770 = 2.
- Art keys sit in the description, as Kaarg's do.

## Proof

- `pkg/game` `TestRom2TownDescriptionEveryValueIsCited` walks the three
  second-game descriptions; `TestRom2TownThreeDescriptionShape` checks rows,
  keeper dialogue keys, missing keeper tips, music and the inn page against
  town 1's. `TestSecondInnTalkEffects` covers NPC 675 topic 78 in kinds 0
  and 3. `TestSecondCampaignThirdTown` sees the composed rows.
- `pkg/town` `hoversheet_test.go`: hover start at once, idle-only start,
  route cursor kept across starts, residue choice and hold, no-tip, and
  the bad forms refused at decode.
- Release witnesses on both second-game roots
  (`secondtowndruid_release_test.go`, `TestReleaseSecondTownDruid*`, nine
  tests): mask census 262188/5421/2144/3397/24098/9952 and the five exclusive
  rectangles of `R2-ASSET-064`, each byte's hotspot, tip and control, no
  byte 176; painter order, destinations, sprite counts 42 and 183 and person
  subsets (`R2-ENGINE-253`); hover starts subset 2 at cursor 0 and the
  highlight stays for its 20 files (`R2-ENGINE-254`); person waits and
  idle-only starts, bug routes of 61 with a kept cursor (`R2-ENGINE-256`);
  keeper clicks and rows open `druidinnkeeper60`, `druidshopkeeper60` and
  the `70` sections by stage, the shop click acts on nothing, GATES keeps
  town 3 (`R2-ENGINE-255`, `R2-ENGINE-369`); inn art, water files 0..8,
  taverner residues and its still at (264,152), the TextBackL crop at
  (464,238), Exit and Escape; the NPC 675 talk stores bank slot 770; SAV on
  the square and in the inn, cold LOAD, the same room and the mission 70
  World hash `34db738904977c81` equal for 8 ticks.
- The lizard vectors and the shop's category gate and modulo-30 keeper are
  not witnessed: the vectors are not public (DIV-2984) and the shop page does
  not exist (DIV-2963).
- RU: the square claims cover both locales; the RU run asserts the same
  values. The inn rectangles are EN values on RU (DIV-2965).

## Open debt

- The shop page with its category gate and keeper (DIV-2963); the tips panel
  (DIV-2968).
- The lizard vectors (DIV-2984); the square and inn sound keys and the
  ambient request clock (DIV-2983, DIV-2985, DIV-2972).
- The taverner's idle draw and state-3 wait (DIV-2985); the lower-right
  field (DIV-2986); draws and cadence (DIV-2982); inn field equality
  (DIV-2973).

# A hero's body and class come from one derivation

## Intent and authority

A hero's drawn body, its sheet directory and its class are now answered by one
builder, `heroAppearance` in `pkg/game/heroappearance.go`. Every site that
draws, stores or reads a hero's class asks it, and the flight class rule reads
the class the World stores instead of deriving its own.

Owner rulings: one builder per kind; ROM1 is a reduced ROM2; the town is
timeless; SAV is the only save format.

Claims (knowledge pin k363):

| Rule | Claims | Grade followed |
|---|---|---|
| The class is the seventeen-arm name chain over visible slots 0 and 1; a mage whose name stays `unarmed` is `mage`, a dying mage `mage_st` | `HERO-APPEAR-042`, `ANIM-107` | High / Medium |
| The hero range is `[0x20, 0x40)`; the fighter/mage axis is bit 1 of `typeID - 0x21` | `HERO-APPEAR-041` | High |
| The class is rewritten on each processed state or equipment message | `HERO-APPEAR-044`, `ANIM-107` | High / Unknown for timing |
| A map-authored human keeps its Humans-row class; an npc `Hero` placement takes the hero path | `ANIM-106` | High / Medium |
| A mercenary's class is keyed by its 1..15 type | `MERC-TYPE-001` | High |

## As built

| Kind | Builder |
|---|---|
| Hero body, directory and class | `heroAppearance(w, id, member, list)`. Input: the World's slots of the actor when the World holds him, else the party record's worn codes; an empty slot 1 widened by an unmaterialized starting weapon (`DIV-111`); the member's mage arm. It is the only non-test caller of `data.HeroAppearance`; `TestHeroAppearanceHasOneBuilder` in `internal/archtest` holds that. |
| Stored class | `redressPartyMember` writes the builder's body, directory and class to the record and the class to the World (`World.SetClientClass`). It runs at mission open, when a roster actor joins (`syncJoinedHeroes`), after an equip, unequip or worn drop, and, new, for each living map-placed Hero-type person at mission open (`redressPlacedHeroes`). Inside the step that applies an equipment order the class is stored once already: `tickStep` binds `mapWorld.equipmentClassRule` as the World's `sim.EquipmentClassRule` around `sim.StepReported`, and `stepWorld` calls it after each Equip (EquipDisplacing included), Unequip and DropWorn arm. It answers, from `redressedLook` (the builder), the class `redressEquipped` writes after the step, for the same members (`queuedEquipActors`, unhired). |
| Flight class | `flightClassRule(e)` returns the stored `e.Class` for a hero, a fixed client class, a Class 1 source and a non-humanoid; a non-hero humanoid keeps its `TypeID` arm. `sim.FlightClassRule` takes the entity only; the body list left the flight inputs. |
| Readers | the inventory doll (`refreshAppearance`), the joined-roster canonicalization, the campaign return, the town portrait (`composeMemberFace`), chargen (`assembleParty`), the original-party restore (`restoredMember`, unhired only), the spell client class (`spellClientClass`) and the mission art all call the builder. |

A dead party member at mission open, the open after a LOAD included, has his
record follow his slots through the builder, but his actor keeps the class the
World holds, the one the SAV restored (`DIV-4141`). Without this, the round-trip
gate's `game0013.sav dying` cases (two copies in the owner corpus) saved class
14 for a dead archer whose bow had left him and loaded class 1; that mismatch
was already present on main `d77853a03` with the same world hashes.

Kept arms: the hired skip, `MERC-TYPE-001`, `ClientClassFixed`, the non-hero
humanoid arm and the Class 1 arm.

The second game's hero pane (`pkg/game/secondheropane.go`) draws through
`composeMemberFace`, which now calls the builder. Its frames and hashes did not
move: the census below found no disagreement on rom2-en, and the rom2 release
expressions pass on rom2-en and rom2-ru.

## Proof

Instruments: `pkg/game/heroappearance_test.go`,
`pkg/game/heroappearance_release_test.go`,
`pkg/game/flightinputs_catalog_test.go`,
`internal/archtest/heroappearance_test.go`, `pkg/sim/equipmentclass_test.go`.

### Disagreement census

A temporary instrument, built on the base commit and removed before the
change, recorded at each derivation site the site's own answer, the World
slots, the record, the source and native class arms, the stored class and the
old flight class, and computed the builder's answer separately in an analysis
script. Population: the pkg/game release tests matching
`Hero|Class|Body|Appear|Redress|Flight|Projectile|Cast|Spell|Inventory|Doll|Portrait|Campaign|Dragon|Sav|Save|Load`
on EN (1439 pass, 0 fail) and RU (0 fail), and
`Town|Hero|Portrait|Inventory|Doll` on rom2-en (0 fail).

- Look sites (joined, placed, inventory, town, roster, companion, chargen,
  original, spell) with the shipped body list: 0 disagreements between a site's
  own body, directory and class and the builder's, on EN, RU and rom2-en.
- Flight rule against the stored class at the cast: EN 124553 events in 61
  distinct states, RU 124549 events in 61 states. Four causes:

| Cause | Old flight class | Stored / builder | Decided by |
|---|---|---|---|
| Widening: an unmaterialized starting weapon, slot 1 empty | unarmed 1, mage 23 | 3, 24 | no claim; World answer kept, `DIV-4140` |
| Source mage arm against the member: hero type 33, record fighter, source mage (EN ids 35, 41, 42, 76) | 23 | 1 | `HERO-APPEAR-041`, `HERO-APPEAR-042`: record and type axis agree on fighter |
| Stored class stale in the step that applies an equipment order: the redress ran after the step, so that step's shot or cast read the class of the slots before the order. Production route: the inventory and the weapon swap queue Equip, EquipDisplacing, Unequip and DropWorn into `mw.pending`, which `tickStep` applies (the front-end census case, id 73, is one). Tests that equip through `sim.Run` or `sim.Step` without the front end reach the same state | builder value | old value | `ANIM-107`; the class is now stored inside the step (below); the step order is `DIV-4148` |
| Placed Hero-type humans never redressed: m70 id 105 3 to 4, m100 id 110 24 to 23, m140 id 124 35 to 24, an original-save id 0 33 to 5 | builder value | Humans-row or saved class | `ANIM-106`, `ANIM-107`, `HERO-APPEAR-042` |

### Focused tests

- `TestHeroAppearanceReadsTheWorldOverTheRecord`,
  `TestHeroAppearanceWidensAnUnmaterializedStartingWeapon`,
  `TestAPlacedHeroTypeHumanStoresTheClassHisEquipmentSelects` (fixtures),
  `TestFlightClassReadsTheStoredClass` (stored 14; stored 1 over a mage source
  arm; non-hero 19; Unit 9; fixed 33).
- `TestReleaseAFighterHeroCastsFromTheClassTheWorldStores` (EN, RU): the
  mission-10 hero, record fighter, source mage: builder 1, old rule 23, stored
  1, flight 1.
- `TestReleasePlacedHeroTypeHumansStoreTheClassTheirEquipmentSelects` (EN,
  RU): mission 70, Naira beside a male primary stored, builder and SAV 14;
  Danath beside a female primary 4.
- `TestReleaseCampaignJoinIdentitiesAndMission20ClubmenUseCanonicalRows` moved
  its Danath golden from 3 to 4 (sword and shield, `ANIM-106`, `ANIM-107`).

### Sensitivity

`TestReleaseAHeroesWornWeaponDrivesBodyClassSaveAndShot` (EN, RU): a mission-10
hero equips a bow, then a short sword. Body, record class, stored class, SAV
class byte and flight class follow: archer 14 with a projectile, then
swordsman 3 with none.

### Equipment order in the release step

- `TestAnEquipmentCommandStoresTheClassTheBoundRuleAnswers` (pkg/sim
  fixture): the bound rule's answer is the class after each Equip, Unequip,
  Equip and DropWorn (3, 1, 3, 1); a fixed class and an unbound World keep 9.
- `TestReleaseAWeaponSwapInTheReleaseStepDecidesThatStepsShot` (EN, RU):
  mission 10, an archer attacking the nearest hostile looses his first arrow
  in step 178. An inventory equip of the short sword applied in step 178
  looses no arrow then or later (stored class 3); the sword-to-bow equip in
  step 178 looses none in that step and the first arrow in step 180 (stored
  class 14).
- `TestReleaseAStaffTakenOffInTheCastStepMovesThatCastsStart` (EN, RU):
  mission 41, the chargen mage's book Fire Arrow releases in step 11 at
  facing 64. An inventory unequip of the staff applied in step 11 moves the
  record's start by -80,-24, the cast offset of class 23 against class 24.
- Sweep: a temporary copy of the review's overlay (mission 10, bow and short
  sword, the other weapon equipped in step k for k 0..219, both directions,
  240 steps) ran on EN at the change and at a `git archive` export of
  `d77853a03`. All 440 swap runs and both no-swap runs gave the same flight
  starts and the same final World hash.

### Loss control

Two mutations of the committed change, each run on EN and then restored from
the commit:

- The flight rule's hero arm removed, so a hero takes the `TypeID` arm: 5 of
  5 witnesses fail (`TestFlightClassReadsTheStoredClass` 33 for 14; the
  sensitivity witness flight 33 for 14; the fighter witness flight 33 for 1;
  the placed witness Naira 34 for 14 and Danath 33 for 4; the placed fixture
  35 for 24 and 3).
- The placed redress call removed: 3 of 3 witnesses fail (Danath stored and
  SAV 3 for 4; the fixture actor 50 stored 3 for 24; the campaign join golden
  3 for 4).
- The dead-member rule disabled: `TestADeadPartyMemberKeepsTheClassTheWorldHolds`
  fails, stored class 1 for 14.
- `tickStep` binding no rule, so the step reads the class from before the
  order: the swap witness fails (arrow 1, picture 1, at 6280,14592 in step
  178) and the staff witness fails (start unshifted, 11608,17432). The
  in-step store disabled in `pkg/sim`: the fixture fails 4 of 4 and both
  release witnesses fail the same way.

## Open debt

- `DIV-4140`: the flight class of a hero in the widened state is the armed
  class the World stores. Research question: for a hero whose server equipment
  slot holds no item, which class does the server read when it builds his
  projectile and cast origin, the client-derived class or the actor class
  word?
- `DIV-4141`: a dead party hero keeps his actor's class at mission open.
  Research question: after a hero dies and his worn items leave him, does the
  class the original stores and draws for his corpse change with the emptied
  slots, and is it rewritten when a save of that corpse is loaded?
- `DIV-4148`: a shot or cast released in the step that applies an equipment
  order reads the class of the new slots; reach, damage and the record body
  follow after the step. Research question: when a hero's equipment changes
  in the same server tick in which his shot or cast is released, which class
  do that release's projectile and cast origin read, the class before the
  change or the class after it?
- A World stepped without the front end (`sim.Step`, `sim.Run`) binds no
  equipment class rule. Tests do this; `cmd/missionrun` equips that way but
  binds no flight inputs; the scenario driver sends no equipment order.
- The census instrument was run on the base only; the population after the
  change is witnessed by the release expressions, not by a second census.
- Research question: does a map-placed npc `Hero` person's class reach the
  server's swing and projectile reads before his first equipment message, or
  only the client's drawn class?

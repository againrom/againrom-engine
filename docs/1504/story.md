# Town SAVE writes the actor and Player fields as the original does

## Intent

A town SAVE writes each party actor's state word U50, U136, the action-run end U138, U6C, the
Mover U154 and its word lists U15C and U178 from what the actor holds in the town: for a member
kept from the last mission, the mission's values after the end-of-mission cull; for a member
built in the town (hire, tavern, AddHero), the Unit constructor's values. A town LOAD reads the
same fields back. At the next mission start a kept hero keeps the regeneration rate 1 until the
clock passes his carried run end by 80 sub-ticks, then rate 3, whether the mission starts
straight from the town or after a town SAVE and LOAD. The local Player's colour F44 is carried
the same way. Audit row G15, DIV-2500.

The story is first-game only. The second game keeps its earlier answer, as DIV-3358 does: every
claim cited here is first-game, and no second-game claim states these rules. Its town SAVE, mission
SAV and World hash equal engine main `e135b68d` on every path the story touched.

## Authority

- Owner rulings (`AGENTS.md`, brief "Owner rulings"): SAV is the only save format and every save
  point writes it; a field is written from the current state, then a rule derived from
  evidence, then the loaded document's bytes, then a constructor value research established;
  what cannot be written is debt, never a refused save. The town is timeless.
- Seat decision for this story: Player F44 is written from the mission's current value through a
  town carrier; F3D is 0 by the cull; F50, Outcome and the 32-byte settings block stay debt in
  DIV-2500; the selection list stays empty with its reason in DIV-2500.
- Claims, knowledge pin k346: `SAV-1353` (STORE writes in-memory values; cull stores U50 0 and
  U136 1 and leaves U6C, U138, Mover, Order, lists and Diary; town-built actor: Unit init U50 0,
  U136 1, U138 0, then U50 0xb), `SAV-1354` (U136 read first after LOAD; U50 overwritten by the
  join; U6C written before read), `SAV-1355` (U138 crosses STORE and LOAD; the regeneration rate
  is 1 until clock minus stamp exceeds 80; the clock restarts at 0), `SAV-1356` (the join reads
  Mover `+0x80` before replacing Mover and Order), `SAV-1357` (first reader of the lists and the
  Diary not established), `SAV-1358` (Player `+0x3d` 0 from the cull), `SAV-1359` (Player F44 is
  the ALM group colour + 1, overwritten by the next mission's Player build; F50 read only when
  positive; no writer of the block's other 31 bytes), `SAV-1360` (selection list writer and
  readers).

## Engine before this story

- The cull cleared the actor's action clock (`pkg/sim/missionreturn.go:83` at `460f4ef1`), the
  party carried no clock, the town writer wrote U138 = 0, and the next mission minted the actor
  with an unknown clock that the first tick set to 0 plus any remaining action. A kept hero
  therefore regenerated at rate 1 until sub-tick 81, then 3, whatever run end the mission left.
- The town writer (`cityBaseDocument`, `pkg/game/currentsave.go:207`) wrote every actor from the
  constructor `nativeCityUnitData` (`pkg/game/nativecity.go:195`): U50 0, U136 0, U138 0, U6C 0,
  a zero Mover with the domain mask and turn rate, a zero Order and empty lists; the Player from
  `nativeCityPlayerState` (`pkg/game/nativecityplayer.go:14`) with F44 0.

## As built

### Field sources

| Field | Before | After |
|---|---|---|
| U50 kept | constructor 0 (`nativecity.go:195`) | cull value 0 (`pkg/game/townkept.go:18`), carried; LOAD: the record |
| U136 kept | constructor 0 | cull value 1 (`townkept.go:18`), carried; LOAD: the record |
| U138 kept | constructor 0 | the actor's run end after the mission (`townkept.go:34`, `:72`); LOAD: the record |
| U6C kept | constructor 0 | native value an original LOAD read, else 0 (`townkept.go:72`); LOAD: the record |
| U154, U15C, U178 kept | constructor Mover, empty lists | the projected Mover and lists of the mission (`townkept.go:72`), turn-rate byte 10 from the Human; LOAD: the record |
| U158 and ring kept | constructor zero | after a mission: constructor zero (debt); LOAD: the record |
| U50, U136, U138 town-built | 0, 0, 0 | 0xb, 1, 0 (`townkept.go:26`) |
| Player F44 | constructor 0 | the town's carried colour: the mission map's group colour + 1 (`townkept.go:224`), or the loaded town Player's F44 |
| Player F3D | constructor 0 | constructor 0, equal to the cull's 0 |
| Player F50, Outcome, settings block; actor Diaries; selection | constructor / retained bytes / empty | unchanged, DIV-2500 |

### Game switch

The campaign service's `keepsTownActorState` (`pkg/game/campaignservice.go`) is true for the first
game and false for the second. Its sites: the carrier fill at mission end (`citymissionobjects.go`),
the carrier fill at a town LOAD (`keepLoadedTownActors`, `townkept.go`), and the town writer of the
actor fields and F44 (`cityBaseDocument`, `currentsave.go`). The cull keeps the action clock only
outside a second-game script (`pkg/sim/missionreturn.go`). With no carrier, `keptDeadline` seeds no
clock in the second game.

### Path

- `mapload.KeptActor` (`pkg/mapload/kept.go`) is the carrier on `PartyMember.Kept`
  (`pkg/mapload/start.go`); nil is a member built in the town. `clonePartyMember` copies it.
- Mission end: `NormalizeMissionSurvivors` keeps the action clock (`pkg/sim/missionreturn.go:83`).
  `currentMissionCityParty` (`pkg/game/citymissionobjects.go:30`) calls `carryKeptActors`, which
  fills the carrier of every carried member bound to a living actor. Both end-of-mission paths
  (`campaignreturn.go` and the detached city copy in `savedialog.go`) go through it. The Player
  colour is set beside the carried Diary (`campaignreturn.go:256`, `savedialog.go:233`).
- Town LOAD: `sav.Character.Town` (`pkg/formats/sav/townactor.go`, read in `actorCharacter`)
  holds the record's fields; `restoreOriginalTown` fills the carrier from the parallel character
  list for an engine town document and for an original one (`originalrestore.go:228`, `:241`)
  and the Player colour from the local Player (`originalrestore.go:268`). The Snapshot carries
  the colour (`save.go:285`, `:393`). A Mover or Order equal to what a constructor writes loads
  as absent, and a record holding only the town-built values loads as no carrier
  (`keptFromCharacter`), so a town state crosses an engine SAVE and LOAD unchanged.
- The town SAVE's bytes change for every party (town-built values, colour), so the pinned save
  hashes of the town room and town square traces are rewritten; nothing else in those traces
  moves.
- Town SAVE: `cityBaseDocument` calls `writeTownActors` and `writeTownPlayerColor`
  (`currentsave.go:244`, `:247`) after the constructor; the actor records follow the party's
  order.
- Next mission: `StartMission` seeds a kept actor's action clock with its carried run end
  (`keptDeadline`, `pkg/mapload/kept.go`; called at `start.go:833` and `:972`). The party a
  mission carries out holds no carrier (`CarryParty`, `pkg/mapload/carry.go:94`, beside the
  cleared `Saved`), so a live mission and one continued from a mission SAVE carry the same party;
  `carryKeptActors` fills it from the World. The mission SAVE's
  U138 producer and the carrier share `currentActionRunEnd` (`townkept.go:34`,
  `worldsaveclocks.go`).

## Proof

- Unit: `TestAKeptMemberStartsWithItsCarriedActionRunEnd`, `TestKeptActorClonesItsLists`
  (`pkg/mapload/kept_test.go`); `TestMissionReturnKeepsTheActionRunEnd`,
  `TestKeptStampHoldsRateOneUntilEightyPast` (`pkg/sim/missionreturnclock_test.go`);
  `TestTownActorWritesCarrierOrConstructor`, `TestTownActorLoadThenWriteKeepsFields`,
  `TestTownActorsFollowPartyOrder`, `TestMissionPlayerColor`, `TestCurrentActionRunEnd`
  (`pkg/game/townkept_test.go`).
- Release, EN and RU (`pkg/game/townkept_release_test.go`):
  - `TestReleaseTownLoadSaveKeepsCarriedActorFields`: every town document of the install root and
    the owner corpus, LOAD then SAVE, each carried field compared per persistent member. EN: 27
    documents, 180 members, U50, U136, U138, U6C, U154 (without byte 10), U15C, U178, U158 and
    its ring each kept 180 of 180; byte 10 kept 180 of 180. RU: 26 documents, 178 of 178 each.
  - `TestReleaseTownSAVAfterMissionWritesKeptAndHiredFields`: a town SAVE right after mission 20
    writes the hero's mission run end, U50 0, U136 1 and a Mover mask; four tavern hires write
    0xb, 1, 0.
  - `TestReleaseLoadedTownKeptHeroRegeneratesFromItsStamp`: the owner town `game0010` (hero stamp
    623) opens its next mission with the hero's clock at 623; alone on the mission terrain from
    sub-tick 0, 11 health gains of 262 at rate 1 until sub-tick 703, then 786 (rate 3).
  - `TestReleaseMissionKeptHeroRegeneratesFromItsStamp`: a hero moved late in mission 20 (run end
    329) opens the next mission with clock 329 straight from the town and after a town SAVE and
    LOAD; 7 gains of 158 at rate 1 until sub-tick 409, then 474 (rate 3), in both.
- Second game: `TestReleaseSecondGameProfileWitnessIsUnchanged` against
  `pkg/game/testdata/profilewitness/rom2-{en,ru}.txt`, which equal their content at `e135b68d`, is
  the guard: a second-game town SAVE with the first-game fields changes its pinned save hashes.
  `TestOnlyTheFirstGameKeepsTownActorState` (`pkg/game/townkept_test.go`) and
  `TestSecondGameMissionReturnClearsTheActionRunEnd` (`pkg/sim/missionreturnclock_test.go`) pin the
  LOAD carrier and the cull. A differential run of new game, mission 10, mission 20, town 2, a town
  SAVE, LOAD and SAVE, and mission 21 on rom2-en and rom2-ru gives equal SAV bytes and World hashes
  at base and head.
- Mission SAV RuntimeIDs: the world build and the Building constructor reserve every 4-byte value
  of the World's binary form as a used runtime ID (`reserveCurrentForm`). A seeded run end that no
  other dword of that form holds therefore moves the numbering of the next mission's SAV; in the
  first game this follows from the carried stamp.
- Loss control: with U138 dropped from the carrier (`keptFromMission`, `keptFromCharacter`),
  `TestTownActorLoadThenWriteKeepsFields` and the four release tests fail (45 changed U138
  values over the documents; stamp 0 on the loaded and the mission hero; the after-mission SAVE
  writes 0 for a run end of 29). Restored from the story's commit, all pass.

## Open debt

DIV-2500, narrowed: a kept member's Order after a mission, its U6C when no original LOAD read it,
a town-built member's U6C and Mover, actor Diaries, Player F50, Outcome and the settings block,
the selection list, and members of an original town whose party LOAD fell back to the source
roster. DIV-1016 records the carried run end.

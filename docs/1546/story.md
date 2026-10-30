# The second game's mission villagers look as in the original

## Intent

Every placed person of the second game's missions drew one fighter doll: male face 1, `mfighter`.
The placed figure took the face byte by the first game's rule, from the record word `+0x0a`, while
a second-game person finds its row by server id. This story draws each placed person with its own
row's sprite in the World and its own body in the inspection, and keeps both across SAVE and LOAD.

## Authority

- Owner remark 12 (triage W8): mission villagers and the Astrologer drawn with one fighter doll.
  Owner on the board: as recommended, research first (EXP-2086, knowledge k382).
- `R2-ASSET-172`, `R2-ASSET-173`, `R2-ENGINE-679` to `R2-ENGINE-682`, `R2-ENGINE-525`.
- The first game is not changed: no SAV byte and no World hash moves.

## As built

### World sprite

A placed person draws the File its row's type id names in `units.reg`. A person whose NPC word
(`R2-ENGINE-525`: the server id, above 10000 `(id/10)%1000`) lies in 21..23 draws as a client hero
of class `gender + 2*(ManaMax > 0) + 0x21` (`R2-ENGINE-680`). The World keeps the row's type; only
the drawing changes. Of the 605 placed persons on maps 1..400 of both second-game roots, one meets
the rule: Mage_Hero of mission 40 (record 36, server id 10230, word 23), drawn with the hero mage's
body and sprite.

### Inspection body

The body is `equipment/<class>/<face>.256`: the class from the type and gender, the face from the
row's face byte, gender in bit 7 (`R2-ENGINE-681`); the existing item layer composer draws over it.
The constructor sets the face byte, the SAV writes it in `U4B`, and LOAD rebuilds the same body
from the placement row (`R2-ENGINE-682`).

### Builders

| Kind | Builder |
|---|---|
| placed person's figure | `mapload.PlacedFigure`; the placement key policy `mapload.UnitKeys` gains `placedFaceByte` and `clientHero` as data |
| NPC word | `mapload.PersonNPCWord`, the rule moved out of `secondGameNPCKeys`, whose keys are unchanged |
| World sprite of a placed person | `game.missionAppearanceArt` with `placedClientHeroes` |
| inspection portrait | `unitPicture` and `composeUnitFigure`, unchanged |

No second builder is added.

## Proof

- Synthetic: `TestASecondGamePersonDrawsHisRowsFaceByte` (seven persons, figure, roster template,
  World type, constructor face byte, LOAD relink, client hero), `TestPersonNPCWord`,
  `TestTheFirstGameHasNoClientHero`, `TestAClientHeroDrawsTheHeroBodyOverHisRowsClass`.
- Installed, read only: `TestReleaseSecondMissionPersonsDrawTheirOwnLook` enters missions 30 and
  40 through the campaign and checks the sprite key, body key and saved face byte fresh, after SAVE
  and cold LOAD, and after a second SAVE and LOAD; PASS on rom2-en and rom2-ru, FAIL on base.
- Loss controls: the focused first-game doll and companion tests pass alike on base and
  candidate; the World hash after SAVE is equal on base and candidate for missions 40 and 30 on
  EN and RU.
- Review PASS (`pipeline/reviews/story1546-adversarial-review.md` in the seat): 1 764 synthetic
  placements equal on base and candidate in the first game; in the second only figure and face
  byte change.

## Open debt

- DIV-4274: the `heroes\mage_st` sheet at `+0x8c = 6` is not modelled; the writer of 6 is Unknown.
  The row cites `R2-ENGINE-680` for the server keeping the row's type; that claim does not read the
  server, and the type rests on `R2-ENGINE-525`, whose other `+0xe` writers are Unknown.
- DIV-4275: the inspection route and the item layer keys are Medium readings.
- That bit 7 of the face byte is the gender is Medium; no claim states it directly.
- A SAV written before this story keeps its old face byte on resave; the drawing is rebuilt from
  the row after LOAD.
- Who the Astrologer of mission 40 is stays Medium or Unknown (`R2-ENGINE-679`); the engine draws
  each record by its row.

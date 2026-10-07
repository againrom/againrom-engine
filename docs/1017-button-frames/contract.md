# 1017 — contract

## What this story settles

The owner reported (2026-08-19, chat, with a screenshot of the school): "еще есть бажок в школе,
окошко для кнопок есть, а кнопки сами больше чем окошко", and then "когда поправите этот баг, то в
таком же окошечке красивом нужно завести кнопки в таверне, генераторе и магазине".

The install ships, per room, one 160x238 button-area picture with the button wells baked into it and
one 140x46 bitmap per button per state. This build draws the school's area picture and then covers it
with its own flat boxes at its own geometry: `townSurfaceButtonRect` (`pkg/ui/townshell.go`) returns
166x52 boxes starting 16 pixels left of the shipped picture's own left edge. The tavern, the shop and
the character generator do not load their area pictures at all.

The owner also settled the school's third button (2026-08-19, chat): "Кнопка talk в школе не нужна
(как и в магазине), потому что выдача задания происходит просто при заходе один раз когда задание
можно выдать и диалог запускается. Всего один раз на каждое задание."

**That ruling is why this story also builds the on-entry offer, and it is not scope creep.** The
school's shipped area picture has two wells and this build draws three boxes, so the fix removes the
Talk box. The school's Talk box is the only way this build offers a school mission, and the shipped
campaign carries three of them: `TCMission = 121`, `131` and `61`, read from `scenario.reg` and
identical on both roots. Removing the box without building the offer would make three shipped
missions unreachable to buy a nicer frame. The offer is the precondition for the shipped layout, so
it lands in the same story or the story ships a regression.

## What will work after this story

1. **The school draws its two shipped buttons in their own wells.** `interface/training/buttonsarea.
   bmp` (160x238) is already loaded as `TownSchoolArt.Upper`; the four `interface/training/buttons/
   b{1,2}{on,off}.bmp` (140x46 each) are new. Train and Exit, no third box.
2. **The tavern draws its three shipped buttons in their own wells.** `interface/inn/buttonsarea.bmp`
   (160x238) and `interface/inn/button{1,2,3}{off,on}.bmp` (140x46 each), all new. Hire, Talk, Exit.
3. **The character generator draws its buttons in the same shipped frame.**
   `interface/chrgen/buttonsarea.bmp` (160x238), new. Its own button bitmaps are to be located by the
   lane; `interface/chrgen/buttons/` holds the 1256-byte stat steppers, which are a different control.
4. **The shop draws its four buttons in the same shipped frame.** The shop ships no `buttonsarea.bmp`
   under `interface/shopanim/`; where its button art lives is for the lane to establish. Undo, Buy,
   Sell, Exit.
5. **A pressed button shows its own `on` state.** Both states ship for every button this story draws;
   drawing only `off` would leave half the shipped art unread and a click with no feedback.
6. **The school and the shop offer their chapter's mission once, on entering the room**, as a
   dialogue, and never again for that mission. The Talk box goes from both. The tavern keeps its
   Talk button, which its shipped art has a well for and which selects among several NPCs rather
   than firing one offer.

## Why this contract does not split, at six behaviours

The rule is that a contract naming more than five behaviours either splits or records why it does
not. This one records why.

Behaviours 1 to 5 are **one mechanism over four rooms**: load the room's shipped area picture and its
per-button bitmaps, draw each button in its own measured well, and move the hit rectangle to match.
Counting them as four behaviours is counting rooms, not mechanisms; splitting them would buy four doc
stacks and four lanes for one result, which is the thing a vertical slice exists to avoid.

Behaviour 6 is a genuinely different mechanism and would normally be its own story. It is here
because behaviour 1 cannot ship without it without making three shipped missions unreachable, and
shipping them in sequence means either the school keeps a box its own art has no well for, or the
game spends a landing with content the player cannot reach.

**This story therefore takes the four-pass ceiling, not the three-pass one**, because behaviour 6
reaches Campaign & Scripts and Persistence alongside Client. If a fourth pass still finds a
player-visible defect, the diagnosis is that this contract should have been two and behaviour 6
should be lifted out with whatever remains.

## The observable result

`builds/current/`: the school's buttons sit inside their wells instead of over them, and the tavern,
the shop and the character generator show the same framed buttons the school does. Measured rather
than asserted, by a new developer tool `cmd/buttonframecheck`, on both preserved roots:

- each button bitmap's own offset inside its room's area picture, as a correlation winner with the
  next-best fraction anywhere else in the picture printed beside it;
- the composed region compared against the area picture outside the wells, so a button that paints
  one pixel outside its own well fails;
- a hit test at each well's own centre resolving to that button's action through production code.

## Positions already measured at the seat, and how

These are handed as **premises to verify, not as facts**. Each was produced by correlating the
shipped button bitmap over the shipped area picture at every offset and taking the winner, the same
method stories `1015` and `1016` used. Re-derive them in the tool before building on them; if the
tool disagrees, the tool is right and this section is wrong.

| Room | Area picture | Button | Offset in the area picture | Next-best | Ratio |
|---|---|---|---|---|---|
| school | `interface/training/buttonsarea.bmp` | `b1off` / `b1on` | `(4, 71)` | `(5,121)` / `(5,121)` | 38x / 99x |
| school | same | `b2off` / `b2on` | `(4, 117)` | `(10,134)` / `(17,144)` | 57x / 101x |
| tavern | `interface/inn/buttonsarea.bmp` | `button1off` / `button1on` | `(4, 44)` | `(18,63)` / `(10,89)` | 85x / 102x |
| tavern | same | `button2off` / `button2on` | `(4, 90)` | `(3,55)` / `(12,122)` | 57x / 101x |
| tavern | same | `button3off` / `button3on` | `(4, 137)` | `(20,120)` / `(11,108)` | 85x / 101x |

Every `off` and `on` pair agrees on one offset. The wells are 46 apart, which is the button height, so
they stack flush. The school's two sit lower in the same 238-pixel canvas than the tavern's three,
which is the area art's own composition and not an offset this story chooses.

The school's area picture is drawn at `TownUpperRegion.Min` = `(480, 0)`, so its wells are at screen
`(484, 71)` and `(484, 117)`. **Whether the tavern's, the shop's and the generator's area pictures
share that origin is not established here** and is part of the lane's measurement.

## Research

- `TOWN-154` (High) inventories all 63 `interface/training/` entries and names `buttonsarea.bmp`
  160x238 and the four `buttons/b{1,2}{on,off}.bmp` 140x46, with their filenames pushed at
  `L11671`..`L11672` **in a routine it did not read**. The sizes are research-backed; the
  destinations are not, and no claim publishes them.
- `TOWN-150` (High) decodes the two blit primitives: vtable slot `+0x18` writes every source pixel,
  slot `+0x38` skips source value zero. Which slot the button bitmaps use is not published.
- Nothing decodes the tavern's, the shop's or the generator's button art at all.

**UNKNOWN, and the story proceeds on the owner's directive rather than holding**: the destinations,
the blit slots, and the state machine that chooses `on` against `off`. Each becomes a divergence row.

## The twelve aspects

| Aspect | Applies |
|---|---|
| data | yes — new art nodes in three or four loaders |
| runtime state | yes — which button is pressed |
| simulation | no |
| player input | yes — the hit rectangles move to the wells |
| AI | no |
| UI/HUD | yes — the subject of the story |
| triggers/scripts | no |
| inventory/equipment | no |
| persistence | yes — behaviour 6's "once per mission" must survive a save and a reload |
| campaign/session | yes — behaviour 6 fires the chapter's own offer on entering the room |
| shipped content | yes — both roots, every room |
| interactions with existing mechanics | yes — the existing button actions must keep working |

## Domains touched

**Client** (8) for behaviours 1 to 5. Behaviour 6 adds **Campaign & Scripts** (6), which owns what a
mission offer is and when it fires, and **Persistence** (9), because "once per mission" is a fact
that has to survive a save and a reload. **Town & Economy** (7) is read through the existing
`Town.Offers` seam and its rules are not changed.

Four domains, which is what sets the four-pass ceiling above.

## Divergence rows expected

Reserved: `DIV-155`..`DIV-159`, allocated by `pipeline/next-div-id.sh` at the contract. Expected:
the unread destination routine, the unpublished blit slot, and any room whose button count in this
build differs from the count its shipped art carries.

## Out of scope

- **The tip windows.** `main/text/tips/*.txt` ships eleven texts and the original draws them in a
  bordered panel with a "show tips next time" checkbox and a Close button. That is story `1018`.
- **The tavern's own offer flow.** The tavern's NPCs are selected from its bottom cells and its Talk
  button acts on the selected one. That is existing behaviour and this story does not touch it.
- The shop's four button actions themselves, the generator's window layout beyond the button frame,
  and every room's character region.
- **What the offer dialogue looks like.** Behaviour 6 owes the offer firing once and the mission
  becoming takeable; the dialogue presentation reuses whatever this build already shows for a tavern
  offer rather than designing a second one.

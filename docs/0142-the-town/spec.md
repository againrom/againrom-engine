# 0142 — the town

**Intensity:** spec-first / breadth-over-polish. **Terrain:** greenfield screen on brownfield
wiring — the campaign spine, the carry and the front-end flow all exist and none of their
contracts move.

**Owner's calibration, normative for this story** (2026-08-11): *"Делаем всю эту историю как можно
более вертикально и быстро, тщательность не нужна, шлифовка будет после."* Breadth over polish. A
mechanism standing end to end beats any part of it being finished. Evidence honesty is not on that
axis.

## Why

Winning a mission that the campaign says leads to a town has, until now, put the player back on the
map list under a sentence saying the town is not built. The campaign's real spine is a town with
three buildings handing out missions; without it there is no game between missions, nothing for a
won mission to return to, and no reason for a party to be carried anywhere. This story builds the
loop: town → mission → town, with the state that survives it.

## Scope

**In:** the town screen and the four doors it opens; where the campaign lands in it and where a
finished mission returns to; the between-missions state (party, gold, which missions are done,
which offers are consumed, what the buildings unlocked); the **one** list of available missions,
the three buildings that append to it, and the **gates** that read it and start one.

**Out of scope:** graphics of any kind — every screen here is placeholder text in the engine's 6x16
debug font, exactly as the map list already is. Mercenary hire (the inn's other half). Buying,
selling and training — the shop and school hand out **missions** here, which is what their registry
keys actually are, and trade is a later story. **The world map itself**: it is what stands behind
the gates in a later story, and what stands there today is the list. Saving to disk. Anything that
changes what crosses into a mission.

## Functional requirements

**FR-1 — the town is where the campaign's town boundary lands.** Winning a mission whose successor
the campaign offers through a building puts the player in the town, not on the map list. That
boundary is read from the registry and is not a number written here: it is the campaign's own
`Offer.Town`, already computed, whose sentence this story replaces with a screen. On a stock
scenario that makes the first arrival 10 → 20 → town, because missions 10 and 20 are the two no
building names.

**FR-2 — the town stands in a chapter, and the chapter is derived.** The town's live chapter is the
lowest main mission the campaign offers that has not been won. It is recomputed, never assigned, so
winning a **side** mission does not advance it and winning the chapter's main mission does. A
campaign with nothing left to offer has no chapter and the town says so.

**FR-3 — a mission ends in the town, either way.** Once the town has been reached, dismissing a
won mission's banner returns to the town, and so does dismissing a lost one. Before the town has
been reached nothing changes: a win goes to the map list and a loss to the main menu, exactly as
today. A mission whose successor the campaign declares under `AutoGetMission` still opens directly
and does not pass through the town — that is the campaign's own key and it outranks this.

**FR-4 — the state survives the round trip.** Returning from a mission the player still has: his
party and everything each member is wearing and carrying; his gold; which missions are done; which
building offers are consumed; and which missions the tavern has unlocked. The party is carried by
the mechanism that already carries it — this story adds no second answer to what a party is.

**FR-5 — four doors, each with a way in and a way out.** The town square opens the tavern, the
shop, the school and the **gates**. Each states something true about the state of the game when it
is entered, and each returns to the square. No room is an empty rectangle; a room with nothing in
it says what it is and that it is empty.

**FR-6 — three producers, one list, one consumer.** The tavern, the shop and the school each append
to **one** list of available missions owned by the town; the gates are the only thing that reads it
and the only thing that starts a mission. There is exactly one such list and no second answer to
what is available. It is empty when the town is first reached — the campaign hands missions out
through buildings, and nothing is available until one does.

**FR-7 — the shop and the school hand out the head of their list.** Entering shows the chapter's
offer list; the first un-taken element can be taken, which appends its mission to the available
list and consumes it. That is the registry's own mechanism for those two keys. The shop also states
the chapter's own price bounds, which is what else its section carries.

**FR-8 — the tavern is per NPC and it is a conversation.** It lists the NPCs of the chapter,
pairing each with what he has to give. Choosing one opens a conversation: greeting, then what he
wants, then a choice to accept or leave. Accepting appends his mission to the available list and
consumes that NPC's entry so he has nothing more to give. An NPC whose entry holds the no-mission
sentinel says so and stays where he is — he is not consumed and has nothing to accept.

**FR-8a — going on a mission is walking out of the gates.** The gates list the missions available
and not yet won; choosing one starts it with the carried party. This story adds no other way to
start a mission anywhere in the front end.

**FR-9 — winning pays.** Winning a mission adds that mission's own declared payment to the player's
gold, and the town states the total. A mission that declares none pays nothing.

**FR-10 — nothing is written to disk.** The whole of this state lives for the life of the process.
Where a save would attach is named in one place and nothing is built there.

**FR-11 — no graphics.** Every screen is placeholder text in the debug font, drawn and hit-tested
through the map list's own layout so that a row that is drawn is a row that can be clicked.

## Acceptance

**AC-1** From a front end holding a stock campaign, winning mission 20 leaves the player on the
town screen and not on the map list.

**AC-2** In the town at chapter 30, the gates list nothing; after the tavern conversation with the
chapter's NPC is accepted, mission 30 is listed there and can be started.

**AC-3** A party carried into a mission and back out of it arrives in the town with the same
members, the same equipment codes and the same skill experience it ended the mission with.

**AC-4** Losing a mission started from the town returns to the town, with the town's own state —
gold, unlocked missions, consumed offers — exactly as it was.

**AC-5** Each of the four rooms can be entered from the square and left back to it, and the square
can be left.

**AC-6** Taking the shop's head offer twice is impossible: the second visit no longer offers it.

## Properties

**P-1 — the carry is not touched.** No statement in this story changes what a party is, what
crosses into a mission, or what a mission carries out. The threshold for anything reaching hashed
simulation state is High and nothing here reaches it.

**P-2 — the screen tier names no simulation type.** The town crosses into `pkg/ui` as strings,
integers, booleans and callbacks, and by nothing else. `pkg/ui` may import the render tier and no
other, and this story does not change that.

**P-3 — both enums are appended, never inserted.** The screen identifier and the notice
destination are compared and switched on by value; a new member of either goes after the existing
ones so that no existing value moves.

**P-4 — no room can trap the player.** Every screen this story adds has a key that leaves it, and
the square's own leaves the town.

## Authored, and on what grounds

The original's town screen is not decoded and this story opens no research item. What the registry
establishes — the three offer lists, one reader each, the shop and school taking element 0, the inn
pairing per NPC with its own zero sentinel, and the payment key — is used exactly as it stands.
Everything else is **AUTHORED** and disclosed:

- The chapter rule of FR-2. The registry says which section holds which offers; nothing read says
  which section is live in a given town visit.
- The conversation of FR-8 — its shape, its lines and that accepting is a row rather than a key.
  The registry says an accepted inn offer is removed from both arrays; it says nothing about what
  is said.
- That the square is four doors and their order. The **gates** are the owner's own naming
  (2026-08-11) and the world map behind them is his too; what stands there today is the list, and
  that it is a list rather than a map is this story's scaffolding.
- **A disclosed divergence, not a second door:** the main menu's own map list existed before this
  story and still starts a mission directly. This story adds no way to start one except the gates,
  and touches that list not at all; it remains the developer door it has always been, and the
  *campaign* now routes only through the town.

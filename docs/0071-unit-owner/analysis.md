# Analysis — what a trigger hands over, and what it does not

## The premise this story was opened on, and what measuring it did to it

The story was opened as *the missing link in the campaign's first win chain*: on mission 10 the chain
is trigger positions 2 -> 3 -> 4, position 2 is inert, and its action hands a group to the player
through an instant this build does not run. Implement the instant, the reasoning went, and the chain
completes.

**That is not what the map says.** The chain was re-read here from the map's own bytes rather than
taken on report, and the dependency it carries is not the one the premise assumes:

| position | conditions | actions |
|---|---|---|
| 2 | distance(hero, (36,51)) <= 3 **and** population(group 1) == 0 | message; **instant 22** group 2 -> player 1 |
| 3 | distance(unit 21, (56,21)) <= 3 | message; **instant 19** unit 21 -> player 2; instant 28; `slot[50]++` |
| 4 | distance(hero, (66,16)) <= 3 **and** `slot[50] != 0` | message; **win** |

Position 3 reads no register position 2 writes. Its only condition is a distance arm this build
already evaluates, so **the trigger is live today and is evaluated every pass** — it is not inert,
and no instant arm can make it more or less so, because inertness is derived from unimplemented
*checks* alone. Position 4 likewise. What actually stands between position 2 firing and position 3
firing is that unit 21 has to *be* at (56,21), thirty cells by Chebyshev from the (36,51) it is
placed at, and no arm of the script moves it. In the game the hand-over is what lets the **player**
walk it there. In this tree the player can already walk anything: selection and ordering consult no
owner, so the win chain has no ownership gate on it at all.

So instant 22 is **not** the gate. The sibling group-population story reached the same place
independently — its own end-to-end chain test stands the hand-over in by moving the unit directly,
which is only sound because the hand-over is not load-bearing for the win.

What remains between mission 10's start and a firing win, measured rather than assumed:

1. unit 21 within 3 cells of (56,21) — a thirty-cell walk nothing but a player order produces;
2. the hero within 3 cells of (66,16), and the hero reference resolving;
3. `slot[50]` nonzero, which position 3's own instant supplies once it fires;
4. neither unit 21 nor unit 51 dead — position 11's two VIP checks are live and each increments the
   lose counter every pass once its unit is not alive. Both dying inside one pass takes the counter
   from 0 to 2 and no outcome is ever reported.

None of those four is an ownership arm.

## Why the story survives anyway, in a different shape

Because the map's own account of mission 10 is an **escort**, and this tree cannot express one. The
type-5 roster names five owners — `Self`, `Villagers`, `Rogues`, `Beasts`, `Nocturnal` — and every
placement carries one of them; unit 21 is a `Villagers` unit until a trigger says otherwise. Today
that word is decoded by nobody, so every unit on the map is equally the player's, and the mission's
first objective can be walked around by ordering the escortee before finding her.

The story therefore becomes *carry the owner, and run the two arms that change it*. It buys no
trigger and no win — that is stated in the contract rather than hoped for — and it is the state every
later ownership consumer needs.

## What was looked at

**The corpus, both installed roots.** Instant opcode 22 is authored by **14 nodes across 8 maps** and
named by **7 built triggers** on each root; opcode 19 by 14 nodes across 10 maps. Mission 10 itself is
byte-identical across the roots (md5 `2d983ccbf249c5336ebb7ccc41fc405c`), so its reading is made on
both. Four maps do differ between roots — `100.alm`, `140.alm`, `Horror.alm`, `LuMoir.alm` — which is
why every count here is given per root rather than once.

**The owner field.** Over every placed record of every map that opens: **8094/8094** (en, 38 maps) and
**3991/3991** (ru, 34 maps) carry an owner in `[1, type-5 record count]`, observed range `[1,9]`. Only
**9** records in **4** maps are owned by slot 1 on either root — the player owns almost nothing a map
places, which is the same fact as a campaign map placing no hero.

**Which id space a `Target_Player` parameter names** was open and the corpus closed it. Over every
script parameter of that type: **170/170** (en) and **168/168** (ru) fall inside `[1, type-5 count]`,
while reading them as the type-5 record's own id word fails on **26** of them on each root — e.g. a
map whose roster ids are `6 9 10 4` carries thirteen parameters naming player `1`. So a script's
player parameter and a placement's owner field are **the same 1-based slot space**, and the arms are a
straight assignment rather than a translation.

**The ledger** was read for what an ownership change does at runtime before anything was specified.
`ALM-OWN-039` fixes the field and its id space; `PARTY-OWN-001`, `PARTY-ROSTER-002` and `AI-GROUP-009`
fix what an owner *is* at runtime — a `Player` the actor points at, whose group collection contains
the actor — and `MISSION-M10-009` reads this map end to end and renders both arms. `UNIT-OWNER-009`
and `MOVE-TERM-003` name the one movement consumer that already exists.

## What is not established, and is left open rather than guessed

The **instruction-level body of either arm** is not published. Their effect is known from a shipped
catalogue rendering of one whole map at High confidence, and from the parameter grammar; what is not
known is whether the original re-keys group membership, what it does with a group identifier that two
owners share, and what it writes besides the actor's owner pointer. The published instant-table row
names four arms' effects exactly and grades five others Medium; neither 19 nor 22 is among either set.
This story specifies the only reading a tree with no group object can express, and says so in its
contract.

The second open thing is **which roster slot the human participant is**. The evidence says a human
participant's `Player` is created by the session-join path and not by the map, and that the
discriminator every "is this mine" consumer reads is a field of that object rather than a slot number.
Nothing here settles it, so nothing here reads an owner.

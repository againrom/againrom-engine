# 0159-join-persistence — spec

A player-character-band unit the mission script hands to the player joins the party and stays in it
after the mission ends. A low-TypeID Human is mission-only and is culled by the same boundary.

This spec is self-contained. Research provenance is in `provenance.md`.

## Terms

- **Server type id** — an integer every actor carries. A zero-mode Humans placement preserves its
  table value. Only an exact `Hero` NPC constructor and a party-member mint overwrite it into
  `[0x21,0x40)`. A creature carries its row's id; an unresolved actor carries zero.
- **The band** — the interval `[0x21,0x40)`.
- **Hand-over** — script instant 19 (one unit) or 22 (one group), the two acts that change an
  actor's owning roster slot.
- **The human participant** — the roster slot the player's own party stands on.
- **Roster member** — an entry in the party a mission is started with and a mission ends with.
- **Primary character** — the one roster member the mission-entry gate tests for. A party has at
  most one.

## FR-1 — construction mode decides whether a Human receives a band type id

A zero-mode Humans placement preserves its row's TypeID. The npc placement arm requests the
player-character overwrite only when its scenario section carries the exact `Hero` token. A party
member minted by the campaign also carries a band TypeID. A creature carries its class-row id, and
an unresolved placement carries zero.

The value is canonical state: it is written to the byte form, restored from it, and enters the
digest. The byte layout does not change, so no format version is consumed.

The death-gold roll is gated at `> 0x40` and is therefore unreachable for any value inside the band.
A person minting a band type id drops no gold he did not drop before.

## FR-2 — a hand-over moves the actor and gives it a group of its own

A hand-over writes the named actor's owner to the named roster slot, as before, and in addition:

1. The actor leaves whatever group it stood in.
2. The actor is placed alone in a group no actor in the world already stands in and no script node
   names.
3. The actor's command-group overlay is cleared, so no order issued before the hand-over survives
   it.

Instant 22 names a group. Every actor in that group is handed over, and each of them arrives in a
group of its own — a group of three becomes three groups of one, not one group of three. The
membership is read once before the first write, so the loop's own writes do not change which actors
it visits.

A hand-over that names no roster slot, or names a group or unit this world does not hold, changes
nothing at all — neither owner nor group.

A hand-over does not make the actor a primary character. Nothing in this build's hand-over path
writes the field the mission-entry gate tests, and no roster member this story produces sets it.

## FR-3 — the boundary keeps the human participant's band survivors

At the end of a won mission, the actors that cross into the next mission are exactly those that:

1. are owned by the human participant, and
2. are alive, and
3. carry a server type id inside the band.

An actor failing any of the three does not cross. This is applied to every actor the world holds,
not only to the ones the mission was started with, so an actor that arrived by hand-over is judged
by the same three tests as one that walked in with the party.

The answer is in ascending entity-id order, so it does not depend on the order the world stores
actors in.

## FR-4 — a band survivor that was not a roster member becomes one

A band survivor whose entity was not minted from a roster member is appended to the party the
mission ends with, after the members that walked in, in ascending entity-id order.

He carries:

- his pack, exactly as the mission left it;
- his twelve worn slots, exactly as the mission left them;
- his six skill experience integers and his six skill levels, exactly as the mission left them;
- the four statistics, the profile and the spellbook of the row his placement resolved to.

He is a persistent roster member and not a primary character: he is not temporary, he is not the
starting hero, and he is not a mercenary.

He has a stable identity derived from the runtime id he carried in the mission he joined in, so the
same companion is the same roster entry in every later mission. A companion whose placement resolved
through the scenario npc arm additionally carries that npc record's own subscript, which is what the
town screen and the localized-name lookup already key on.

Nothing is invented for him. Where his placement resolved to no person row he is not appended at
all, because there would be no statistics to mint him from on the next map.

## FR-5 — persistence is unbounded

A joined companion carried into mission N+1 is a roster member of that mission, is minted there like
any other roster member, and is carried out of it by the same rule. Nothing marks him as having
already crossed once, and no path removes him at a later boundary that would not equally remove a
member who started the campaign.

Losing a mission carries nothing: the party after a loss is the party that walked in, which is the
rule this build already applies to every member.

## Acceptance

- **AC-1** — An exact-Hero NPC placement is in the band; an otherwise identical low-TypeID Humans
  placement preserves its row value and stays outside it. (FR-1)
- **AC-2** — A minted party member's type id is inside the band, and the death-gold roll pays zero
  for him at every gold chance. (FR-1)
- **AC-3** — Instant 22 over a group of three under one owner leaves all three under the new owner,
  in three distinct groups, none of them the old group, and none of them equal to another's. (FR-2)
- **AC-4** — An actor carrying a command-group order that is handed over has no command group after
  the hand-over. (FR-2)
- **AC-5** — Instant 19 naming an entity the world does not hold changes no owner and no group.
  (FR-2)
- **AC-6** — Over a world holding, under the human participant, one live band actor, one dead band
  actor, one live out-of-band actor, and under another slot one live band actor, the boundary
  answers exactly the first. (FR-3)
- **AC-7** — A companion handed over mid-mission appears in the party the mission ends with, holding
  the items his entity held and wearing what his entity wore. (FR-4)
- **AC-8** — That companion's statistics are the row his placement resolved to, not zeroes: the
  party he is in mints him on the next map with the same maximum health his row derives. (FR-4)
- **AC-9** — He is not the starting hero, not temporary, and carries no mercenary type. (FR-4)
- **AC-10** — Winning two missions in a row leaves the companion in the party after the second, with
  the same stable identity he had after the first, and exactly once. (FR-5)
- **AC-11** — A joined companion whose placement resolved to no person row does not enter the party.
  (FR-4)
- **AC-12** — Mission 40's own script, run against its own map, hands the paladin group to the human
  participant and the mission-end boundary reports him as a survivor. (FR-2, FR-3, FR-4)
- **AC-13** — Mission 20's Sarindar and three NPC14_1 guards retain TypeIDs `0x17` and `0x0a` after
  transfer and do not enter the town roster. Later tavern Clubmen are fresh stock. (FR-1, FR-3)

## Properties

- **P-1** — Which actors the boundary reports does not depend on the order the world stores entities
  in.
- **P-2** — The hand-over is idempotent in its owner write: handing the same actor to the same slot
  twice leaves it under that slot. It is not idempotent in its group write, and it is not required
  to be: the second hand-over allocates a second fresh group, exactly as the original's routine
  allocates one per call.
- **P-3** — A party that receives no hand-over crosses a boundary exactly as it did before this
  story. Every existing carry behaviour is unchanged.
- **P-4** — Nothing this story adds writes to a party member on the loss path.

## Disclosed divergences

- **DIV-1 is retired.** The build now preserves zero-mode Humans TypeIDs and requests the band
  overwrite only for an exact-Hero NPC or a party mint, matching the decoded constructor rule.
- **DIV-2** — The eight reset fields. The original keeps a surviving actor object across the
  boundary and resets eight of its fields in place, among them restoring each pool from its own
  maximum. This build destroys the world at the boundary and mints a fresh actor on the next map
  from the roster member, which arrives at full health and full mana by construction. A player sees
  the same thing: a companion who ends a mission wounded starts the next one whole. What a reader
  loses: any field the original resets that this build's mint does not reproduce would be invisible
  here, and this build cannot name such a field because it does not carry the eight offsets.
- **DIV-3** — The joined companion's display name is his placement's template name, not a localized
  person name, unless his placement took the scenario npc arm and the install ships a name for that
  record. Nothing decoded says what name the original shows.

## Out of scope

- Reading a joined companion out of an owner-produced save file.
- Drawing a joined companion's portrait from anything other than his class record.
- The `AddHero` town-view mechanism, which is a different act and already built.
- Removing a companion from the party. Nothing decoded describes one.

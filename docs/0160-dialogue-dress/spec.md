# 0160-dialogue-dress — live dialogue speakers wear their own equipment

## Problem

A dialogue draws the speaker's picture in a pane. A live actor must appear in the equipment that
actor actually wears. A speaker for which no live actor exists is a different arm: the original
constructs a portrait drawable with twelve empty equipment slots. Hotfix `00ddc4fd` restores that
distinction after the owner observed a tavern keeper and a school quest giver wearing invented
Humans-row equipment.

## Terms

- **Speaker record** — one `npc<n>` section of the scenario NPC registry. It carries a `Flags`
  token list, an optional `Face`, an optional `Picture`, an optional portrait window origin and an
  optional `DataBinID`.
- **Speaker predicate** — the terms a speaker record's `Flags` list states, one term per token
  present, combined with AND.
- **Live speaker** — an actor in the current world that satisfies the speaker predicate.
- **Synthesised speaker** — the answer when no live actor satisfies the predicate.
- **Worn set** — the twelve equipment slots, slot 6 being the head slot.
- **Outfit row** — a `Humans` collection row whose ten equipment cells state a worn set.

## Functional requirements

**FR-1.** A dialogue figure is composed through the world-figure composer. A live speaker supplies
the actor's own equipment. A synthesised speaker supplies the fixed empty set established by
`DLG-SPEAKER-022`.

**FR-2.** The speaker is resolved before any registry picture kind is chosen. Candidates are the
map's placed persons in ascending entity id, followed by the mission party in start order. A
candidate is considered only while it is alive. The first candidate satisfying the speaker
predicate is the live speaker. A companion carried into a later mission therefore remains a live
speaker even when that later map has no placement for him.

**FR-3.** The speaker predicate is evaluated term by term over the tokens the record's `Flags`
states:

| Token | Term |
|---|---|
| `Hero` | the candidate is a Hero-mode scenario actor or a persistent party player |
| `Human` | the candidate belongs to the live speaker population: a person placement or any mission-party actor |
| `Mage` | the candidate's figure directory is a mage directory |
| `Female` | the candidate's figure directory is a woman's directory |
| `Me` | the party candidate marked as the starting hero |
| `MySex` | the candidate's sex bit equals the player's |
| `MyClass` | the candidate's class bit equals the player's |
| `!Hero`, `!Human`, `!Mage`, `!Female`, `!Me`, `!MySex`, `!MyClass` | the negation of the above |
| `Face` | the record's `Face` equals the candidate's face byte |
| `Picture` | the record's `Picture` equals the candidate's type id |
| `Platoon` | not evaluated; states no constraint |

A record stating no token at all states no constraint, and the first live candidate answers it.

**FR-4.** A live speaker's figure is composed from the worn set the world holds for that actor at
the moment the picture is asked for, so equipping an actor changes what its dialogue figure wears.

**FR-5.** When no live actor matches, the record's picture kind chooses the fallback. A synthesised
figure is bare: its directory and sheet stay the record's own `Flags` and `Face`, and its equipment
half is twelve empty slots. A flat portrait remains a flat portrait. A no-picture record retains
the existing composed-player or named-companion fallback. A joined `DataBinID` or archetype row is
definition data, not a hero instance, and does not materialise its starting outfit onto a fallback
figure.

**FR-6.** No layer is excluded because the picture is going to a dialogue. The head slot is
composed whenever the resolved worn set fills it.

**FR-7.** The town screen resolves speakers the same way. It holds no world, so every tavern,
school and shop NPC using the figure arm is a bare synthesised speaker under FR-5. Its no-picture
party/player records retain the composed-player and named-companion fallbacks.

## Acceptance criteria

**AC-1.** A speaker record naming a figure, with a live candidate wearing items, composes with that
candidate's item codes and not with an empty set.

**AC-2.** Mission 40's `npc25` resolves to the placed paladin, and the worn set the composition
receives fills slot 6.

**AC-3.** A `Mage` record does not resolve to a fighter candidate, and a `Female` record does not
resolve to a male candidate.

**AC-4.** A `Face` record resolves only to a candidate whose face byte equals the record's `Face`.

**AC-5.** A dead candidate is not the speaker, and a live candidate later in id order is.

**AC-6.** A record with no live candidate composes with twelve empty slots even when its joined
Humans row states armour, including the head slot.

**AC-7.** The same empty-set rule covers a record with no `DataBinID`, a record whose row is empty,
and the town's no-world resolver. No archetype fallback dresses the portrait.

**AC-8.** Changing a live speaker's equipment changes the picture the dialogue returns.

**AC-9.** A record naming a creature portrait with no matching live person still loads the flat
portrait at its own tier.

**AC-10.** A no-picture record with no matching live actor still returns its composed-player or
named-companion fallback.

**AC-11.** Mission 70's no-picture `npc23` resolves to the live Naira placement and composes her
current bow and armour rather than the current inventory subject.

**AC-12.** Brian carried from mission 40 into mission 70 resolves as the live party actor and
composes his current equipment although mission 70 has no `npc25` placement.

## Properties

**P-1.** Nothing here reaches `pkg/sim`. No world field, byte form or digest changes.
`formatVersion` stays 46.

**P-2.** Resolution is a pure function of the record, the candidate list and the worn sets. It reads
no clock and no file.

**P-3.** Every failure is an absence. A speaker number naming no record, a record whose class this
bundle does not hold, and an install missing a sheet all answer "no picture" and draw the pane
empty. None is an error; no definition-table outfit lookup is required by the synthetic arm.

**P-4.** The composed picture is cached by figure and worn set together, so two speakers of one
figure wearing different things are two pictures and one speaker who re-equips is a third.

## Cut from this story

The two below are numbered `Cut n` and not `SC-n`: an `SC` id is a plan-level success criterion
and naming a scope cut with one puts it in a namespace the audit reads from plan.md.

**Cut 1.** Whether the original hides the helmet on a live matched actor is EXP-0165's question and
is not answered here. A synthesised actor has no helmet to hide.

**Cut 3.** The `Platoon` term is not evaluated.

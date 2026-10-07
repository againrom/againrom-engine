# 0160-dialogue-dress — plan

## Shape

Four tiers change. `pkg/data` learns the whole token list a speaker record states. `pkg/mapload`
gains one exported resolution: the worn set a speaker record's own outfit row states.
`pkg/game` gains the candidate table and the predicate, and the dialogue face resolver stops
composing with an empty worn set. `pkg/ui` is untouched — it still receives a picture and nothing
else.

## Steps

**Step 1 — `pkg/data`: the record carries its tokens.** `LoadNPCFaces` today reads three tokens and
decides an arm. It now records every token `DLG-SPEAKER-023` names, negations included, as a
bitset on `NPCFace`, and stores the record's `Picture` value whenever the key is present rather
than only on the portrait arm. Adds `FigureDir.Female` beside the existing `Mage`. Covers FR-3.

**Step 2 — `pkg/mapload`: the outfit row.** One exported function answers "what does the row behind
this speaker record dress him in", taking the npc subscript and the record's two figure axes. It
resolves the section's `DataBinID` through the lookup a placement already uses, falls back to the
archetype row `data.ChargenBase` names — on a missing id and on a row whose cells are all empty
alike — and runs the row's ten cells through the same `wearRow` and `startingLoadout` a placed
person's row runs through. Covers FR-5.

**Step 3 — `pkg/game`: the candidate table.** A walk over the map's placements, beside
`entityFigures`, recording for each placed person the figure axes, the row's face byte and the row's
type id. Built once when a map opens, keyed by entity id, read by nothing that writes. Covers FR-2.

**Step 4 — `pkg/game`: the predicate and the composition.** The speaker resolver gains the candidate
list, a liveness test, a worn-set reader and the player's own two bits. `SpeakerFace`'s figure arm
resolves a speaker, composes a live one from the world's own worn set for that entity, and composes
a synthesised one from the outfit row. Covers FR-1, FR-4, FR-6, FR-7.

**Step 5 — wiring.** `openMission` fills the candidate table, the liveness test and the worn-set
reader from the driver it already holds; the town screen fills the outfit table from the front end's
own table and leaves the candidate list empty. Covers FR-7.

## Design decisions

**DD-1. The predicate lives in `pkg/game`, not in `pkg/data`.** A term reads a live actor's state.
`pkg/data` holds no actor and no world, and a predicate split across two tiers would put the token
list in one place and its meaning in another.

**DD-2. The record carries tokens, not answers.** `NPCFace` already carries the resolved picture
arm rather than the columns. The tokens are different: which of them narrows a candidate is a fact
about the actor, so the record hands over the token set and the predicate does the reading.

**DD-3. `Hero` and `Human` initially tested one property.** The later live-party hotfix supersedes
that approximation. `Human` still means any live person; `Hero` is the scenario registry's exact
Hero mode for a placement and persistent-player mode for a party actor. Mission 70 makes the split
observable: without it `npc23` selects an ordinary female fighter before Naira.

**DD-4. `Me` was initially false for every candidate.** The original story limited candidates to
map placements. The later live-party hotfix supersedes that cut: the candidate population now also
contains the mission party, `Me` selects its explicit `StartingHero`, and `!Me` excludes that actor.

**DD-5. The worn set is read at call time, not cached on the candidate.** `DLG-SPEAKER-022` states
that a live speaker's slots change by re-send and that the figure follows. The candidate carries
only identity; the worn set is read from the world when the picture is asked for.

**DD-6. The outfit table is resolved once per map, keyed by npc subscript.** It is a fact about the
registry and the definition table, not about the world, so it is resolved where the table is already
in hand and is never re-read per frame.

**DD-7. The archetype fallback reuses `data.ChargenBase`.** It is the one place this tree turns a
`(mage, female)` pair into a shipped `Humans` row, and it already returns the collection index its
equipment cells live on. A second selection here could disagree with the one a generated character
gets.

**DD-8. `startingLoadout` is applied with the row's own name.** A synthesised speaker then wears
what a placement of that row would wear, including the two-handed displacement, rather than a
second reading of the same cells.

**DD-9. Nothing enters `pkg/sim`.** The candidate table, the outfit table and the composed picture
all live beside the world, on `tiers`' own precedent. `formatVersion` stays 46.

## Traceability

| Requirement | Step | Witnessed by |
|---|---|---|
| FR-1 | 4 | AC-1, AC-8 |
| FR-2 | 3, 4 | AC-2, AC-5 |
| FR-3 | 1, 4 | AC-3, AC-4 |
| FR-4 | 4 | AC-8 |
| FR-5 | 2, 4 | AC-6, AC-7 |
| FR-6 | 4 | AC-2, AC-6 |
| FR-7 | 5 | AC-6 |

## Success criteria

**SC-4.** `go build`, `go vet`, `gofmt` and `go test -trimpath -count=1 ./...` are clean, and every
`scripts/check-*.sh` passes.

**SC-5.** The two script-gap census numbers for missions 10 and 20 are recorded against the
baseline `pipeline/milestone-baseline.txt` carried.

**SC-6.** A test over a synthetic fixture reproducing mission 40's decoded join drives it end to
end: the placed paladin is the speaker `npc25` resolves to, and the worn set handed to the composer
fills slot 6. The fixture is built in test code from the format contracts, never read from an
install.

**SC-7.** A developer tool reports, against a lawful install, how many of the campaign's speaker
records this build dresses and how many it leaves bare.

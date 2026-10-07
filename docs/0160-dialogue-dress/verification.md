# 0160-dialogue-dress — verification

Branch `0160-dialogue-dress`, merged up to master `4b6e4d7`. Research pin `e1b27fd`; the five DLG
claims this story reads are unchanged between the `733bdc9` pin the story opened on and this one,
re-read through `go run ./tools/claim` after each merge.

No `tasks.md`: the slice was implemented in one lane context, so every FR and DD is accounted for
below instead.

## Gate

Run in the lane worktree on a clean tree, after the second merge of `origin/master`:

```
go build ./...                     clean
go vet ./...                       clean
gofmt -l $(git ls-files '*.go')    printed nothing
go test -trimpath -count=1 ./...   all packages ok
scripts/check-no-game-assets.sh    PASS
scripts/check-doc-budget.sh        PASS
scripts/check-sdd-audit.sh         PASS (one note: no tasks.md)
scripts/check-sdd-audit-selftest.sh PASS
scripts/check-hotfix-ledger.sh     PASS
```

## SC-4 — every gate

Above. `scripts/check-*.sh` was run by glob, five scripts.

## SC-5 — the script-gap census

`go build -o /tmp/mr ./cmd/missionrun`, then
`AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission N -trace -ticks 1 | grep -c UNSUPPORTED`:

| Mission | Baseline (`pipeline/milestone-baseline.txt`) | This branch |
|---|---|---|
| 10 | 17 | 17 |
| 20 | 11 | 11 |

Unchanged, as expected. This story changes figure composition and speaker resolution; it adds no
script opcode and removes none. The result this story owes is the one below, not a falling census
number.

## SC-7 — the speaker census, against both lawful installs

`wearcheck -speakers` walks every `npc<n>` section that composes a figure and reports the starting
outfit its joined Humans row would carry. Run as `$env:AGAINROM_ASSETS=...; wearcheck.exe
-speakers`. The census explains the visible regression hotfix `00ddc4fd` removes; it no longer
defines the runtime synthetic-speaker arm.

| Root | Figure records | Dressed | With slot 6 filled | Bare |
|---|---|---|---|---|
| EN | 53 of 105 | 53 | 8 | 0 |
| RU | 53 of 105 | 53 | 8 | 0 |

At the story's original landing, all 53 potential outfits were materialised onto a speaker whenever
no live actor matched. The 2026-08-24 owner report showed that decision on the tavern keeper and the
school quest giver. The current runtime ignores this table on the synthetic arm and passes
`data.Equipment{}` exactly as `DLG-SPEAKER-022` specifies. The report remains a read-only corpus
join and does not fail merely because a row is bare.

**Independent confirmation of `DLG-DRESS-024`.** The tool reports `npc25 -> row=PC_Paladin`, worn
slots 1, 6, 7, 8, 9, 10 and 12. The claim reads the same seven slots off the shipped `Humans` row
through the `40.alm` placement arm. This build reached them through the registry section's
`DataBinID` and this tree's own `wearRow`, which reads none of the claim's evidence.

The seven rows with ten empty equipment cells and the former archetype fallback remain useful data
facts, but neither is a portrait rule. A definition's starting equipment applies when a person is
materialised as a person, not when the dialogue system synthesises its bare fallback drawable.

## SC-6 — mission 40's decoded join

`TestMission40SpeakerIsThePlacedPaladinInHisHelm` (`pkg/game/speakerdress_test.go`) drives it over a
synthetic fixture built in test code: a placement below the units floor with the npc flag naming
subscript 25, a section `npc25` carrying `DataBinID = 42`, and a `Humans` row at server id 42 named
`PC_Paladin` wearing a plate helm whose own `Armors` row states slot 6. The test asserts the
candidate list holds exactly the placed person, that his id is the placement's own index, that
`SpeakerOutfit` names `PC_Paladin`, and that the worn set the composer receives fills slot 6. It
reads no install.

## Requirements

| Id | Where it landed | Witness |
|---|---|---|
| FR-1 | `pkg/game/speakers.go` `speakerFigure`, `liveFigure` | `TestDialogueSpeakerComposesTheLiveActorsWornSet` |
| FR-2 | `pkg/game/speakeractors.go` `missionSpeakers`, `speakerCast.resolve` | `TestMission40SpeakerIsThePlacedPaladinInHisHelm`, `TestMissionSpeakersIncludesThePartyWithStableIdentity`, `TestDialogueSpeakerSkipsTheDead` |
| FR-3 | `pkg/data/npcface.go` `npcTokens`; `speakerCast.matches` | `TestNPCFacesReadsEverySpeakerToken`, `TestDialogueSpeakerPredicateNarrowsByToken` |
| FR-4 | `speakerCast.worn`, bound to `mapWorld.equipmentOf` per call | `TestDialogueSpeakerComposesTheLiveActorsWornSet` (the re-equip half) |
| FR-5 | `pkg/game/speakers.go` `SpeakerFace`, picture-kind fallback | `TestSynthesisedSpeakerUsesTheBareFaceSheet` |
| FR-6 | no layer filter anywhere on the path; slot 6 composed from the worn set | `TestDialogueSpeakerComposesTheLiveActorsWornSet` asserts slot 6 |
| FR-7 | `pkg/game/townscreen.go`, a resolver with an empty cast and no outfit table | `TestSynthesisedSpeakerUsesTheBareFaceSheet` (the cast is empty, which is the town's state) |

| Id | Verdict |
|---|---|
| AC-1 | PASS — the cache key carries the live actor's item codes. |
| AC-2 | PASS — SC-6 above. |
| AC-3 | PASS — the `Mage` and `Female` rows of `TestDialogueSpeakerPredicateNarrowsByToken`. |
| AC-4 | PASS — the two `Face` rows, one matching and one naming nobody's face. |
| AC-5 | PASS — `TestDialogueSpeakerSkipsTheDead`. |
| AC-6 | PASS — `TestSynthesisedSpeakerUsesTheBareFaceSheet` starts from rows that would supply armour and requires twelve empty composition slots. |
| AC-7 | PASS — the same test covers a named row, an archetype fallback and an outfit-less row; all three stay bare. |
| AC-8 | PASS — removing the helm produces a different cache key and an unoccupied slot 6. |
| AC-9 | PASS — `TestSpeakerFaceAnswersTheThreeKinds` and `TestSpeakerFaceCarriesTheWindowAndTheTier` keep the unmatched portrait arm. |
| AC-10 | PASS — `TestSpeakerFaceAnswersTheHeroFromTheInventorySubject` and `TestSpeakerResolverUsesTheAddedCompanionsOwnFigure` keep the unmatched no-picture fallbacks. |
| AC-11 | PASS — `TestLiveSpeakerPrecedesEverySyntheticPictureKind` covers no-picture precedence; the EN/RU release witness composes mission-70 `npc23` from Naira's live figure and equipment. |
| AC-12 | PASS — `TestMissionSpeakersIncludesThePartyWithStableIdentity` covers the carried population; the EN/RU release witness carries mission-40 Brian into mission 70 and composes `npc25` from his live equipment. |
| P-1 | PASS — no file under `pkg/sim` is touched. The hotfix removes the derivable `speakerOutfits` cache; world bytes, hash and format are unchanged. |
| P-2 | PASS — `speakerCast.matches` and `resolve` read only their arguments and the cast's own fields; `SpeakerOutfit` reads only the table. |
| P-3 | PASS — the synthetic arm needs no table; `TestSynthesisedSpeakerUsesTheBareFaceSheet` and `TestSpeakerFaceIsTotalWithNoTable`. |
| P-4 | PASS — the key is `{figureID, data.Equipment}`; AC-8 shows two worn sets giving two entries. |
| Cut 1 | Held for live matched actors. The synthetic arm has no helmet to hide. |
| Cut 2 | Superseded by the live-party hotfix after the owner reproduced a bare carried Brian and Naira borrowing Danath's picture. |
| Cut 3 | Held. `Platoon` is read into the token set and evaluated by no term, asserted by the `Platoon` row of the same test. |

| Id | Verdict |
|---|---|
| DD-1 | Carried. The token set is on `data.NPCFace`; every term is evaluated in `pkg/game`. |
| DD-2 | Carried. `LoadNPCFaces` stores tokens verbatim and interprets none. |
| DD-3 | Superseded. The live-party hotfix separates `Hero` from `Human`; the mission-70 release witness requires the distinction before Naira can win the predicate. |
| DD-4 | Superseded. `Me` now selects the party's explicit `StartingHero`; `!Me` excludes it. |
| DD-5 | Carried. `speakerActor` holds no worn set; `speakerCast.worn` is called at picture time. |
| DD-6 | Superseded by hotfix `00ddc4fd`; no synthetic outfit table is built or cached. |
| DD-7 | Superseded for dialogue runtime; `SpeakerOutfit` remains only a corpus/data join. |
| DD-8 | Superseded for dialogue runtime; ordinary person materialisation retains its own loadout rule. |
| DD-9 | Carried. See P-1. |

## What was not verified

**Nothing was watched on a screen.** No windowed game was launched and no dialogue was looked at.
The evidence above is the test suite, which reads the composition input rather than pixels, and the
two developer-tool runs against the lawful installs. A composed figure that is correct at the cache
key and wrong at the paint step would not be caught by any of it.

`DLG-SPEAKER-022` grades as **Unknown** which arm a given shipped dialogue takes at run time, so no
evidence here says how many of the 53 records resolve to a live actor in play. Both arms are now
correct independently: a live actor carries its live worn set; a synthetic speaker is bare. The
owner's tavern and school screenshots are the player-visible witness for the corrected arm; this
hotfix did not launch the game.

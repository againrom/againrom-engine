# Story `1044` — closure

As-built evidence. `spec.md` owns behaviour; this file owns aspect closure, shipped witnesses,
research reconciliation and the remaining review surface.

Base `d5b76f4c`, production commits `89229421`, `d8db7d57`, `80c0bf75` and `addf6b88`; merged
masters `75fe35f9`, `220bf4c2`, `d3049be7`, `1f62579c` and `b4e119ef`; merge commits
`6a50ad62`, `bfde5379`, `57012c63`, `8e795537` and `0e381ab3`; research pin `23daf74f`.
Simulation form 61 remains current.

## Result

An original save now replaces fresh campaign defaults before town, mission-list or world-map
consumers run. The owner EN save `game0020.sav` loads main 50, selected 41, retained children 41 and
51, and `InnMission=[0,50,51]`. Completing 41 removes only child 41, drains its type-10 unlock and
returns to main 50. Missions 30 and 40 are below restored progress and cannot reopen. The one live
party member with stable identity `player:brian` remains the only Brian; no Brian-specific dedup
rule was added.

Restored main 10 and 20 remain pre-town. Defeat returns to the menu, an off-map save is refused,
and main 20 opens the town only when completion advances to the campaign's mission-30 boundary.
Candidate arrays and an active mission are accepted only when they join the restored main or a
retained child. A mission selected after a zero-marker import keeps its MapPoint and marker latch
through an againrom save and restore.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `CampaignProjection` decodes every established field in serializer order, ignores the two unnamed top-level scalars, bounds every collection and rejects malformed flags, document kinds and marker strings, including a zero-length field and a one-NUL-byte empty picture. Returned values are detached. |
| Runtime state | PASS | `campaignProgress` replaces the fresh main, children, candidates, mercenary state, documents, selection, MapPoint relation, mission time and marker cache. Its open state is derived from the restored main and the campaign boundary. |
| Simulation | N-A | Campaign projection is game-layer state. The inner binary, digest and upgrade paths remain form 61. |
| Player input | PASS | Town acceptance mutates paired or single candidates before the announcement latch. School and shop production screens consume their restored candidates. Mission selection updates and persists selected and marker state. Lower-main open is refused before map I/O. |
| AI | N-A | No AI state or decision route changes. |
| UI/HUD | PASS | Existing town and world-map consumers read the restored chapter, offers, availability, payment, shop bounds, mercenaries, documents, selected point and marker-cache presence. The original-save row and headless disclosure say that campaign loads while the mission world restarts. |
| Triggers/scripts | N-A | World-half script registers and trigger latches are outside the campaign tail and unchanged. |
| Inventory/equipment | PASS | The importer does not rebuild or deduplicate party items. The owner save retains Brian's stable identity, twelve worn slots and carried state from story `1040`. |
| Persistence/save-load | PASS | The original fixed record parses atomically. Againrom save/load round-trips the typed projection and separate marker-selection latch through additive gob fields. Dead candidates, active-location mismatches and pre-town town-only states are refused before front-end mutation. `DIV-417` records the authored envelope location. |
| Campaign/session | PASS | Side completion, main advancement, child aging, candidate reload, one-shot grants, permanent unlocks, document append and selection all proceed from restored state exactly once. Town openness is derived from `Campaign.TownBegins`. |
| Shipped content | PASS | The owner EN discriminator and independent EN/RU controls load through the production importer. The two controls carry different authored main, selection, latches, documents and mission time. |
| Interactions with existing mechanics | PASS | Fresh campaign and legacy `.ags` paths keep their defaults. Shop generation, town arrival, world-map position, party carry and mission completion consume the restored overlay without changing world-half import. |

No in-scope aspect is `GAP`.

## Exact shipped witnesses

The fixed EN witness printed before completion:

```text
main=50 selected=41 children=[41 51] announced=[41]
inn=[0 50 51] shop=[] school=[] documents=4 markers=[] mission-time=1789
```

It printed after completion:

```text
main=50 selected=50 children=[51] announced=[] inn=[0 50 51] available=[]
party 44 stable=player:brian ... worn [8486 0 0 0 0 9770 10032 10291 10550 10810 0 11326]
```

The completion result was `next=0 offered=50`; the production opener guard refuses mission 40 at
restored main 50 before archive I/O. The output contains one `stable=player:brian` row and no second
identity.

The same load seam prints:

```text
ORIGINAL SAVE: party, positions, stats, items, explored map and campaign load -- mission world restarts
```

The unrelated EN `game0014.sav` control read main and selected 20, no children, announcement 20,
three documents and mission time 217. Completing it paid the authored `+500` transition reward,
rebuilt main 30 from the EN registry and appended child 31. The unrelated RU `game0001.sav`
control read main and selected 10, no children, no announcement, three documents and mission time
0. Completing it advanced directly to main 20.
These unequal values come from each lawful save and registry; neither route contains owner-fixture
constants.

## Research reconciliation

- `SAV-CAMPAIGN-076` supplies the complete field identities and serializer order. The two unnamed
  top-level scalars remain absent from canonical state.
- `SAV-CAMPAIGN-077`..`082` supply monotone main progress, selected state, candidate versus
  announcement state, destructive acceptance, side removal and child aging.
- `SAV-CAMPAIGN-083`..`085` supply mercenary state, consumed `AddHero` and pair-deduplicated
  documents. The implementation adds no Brian rule.
- `SAV-CAMPAIGN-086`..`088` supply marker-cache role, the exact owner-save prediction and the
  lower-main guard that prevents Brian's handover from repeating.
- `SAV-CAMPPOS-072` closes the original-save half of `DIV-137`. `SAV-CAMPMARK-073` establishes the
  marker wire while leaving its three numeric fields unnamed; `DIV-418` carries that remaining gap.

`DIV-417` records the additive `.ags` projection location. `DIV-418` records the non-empty marker
mapping; the separate `.ags` marker-selection latch assigns no meaning to those original dwords.
`DIV-026` now records only the original world-half debts, not a campaign loss that no longer occurs.
No concurrent unreviewed research is consumed and no new research claim id is allocated.

## Remaining surface

The story's production surface is exhausted across parser framing and malformed-field classes,
including zero-count and sole-NUL marker pictures;
record, candidate and active-location validation; pre-town, defeat, save and boundary-entry routes;
original import and disclosure; tavern, school and shop consumers; mission completion; first and
selected MapPoint plus marker producer-to-screen continuity; and againrom persistence. The only
known open surface is the evidence-limited mapping of a non-empty original marker record, recorded
as `DIV-418`; all preserved saves carry zero markers, so no lawful shipped discriminator is
available. World-half continuity remains explicitly outside this contract.

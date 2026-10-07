# 1337 — FrontEnd town screen

## Result

`townScreen` no longer holds a `*FrontEnd`. It holds the components it reads
and writes, plus one service. Every operation of the screen runs over a bare
`CampaignSession`, `InstallResources`, `RuntimeServices`, `Presentation` and
`PersistenceContext`, so the town's shop, tavern and campaign logic is tested
with no FrontEnd. Player-visible behaviour, world hashes, SAV bytes and
scenario hashes are unchanged; this is a refactor.

## Intent

Owner direction: FrontEnd's operations move to the components that own their
state. The top level calls named operations instead of editing each system's
internals. Campaign logic is testable without building a whole FrontEnd.
Passing the whole `*FrontEnd` into a component does not meet it.

This story continues 1332, 1333 and 1336. Their open debt named
`townScreen` holding a `*FrontEnd`, the art setters of `loadMap` and the
`openCityBase` and `continuity` ports.

## Authority

No ROM1 behaviour is touched. No divergence row is added or changed.

## As built

### Ports of the screen (`pkg/game/townscreen.go`)

| field | type | what the screen uses it for |
|---|---|---|
| `sess` | `*CampaignSession` | town, shop, carried party, city history |
| `in` | `*InstallResources` | table, archives, words, bodies, units, campaign, install art, tip font |
| `rt` | `*RuntimeServices` | sound, speech and ambient players, presentation clock and draws |
| `pr` | `*Presentation` | first-use art caches, bird and star counters |
| `pc` | `*PersistenceContext` | the stored tip preference |
| `openMission` | `func(int) ui.MapOpener` | opens a mission screen |

All six are nil together on a screen bound to no game; every former
`t.f == nil` guard is `t.sess == nil`. `(*FrontEnd).bindTown` is the only place
the front end hands its components to the screen.

### Operations moved onto components

| operation | was |
|---|---|
| `(*Presentation).shopArt`, `tipArt`, `documentFont`, `worldMapAssets` (taking the install) | bodies of the same-named FrontEnd methods, which remain as one-line calls |
| `(*PersistenceContext).tipsOffNow`, `setTipsOff` | bodies of `TipsOff`, `SetTipsOff` |
| `(*CampaignSession).cityBookCandidate`, `commitCityBookCandidate` | FrontEnd methods that cloned the whole front end |
| `(*townScreen).nextParty` | `FrontEnd.NextParty` over the screen's session and install |

`resolveShopSnapshot` builds its detached screen over a copy of the install
value and a fresh `CampaignSession`, with zero runtime, presentation and
persistence components, as the detached `FrontEnd` it replaced had.

## Differences from the base

Statement order, error strings, nil guards and hashed state are unchanged,
except where listed.

- The city-book candidate is the cloned `CampaignSession` only. The base cloned
  the whole front end, so an action running on the candidate saw copies of the
  other four components. The candidate screen now shares them. No action run
  under `withCityBookMutation` or `withCityShopMutation` writes runtime,
  presentation or persistence state; the one shared effect that was possible,
  a first-use art cache filled on the live component instead of a discarded
  copy, loads the same value.

## Proof

- `townports_test.go` builds a town screen over bare components and runs the
  footer, the next party, the unbound screen, the city-book candidate and
  commit, `withCityBookMutation` with no city graph, the tip preference and the
  art caches. No FrontEnd is built.
- `TestResetForNewGameDropsExactlyTheGamePopulation` classifies the six ports
  as kept across a load.
- Release fixtures that read the front end beside the screen look it up through
  `frontOf` (`townfront_test.go`).
- Unchanged EN and RU release tests, the milestone-2 acceptance family and the
  scenario hashes carry the no-behaviour claim. Gate verdicts are in the lane
  return.

## Measures

| measure | before | after |
|---|---|---|
| component fields | 82 | 82 |
| coordinating functions (three or more components) | 15 | 15 |
| `townScreen` fields typed `*FrontEnd` | 1 | 0 |
| `CommentBytes` baseline | 8621269 | 8620735 |

The screen's functions were not in the coordinating set (the ratchet counts
reach through a FrontEnd value), so that count does not move. The storyguard
baseline falls by 534 bytes.

## Open debt

- `loadMap` (non-mission rows) keeps its own art setters. They differ from
  `viewerArt.apply`, so moving it onto `viewerArt` would change what the
  picker's map shows.
- `openCityBase` and `continuity` remain method values of the front end:
  the first projects the whole game state as a SAVE would write it, the second
  opens screens and the next mission's opener. `openMission` is the same kind
  of port: opening a mission spans the whole game.
- The screen holds five components, so it still reaches every state group of
  the game. Narrowing `rt` to the audio players and `pr` to the art caches
  needs the screen's art and audio reads to become named services; that is a
  further slice.
- Release fixtures need `frontOf` because they read the front end beside the
  screen.

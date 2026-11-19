# The second game's world map plays its route sounds and cycles its flags as the original

## Result

The second game's world map alternates two sounds while the road to a place grows, Point2 first, and its flags and
party marker cycle by the frame counts the original uses. Map music with the music option on is unchanged. The first
game's map, its sound and every world hash are unchanged.

## Authority

- Owner decision 5, answered «as recommended»: the sound and marker cycles as a small story; the travel cadence
  stays as accepted.
- Claims R2-ENGINE-430 (High), R2-ENGINE-463 (High, Medium for the parts below), R2-ENGINE-429, R2-ENGINE-465.

## Rules

1. When the second game's map opens with music on, it requests map.wav; with music off nothing starts.
2. When a route grows, Point2 starts first; each later start is the other sound once the current one has ended.
3. When the reveal reaches the route's end, no sound starts; leaving the map silences both.
4. When markers are drawn, their frame follows the counter wrapped by Flag1's frame count.
5. When the party flag is drawn, its frame follows the counter wrapped by Flag's frame count.
6. When the first game's map runs, it names no sound or cycle and wraps by each sheet's own frame count.

## Code

| kind | its one builder | change |
|---|---|---|
| world map screen | `pkg/game/worldmapstate.go:208` tick, `pkg/game/worldmapdescription.go:28` WorldMapDescription | `cycle` and `sounds` as data |
| sound request | `pkg/game/townscenehost.go:152` scenePlay | the map asks through it |
| frame cycle | `pkg/game/worldmap.go:170` worldMapWrap | cycle length is the named sheet's count |
| game branch | `pkg/game/worldmaps/rom2.json` | description data, no new game test |

## Proof

| test | rules |
|---|---|
| `TestSecondMapMusicFollowsTheMusicFlag` (ui) | 1 |
| `TestSecondMapRouteSoundsAlternateFromPointTwo` | 2, 3 |
| `TestSecondMapLeaveStopsRouteSounds` | 3 |
| `TestSecondMapFramesWrapByTheNamedSheets` | 4, 5 |
| `TestMapDescriptionsNameSoundsAndCycles`, `TestFirstMapFramesWrapByTheirOwnSheet` | 6 |

Release witness at the release: the second game's map on the EN and RU installs, one travel heard and watched.

## Open debt

- DIV-3069: cursor image Unknown; device packing; Medium: play-once and leave silence, unread arrival, retarget and
  skip bodies, the route test before the step's advance.

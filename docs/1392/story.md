# The second game's music

## Intent

The second game plays its own music on `rom2-en` and `rom2-ru`: the fixed
screen keys, the mission's seventeen-track list with its music areas, and the
stops at movies and mission boundaries. Before this story the music bank opened
only the first game's `Allods/MUSIC.RES`, so a second-game root played nothing.
The first game is unchanged. Base: public main `27c7f72d`; knowledge pin moved
from k229 to k232.

## Authority

- `R2-ASSET-083`: `music.res` holds 21 PCM keys, `b00`..`b16`, `chrgen`,
  `credit`, `map`, `menu`, equal in EN and RU.
- `R2-ASSET-084`: ALM type 12 is a head record and area records of seven
  int32: x, y, radius in tiles, four themes, -1 for none.
- `R2-ENGINE-335`: menu `menu.wav`, pre-create `chrgen.wav`, town
  `b14`/`b16`/`b15`, map `map.wav`, credits `credit.wav`; menu, pre-create and
  town keep a key the player already holds.
- `R2-ENGINE-231`: town IDs 1, 2 and 3 select `b14`, `b16` and `b15`.
- `R2-ENGINE-336`: the mission list `B00`..`B16`, its random start, the area
  select at entry and every 16 mission ticks, the record walk and the theme
  draw.
- `R2-ENGINE-337`: at a track end the player opens the area pick, else the
  next entry; stop pauses, start resumes.
- `R2-ENGINE-338`: logos, movies, mission entry and mission end stop the
  player; a later screen request resumes it.
- `R2-ENGINE-339`, `R2-ENGINE-340`, `R2-ENGINE-311`: the registry settings,
  `-nomusic` and the sound panel.
- `R2-ENGINE-341`: `tune=` is in no shipped file.

No first-game claim decides a second-game point. Medium and Unknown points
are rows `DIV-2860` to `DIV-2867` in `docs/divergences/rom2-music.md`.

## As built

The kind is the music controller: `ui.MusicController`, built once by
`App.SetMusic`, with `game.MusicBank` as its source. Both games use it. The
game facts became data in one music description per game,
`pkg/game/musics/rom1.json` and `rom2.json`, decoded by `ui.DecodeMusic` and
named by the edition's new `Music` field (`pkg/base/edition.go`), as the
generator description is. A description states:

| Field | First game | Second game |
|---|---|---|
| archive | `Allods/MUSIC.RES` | `music.res`, either case |
| tracks | the 21 names of the former manifest | the 21 keys of `R2-ASSET-083` |
| scenes | the former scene table; credits added with `menu.wav` | menu, credits, chargen, campaign (map), mission, town |
| keep | every scene but the mission | menu, chargen, town |
| list | shuffled (former rule, `MAGIC-286` on the shared stream) | ordered, start at one draw modulo the length |
| silence | clear (former rule) | pause |
| unlisted screens | silence (former rule) | keep the held list |
| towns | none | 1 `b14.wav`, 2 `b16.wav`, 3 `b15.wav` |
| areas | none | period 16 |

The controller has one request rule for both games. An unchanged request does
nothing; a completed load makes a fresh request. A silent request clears or
pauses by the description. A scene that keeps holds a list equal to its own,
resuming it when paused; any other request sets its list. At a track end the
area pick, when set, decides the next entry, else the next entry follows. The
first game never sets a pick, never pauses and keeps its former order rules,
so its request log is unchanged.

Second-game screens:

| Screen | Request | Keep |
|---|---|---|
| main menu, cutscene library | `menu.wav` | yes |
| credits | `credit.wav` | no |
| generator, both pages | `chrgen.wav` | yes |
| town 1 / 2 / 3, its tavern | `b14.wav` / `b16.wav` / `b15.wav` | yes |
| destinations | `map.wav` | no |
| mission | `B00.wav`..`B16.wav` | no |
| a movie | pause | |
| game menu, load, save, documents | none: the held list plays on | |
| hall of fame, ending | none | |

Mission rule: `openMission` hands the viewer the area array
(`missionMusicAreas`: head first when its first theme is at least 0, then the
areas, decoded by `alm.Map.MusicAreas`) and the hero
(`mapWorld.musicHero`: the primary party hero's cell centre in 1/256 tile and
the world tick). Setting the list draws the start entry, then selects the area
under the hero; a pick replaces the start. While the mission shows, each frame
that passes a multiple of 16 world ticks runs the select. The pick changes the
music at the end of the current track.

`ui.MusicPauser` is the device seam for pause; the production device pauses
its player and reports no end while paused.

Settings: the second game reads and writes the engine's music preferences
(`pkg/game/music_preferences.go`); no registry (`DIV-2860`). `-nomusic`
(`cmd/againrom`) opens no archive, on either game.

SAVE and LOAD: no claim places music state in a SAV (`R2-ENGINE-337`,
`R2-ENGINE-339`); the SAV carries none. A LOAD of a mission save sets the
mission list fresh, as mission entry does (`R2-ENGINE-336`, `DIV-2866`).

## Proof

Witness: `pkg/game/secondmusic_release_test.go`, run on `rom2-en` and
`rom2-ru` with a recording device; both roots give the same logs.

- `TestReleaseSecondGameMusicPopulation`: the archive lists the 21 keys and
  every key and mission name decodes.
- `TestReleaseSecondGameMusicScreens`: menu `request:menu.wav`; load dialog and
  back: nothing; generator `stop, request:chrgen.wav`; OK, Back, OK across
  both generator pages: nothing; Accept to town 1 `stop, request:b14.wav`;
  tavern and its talk: nothing; gates `stop, request:map.wav`; mission 10
  `stop, request:B00.wav..B16.wav` with the playing track at the pick; game
  menu: nothing; exit `stop, request:menu.wav`; credits
  `stop, request:credit.wav`; back `stop, request:menu.wav`.
- `TestReleaseSecondGameMusicAreas`: mission 10 holds 8 records, head themes
  0, 1, 8. The hero placed in area (14,41) picks 1 and in area (57,10) picks
  3; the playing track does not change until its end, which opens `B01.wav`
  and `B03.wav`. Outside every area, at (11,8), the pick is a head theme. A
  named mission SAVE and a cold LOAD log
  `request:menu.wav, stop, request:B00.wav..B16.wav` with a pick of the area
  under the loaded hero.
- `TestReleaseSecondGameMusicStops`: the mission 10 completion makes the
  destination request and the movie pauses it (`stop, request:map.wav,
  pause`); after the movie `stop, request:map.wav`; exit to the menu; the same
  movie over the menu pauses and its end resumes (`resume:menu.wav`). Without
  the archive the requests log and no track starts.
- `pkg/ui/musicrules_test.go`: the second game's table, the ordered start, the
  pause and resume sequence, the town key, the area walk and track-end pick,
  a map without an admitted head playing in order, the first game reading no
  area. `pkg/ui/music_test.go` still states the first game's former table.
- `cmd/againrom/nomusic_test.go`, `pkg/formats/alm/musicarea_test.go`,
  `pkg/game/missionmusic_test.go`.
- First game: the music and sound release tests and the request-log tests on
  RU and EN, unchanged.

## Open debt

- `tune=` and the off fade are not built (`DIV-2864`); no shipped dialogue
  reaches them.
- The area select reads the hero's cell centre once per frame (`DIV-2861`).
- An unreadable track is skipped rather than closing the game (`DIV-2865`).
- The release witness of a map without an admitted head is the unit test; no
  second-game root map is opened through the engine's mission route.

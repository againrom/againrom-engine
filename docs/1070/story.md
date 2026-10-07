# 1070 — static music lifecycle

## Player result

The selected install now supplies stereo music for the main menu, character
generation, campaign map and gates, missions, town square, shop, tavern and
school. A scene change replaces the old list once; redraws and overlay screens
leave it playing. End of file advances a shuffled ordinary list, including all
twelve mission tracks before repetition. One-file lists repeat.

Music is optional. A missing device, archive, track or malformed WAVE leaves the
game silent and never prevents startup. The existing Sound Options enable and
volume controls effects and music together.

## Authority and as-built behaviour

`VIDEO-MUSIC-001` through `VIDEO-MUSIC-012` establish the exact 21-name archive,
stereo 22,050 Hz PCM population, fixed candidate-list owners, seeded random
initial traversal, ordinary EOF behaviour, class-dependent school ordering,
one retained stream and replacement/destruction lifecycle. The production
table is:

| Surface | Ordinary list |
|---|---|
| Main menu | `menu.wav` |
| Character generation | `chrgen.wav` |
| Campaign map and town gates | `map.wav` |
| Mission | `B00.wav` through `B11.wav` |
| Town square | `town.wav` |
| Shop and shop dialogue | `shop.wav` |
| Tavern and tavern dialogue | `inn.wav` |
| School and school dialogue | `schoolm.wav`, `schoolw.wav`, with the selected mage or fighter entry first before seeded shuffling |

`inn_ssi.wav` is validated as part of the archive but is assigned to no scene.
Text `tune=N`, runtime-computed targets and ambience are outside this story.
`DIV-245` retains those residual identity questions. `DIV-505` records this
build's selected-root archive resolution and shared Sound Options policy.

`MusicBank` opens `<selected root>/Allods/MUSIC.RES` only on first demand. Its
file-backed RES/VFS path retains the registry and reads only the requested
payload. The controller releases decoded ownership to the one retained player;
replacement closes the prior player before starting another. No archive or
complete track population is cached. Music changes no simulation, save or hash
state.

## Proof

- `pkg/audio/track_test.go` distinguishes left and right stereo samples and
  covers conversion and malformed input.
- `pkg/ui/music_test.go` pins the exact scene table, both school orders,
  deterministic shuffle, all twelve mission tracks before repeat, one-file EOF
  repeat, stop-before-start replacement, overlay/redraw idempotence, silence on
  missing inputs, teardown and at-most-one current track.
- `pkg/game/townmusic_test.go` pins every town room and dialogue owner plus the
  selected fighter/mage school order.
- `pkg/formats/res/file_test.go`, `pkg/vfs/fs_test.go` and
  `pkg/game/music_test.go` prove lazy disk-backed indexing and current-only
  reads without retaining the roughly 110 MB archive.
- `TestReleaseStaticMusicPopulation` resolves the exact manifest and decodes all
  21 tracks sequentially on each gated EN/RU install.

## Open debt

ROM1's normal music mount/default and separate music-volume initializer remain
Unknown (`DIV-505`). Exact identities for `inn_ssi`, any computed target and a
shipped `tune=N` use remain Unknown (`DIV-245`). Ambience is a separate backlog
slice and is not inferred from the static list lifecycle.

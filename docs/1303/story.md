# Story 1303: Game Options and Sound Options adoption

## Intent

Adopt k140 (EXP-0457) for known defect B11, the settings screens. The Game
Options and Sound Options dialogs take the original frame, transaction and
control layout where a High part of a claim contradicts the engine.

Owner rulings: every graphics and presentation option defaults to its
highest-quality setting, Smoothing included, and a stored player choice wins;
the debug console is not built.

## Authority

Pin k140 (`0ec3354`). Claims: `MENU-073`, `MENU-074` (Game Options),
`MENU-075`, `MENU-076` (Sound Options), `MENU-077` (dialog frame), `VIDEO-075`
(cutscene list), `VIDEO-076` (music lists), `VIDEO-077` (defaults),
`TEXT-099`, `TEXT-100` (strings).

| Claim | Outcome |
|---|---|
| `MENU-073` | Adopted: layout, nine-position speed slider, staged copy. Rectangles are Medium (DIV-1289, DIV-2121, DIV-2127) |
| `MENU-074` | Adopted: OK writes in claim order, Cancel and Esc drop, Animation 0 forces Lighting 0, the three party commands are sent on every OK |
| `MENU-075` | Adopted: control rectangles, four-row track list, Random, Acknowledgments, Play, Stop, OK. The authored strip stays (DIV-2122) |
| `MENU-076` | Adopted: square-law slider applied at the event. The store stays a linear percentage (DIV-2123) |
| `MENU-077` | Adopted: snapped sizes 488x424 and 488x360, centred, 8 px shadow, `lm` nine-piece body. The cutscene frame is not adopted (DIV-2124) |
| `VIDEO-075` | Divergence: the engine keeps its encountered catalog (DIV-2124) |
| `VIDEO-076` | No change to the list source; the Random Order checkbox keeps the stored value (DIV-2125) |
| `VIDEO-077` | Graphics defaults already agree (all four on); the switches are not read (DIV-2125); save timing differs (DIV-2128) |
| `TEXT-099` | Adopted: Cancel and Speed captions from dialog rows 1 and 50; Shadows, Lighting, Animation from `patch.txt` rows 52 to 54 |
| `TEXT-100` | No change: the tune titles already come from `tunes.txt` |

## As-built behaviour

- Dialog frame (`pkg/ui/dialogsnap.go`). Size snaps to `(w-8)/96*96+8` by
  `(h-104)/64*64+104`, centred on 640x480, with an 8 px shadow band.
- Game Options (`pkg/ui/gameoptions_draft.go`, `gameoptions_draw.go`). A click
  edits `GameOptionControls.draft`; OK commits it, Cancel, Esc and every other
  exit drop it. Map shortcuts stay immediate. A failed write keeps the page
  open with the first error.
- Sound Options (`pkg/ui/soundoptions*.go`, `volumelaw.go`). Slider volume is
  `trunc(-5000*((pos-5000)/5000)^2)`, applied at each move or key step and
  mapped to the percentage store by `round(100*10^(volume/2000))`. Esc keeps
  the applied value.
- Captions (`pkg/game/gameoptions.go`) load from the install; authored RU words
  stand in only for an absent row.

## Proof

Focused tests (`pkg/ui`): `TestDialogBaseSnapsAndCentresTheThreeArgumentRectangles`,
`TestOptionsDialogControlsLieInsideTheirFrames`, `TestSoundSliderUsesTheSquareLaw`,
`TestSoundPercentRoundTripsThroughTheSlider`, and the rewritten Game Options and
Sound Options tests. The Game Options tests fail on the parent because a click
wrote at once and there was no Cancel row.

EN and RU release witnesses: `TestReleaseSettingsDialogsReadInstalledRows`,
`TestReleaseSettingsDialogsFrameCaptionsAndMissionTracks` and the updated
options, graphics, autohealing, tooltip, speed, sound and game-menu release
tests. Loss controls: a click leaves the stored value unchanged until OK;
Cancel and Esc leave it unchanged after changes; a refused write keeps the
page; Animation 0 with Lighting 1 writes Lighting 0.

## Open debt

DIV-2121 slider gating, DIV-2122 authored sound strip, DIV-2123 default volume
(owner decision: original is -700, about 85 percent; engine is 100),
DIV-2124 cutscene list, DIV-2125 switches, DIV-2126 Esc destination,
DIV-2127 authored game controls, DIV-2128 storage timing. The debug console is
not built by owner ruling.

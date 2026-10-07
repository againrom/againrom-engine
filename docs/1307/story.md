# Story 1307: cutscene sound position, final frame and sidecar edge cases

## Intent

Movie frames follow the sound position while a track plays, the stream ends at
the final frame as ROM1 does, and the sidecar and fade rows state exactly what
`VIDEO-081` to `VIDEO-084` now ground.

## Authority

Knowledge pin k143. `VIDEO-081` (High): the wait compares a byte position with a
byte target and no time or count bounds it. `VIDEO-082` (High call order, Medium
cut length): the final blit returns at once and the close stops and releases the
sound buffer. `VIDEO-083` (High census, Medium outcomes): missing sidecar,
negative count, equal start and end frames. `VIDEO-084` (High): the fade scales
the current frame's palette and the open sets the new-palette flag. The
palette-timing clause of `VIDEO-071` is refuted by `VIDEO-084`.

## As-built behaviour

- `pkg/video/soundclock.go`: `SoundClock` reports played bytes in the movie's own
  sample format. `Player.SetSoundClock` hands it to the decoder goroutine.
  Frame k waits until played bytes plus 8 reach k header intervals times bytes
  per second; frame 0 needs none. Without a track, a clock or a session the
  timer wait of `VIDEO-073` paces, as before.
- `pkg/ui/cutsceneaudiodev.go`: the cutscene device is a `SoundClock`. Played
  bytes are the bytes the player read from the queue, silence excluded, halved
  for a mono track. The code subtracts the player's buffered size only when the
  player offers `BufferedSize`; the shipped player (ebiten audio v2.9.9) does
  not, so nothing is subtracted. `App` attaches the device
  after `Start` on the first frame.
- A position that does not move for 2 seconds ends sound pacing for the rest of
  the movie and the timer takes over. `video.StallLimit` stays (`DIV-2055`).
- The stream ends after the final frame with no hold. The close stops the
  device and cuts the queued audio, about one frame (`DIV-2056`, OPEN, UNKNOWN).
- The presenter treats the first frame as carrying a palette (`VIDEO-084`).
  The fade already scaled the current frame's palette, with no lag.

## Proof

- `pkg/video/soundclock_test.go`: target arithmetic; release exactly at target
  less 8; frame 0 immediate; timer pacing with no clock or session; hand-over to
  the timer when the session ends; stall fallback; stop interrupts a held wait;
  a three-frame stream ends with no hold after the final frame.
- `pkg/ui/cutsceneaudiodev_test.go`: played bytes exclude silence, halve for
  mono, subtract buffered bytes, and vanish on Stop.
- `pkg/video/present_test.go`: the first frame sets the palette without a flag.
- Not proved: audible timing. No sound output was opened in this story.

## Open debt

- The position leads audible audio by at least the player's 40 ms buffer, which
  the shipped device does not let the code subtract; video leads audio by about
  that much (`DIV-1256`).
- `DIV-1256` stays a DEVIATION: the position is the player's read count, not the
  DirectSound cursor with interpolation; the byte target is nominal; the decode
  skip while audio runs ahead (Medium) is not reproduced.
- `DIV-2056` is OPEN as UNKNOWN: the build cuts about one frame of audio at the
  final frame per `VIDEO-082` (Medium), superseding the Story1294 directive that
  the last audio is not cut off; it waits for owner confirmation of the cut.
- `DIV-2055` stays: a guard against a held wait is kept; no claim removes the
  need.
- `DIV-2054` moved to Divergences: missing sidecar (graceful), equal-frame fade
  and pan, and out-of-range source origin differ from ROM1; no shipped sidecar
  reaches them.
- Reserved `DIV-2141` to `DIV-2144`: none used.

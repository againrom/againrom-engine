# Story 1294: cutscene presentation

## Intent

Movies play as ROM1 presents them: the sidecar fades, pans and start origin act, a movie is windowed or doubled by the original rule, and frame pacing follows the decoder wait. Closes known defect B13.

## Authority

- `VIDEO-071` (frame step, reader, fade and pan arithmetic), `VIDEO-072` (window and doubling rule), `VIDEO-073` (decoder wait), `VIDEO-074` (shipped census), `REG-CUT-053` (sidecar schema), `VIDEO-031` (sidecar name).
- Pin k132.

## As built

- `pkg/video/sidecar.go`: `ParseSidecar` reads `Common`, `Fading<n>` and `Panaraming<n>` with every key defaulting to 0 and a record count bounded at 4096. `SidecarName` replaces the last extension with `.reg`. `CutsceneBank.Open` reads the registry from the movie archive and passes it to `StartSmackerDecoderWith`; a movie with no registry plays without effects.
- `pkg/video/present.go`: the presenter runs per frame in the `VIDEO-071` order (arm fade, load or clear pan step, scale palette, blit, add pan step unless last). The ARV2 stream now carries presented frames: 640x360 for every movie but the 640x480 logos, which keep their size. A movie with 2w <= 640 or 2h <= 360 is drawn as 2x2 blocks; a wider or taller one shows a 640x360 window at the source origin. Source positions outside the movie draw black.
- `pkg/video/pace.go`: the timer wait in 32-bit units of 10 microseconds (first poll arms, lateness rule re-arms). `Decoder.IntervalUnits` replaces the 100 ms default for a zero header: zero waits for nothing, a non-negative value is milliseconds times 100 modulo 2^32, a negative one its magnitude. Each frame, the first included, waits before delivery, and the stream ends one interval after the final frame (`DIV-2056`); a reader close interrupts the wait.
- The audio gate also reads the track coding (`Decoder.AudioDecodable`).
- `pkg/ui` composition is unchanged in code: the 640x360 presented frame lands unscaled at row 60 of the 640x480 canvas.

## Proof

- Unit tests: `pkg/video` (`present_test.go`, `pace_test.go`) and `pkg/video/smacker/interval_test.go`, covering the doubling table, window and start origin, pan timing, fade-in and fade-out endpoints, record order, sidecar parsing and bounds, the timer wait, wrapped deadlines and skip interruption.
- Release witnesses on each root: `TestReleaseCutsceneNativeAppCompletionAndSkip` compares the first six frames of the installed M10/01 (fade-in from frame 0, pan from frame 0) against the installed decoder's packed output with an independently computed pan origin and fade factor; `TestReleaseCutscenePresentationAndSidecars` presents all 33 movies of both archives and checks the 640x360 and 640x480 sizes, 30 and 3, and the four pan registries.

## Open debt

- `DIV-1256`: the sound-position wait (`VIDEO-073`, Medium) is not reproduced; every movie uses the timer wait.
- `DIV-2054`: sidecar edge cases the claims leave Unknown or Medium, including the one-frame palette lag.
- `DIV-2055`, `DIV-2056`: the player stall guard and the one-interval hold after the final frame.
- With no fade active the display palette is set only by a frame carrying a palette record (`Frame.NewPalette`), so M10/01 and M50/01 stay black after their final fade-out to the last frame. `TestReleaseCutsceneFinalFramesStayBlackAfterFadeOut` checks it on both movies. The reading of `handle+0x68` as the new-palette flag is Medium (`DIV-2054`).

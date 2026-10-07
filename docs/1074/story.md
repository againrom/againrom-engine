# Bounded optional cutscene playback

Fresh campaign missions now play numbered movies from the selected lawful
install before map input or simulation ticks. Both preserved roots produce
90 moving M10 frames and return to the map on completion or skip.

## As-built contract

- `cmd/againrom` configures the optional bank before building App. Fresh
  mission entry after character generation or campaign continuation requests
  `mN/01.smk` through `mN/99.smk`. Save restores and loose maps do not.
  Missing members and decoder failure continue; any focused key/button press,
  or close ends the scan. The destination stays intact.
- `-8x` selects VIDEO8; `-4x` overrides it. Otherwise VIDEO4 is selected.
  `-movies=false` disables playback. Lookup is case-folded and confined to
  that root/archive. No loose-file or other-archive fallback occurs.
- The normal game remains amd64. A separately built Windows/386 helper loads
  the selected install's absolute `smackw32.dll`. Validated PE exports bind
  only the five published callee identities. The adapter uses a read-only
  borrowed file handle, DLL-allocated state, basic indexed output and the
  published palette. It terminates at the header frame count, never native wrap.
- ARV1 transports top-down RGBA. Limits are 128 MiB compressed media, 2,048
  pixels per dimension, block-aligned native extents, 100,000 frames,
  1 ms..1 s frame interval, 20 minutes duration, 4,096 diagnostic bytes,
  one queued frame plus one producer frame, and 10 seconds without progress.
  The child is hidden, killed/reaped on cancellation, and its private input is
  removed after exit. DLL-unknown malformed data remains isolated in the child;
  this is process separation, not an OS security sandbox.
- App composes an opaque black 640x480 aspect-fit movie, suspends music and
  ambient loops, blocks underlying dispatch, clears active drags and drains
  held skip gestures. Completion/skip preserves the world hash and tick.
  Missing helper/DLL/media and unsupported platforms fail open.

## Authority and boundaries

Published authority is research pin `0d829843102d57aa4778aa4407749a2044fe3938`:
`VIDEO-029` through `VIDEO-036`, `VIDEO-045` through `VIDEO-051`, and
`REG-CUT-053`. The corrected allocation-selector contract is respected:
open flag `0x1000` borrows the file handle; `0x02000000` remains clear.
No probes, disassembly or third-party decoder source was copied.

The isolated lawful-DLL adapter is an engineering decision authorized in the
implementation brief, not an explicit owner ruling or a clean-room decoder.
DIV-507..510 disclose the platform/source policy, unresolved gate and omitted
logo/registry route, sidecar/scaling debt, and silent authored timing. No
sidecar fade/pan, audio, original blitter or physical-ROM1 cadence is claimed.
Playback continues while unfocused; key/button skip requires focus.
Reserved DIV-511..514 remain unused permanently.

## Delivery and proof

Build `GOARCH=386 go build -trimpath -o <bundle>/cutscenehelper.exe
./cmd/cutscenehelper` separately; build `cmd/againrom` normally for amd64.
Do not overwrite the helper with the amd64 unsupported-platform stub when
building other commands. Ship neither DLL nor game assets.

`againrom -assets <root> -sound=false -cutscene-check <existing-private-dir>`
drives the actual native adapter, mission opener, App composition and dispatch
without desktop input. It writes two PNGs outside the install. A separate
packed RGB565 decode grounds the first six composed frames independently of
the gameplay indexed-palette path. `verification.md` records the measured
result and its limits; the registered EN/RU release test runs the same proof.

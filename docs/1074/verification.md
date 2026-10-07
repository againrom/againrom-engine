# Cutscene playback verification

## Observable result

Before this slice, no production startup/session route played a movie. The
actual amd64 `againrom.exe` and separate 386 `cutscenehelper.exe` now produced
90 moving frames from `VIDEO4/M10/01.smk` on both preserved roots, then returned
the App to the map. Native extent was 800x360; the composed frame was 640x480
with opaque black letterboxing. VIDEO8 passed the same EN and RU witness.
Natural playback starts through the normal character-generation Play dispatch;
the skip cases also exercise direct mission entry.

The witnessed code commit is `871b85f2714f4ae254c8a077f048ebddf23d0a35`, based
on pushed master `e31ca935d640832bf1de7114899f0922fa8a4205` and research pin
`0d829843102d57aa4778aa4407749a2044fe3938`. The initial evidence-record commit
`f7d31fda7f708267b1103029db9cbe296758b4ba` was documentation only.
`go version -m` reports the witnessed code SHA and
`vcs.modified=false` on both actual executables, with GOARCH amd64 for the game
and 386 for the helper. Neither build disabled VCS stamping.

Each run printed `frames=90 composed=90 oracle565=6/6 motion=yes
route=chargen-to-map complete=map tick=0 unchanged`. Key, left-button,
right-button and close skips each followed
a real native frame and returned the map with unchanged world hash and tick.
VIDEO4 compressed SHA-256 was
`f6252abb235db26b09dda3c3ddeb7e5531bb67b50ca2c46b1e30479b3e9216bc`;
VIDEO8 was `6e04f05d5a5b41416bb73b79bf5be8bd622a2b78e020f8d7f33d8a3b17c0a1e9`.
These agree with the published corpus table.

The pixel oracle decodes the first six frames through the separate packed
RGB565 path, then compares all 640x480 composed pixels after channel
quantization. It does not call App's compositor or the palette conversion to
construct expected colors. First and sixth reference frames differ. The
authored ARV1 fixture independently tests exact red/green and blue/white
colors and black letterboxing without an installed decoder. A separate literal
rectangle oracle pins the complete 640x480 placement. A production x-origin
mutation of +1 pixel failed `TestCutsceneFitMatchesLiteralColoredRectangles`;
restoration passed and reproduced blob
`266b1082818d5ce11d6e1da2a964b1d3c4e28af4` byte-for-byte.

Images are private, untracked `review/story1074/{en,ru,en8,ru8}/frame001.png` and
`frame006.png`. The sixth EN composite was visually inspected: village,
windmill, buildings and black letterboxing. This is inspection of App's CPU
composite, not a physical GUI or ROM1 screenshot. No synthetic desktop events
were sent. `builds/current/` remains the seat's landing/rebuild responsibility.

## Focused checks and ownership

Focused video/UI/game tests pass for native preflight bounds, malformed ARV1,
bounded read-ahead, stalls, process cancellation/reaping/private-input removal,
4,096-byte diagnostics, archive selection and misses, numbered scans,
completion, skip, held-input drain and music teardown. FrontEnd's field census
classifies the bank as install state, not campaign state.

`TestReleaseCutsceneNativeAppCompletionAndSkip` is registered in the release
population. It builds a Windows/386 helper, then runs the real adapter/App
witness for the selected root. Non-Windows has no native subject; absence of
the selected root is explicitly asset-gated.

Allocation sweeps before and after DIV-507..510 both printed `ledgers + DIV: 32,
missing answers: 0`. The floor stayed DIV-532. No simulation, save-byte-form,
research claim, allocation-floor or preserved-install byte was changed.

## Mission census

`pipeline/milestone-baseline.txt` carries mission 10's `16 checks, 27 instants,
12 triggers` and mission 20's `14 checks, 15 instants, 11 triggers`. The prior
checkpoint measured `UNSUPPORTED=0` for each. This slice changes presentation,
not the measured script population. The final stamped worktree missionrun
returned `UNSUPPORTED=0` for mission 10 and `UNSUPPORTED=0` for mission 20,
with those same populations. Both counts are unchanged; playback is the
external result this slice delivers.

## Witnessed code gates

On `871b85f2714f4ae254c8a077f048ebddf23d0a35`:

- `go test -trimpath -count=1 ./...`: pass, 70 package results. The initial
  failing architecture/VFS/screen-registration guards were corrected before
  this clean run. Complete output: `review/story1074/full-go-final.log`.
- `gofmt` and `git diff --check`: clean. Native `GOARCH=386 go vet ./pkg/video
  ./cmd/cutscenehelper`: pass. Linux/amd64 compilation of the video leaf and
  helper fallback: pass. Race execution was not run (CGO is disabled).
- `scripts/check-no-game-assets.sh`: `clean (tree scan)`.
- Four clean stamped executable witnesses: EN/RU, VIDEO4/VIDEO8; 90/90
  compositions, 6/6 packed pixel references, natural completion and all four
  skips. Exact outputs and build metadata are in `review/story1074/`.
- The registered native/App release test passed on EN. The seat owns its
  paired merge-commit run through `check-release-tests.sh`; no branch paired
  chain is claimed. `scenarios/1080-function-keys.json` passed all 17 steps on
  EN. Separate focused tests prove F2/F3 skip before save/load dispatch.
- `check-div-claims.sh`: 267 live rows, 355 distinct cited IDs, 66 partial-
  retraction hits. New DIV-507 and DIV-509 cite the corrected surviving
  VIDEO-045 and VIDEO-031 clauses, not their retracted allocation/first-dot
  clauses. Existing hits were not re-audited.
- `check-preserved-installs.sh`: `ok — 181 file(s), both roots as recorded`.
  This is the instrument's name/size comparison, not a full byte hash.

## Reconciliation with 1081

Merge `c1dc2506ea387c0f0636bf2982ea6e18baee1889` incorporates published master
`3fe429ce065a28cf521a8e5204c3a0d664034181` without conflicts or pin movement.
The release population is 99: the prior 96, two spell-pass tests and this
story's native/App test. Allocation sweeps surrounding the merge again
reported `ledgers + DIV: 32, missing answers: 0`; the floor stayed DIV-532.

The native helper, video package, startup, bank, frontend and App playback
files are byte-identical to the witnessed `871b85f2`. Viewer retains 1081's
renderer ordering; this story's Viewer delta remains only the presentation
entry field and field alignment. The whole game is therefore not claimed
code-identical to the earlier executable.

Focused video/UI/game, architecture, release-population and screen guards
passed, including cutscene/F2/F3 routing and spell/projectile rendering tests.
The retained-overlay cell-arm priority test also passed. `gofmt`,
`git diff --check` and `check-no-game-assets.sh` are clean. The unchanged
native quartet and full Go suite were not repeated for this reconciliation;
their evidence above remains attributed to `871b85f2`. The seat owns the
final merge-commit suite and paired EN/RU release chain.

The seat owns the sole fresh-context review, serialized landing and
`builds/current/` rebuild. No review, landing, physical GUI observation,
audio/sidecar fidelity or portable clean-room decoder is claimed.

# Verification — 0126

Branch `0126-sound`. Three task commits, one merge of `origin/master` (`27aff73`), one
reconciliation commit. `verification.md` and the build are stages and carry no trailer.

## What landed

`pkg/audio` is new: a stdlib-only leaf holding the WAV decoder, the resampler, the placement
arithmetic, the stereo synthesis and the settings. `pkg/ui` gained the grunt rule, the throttle, the
listener and the ebiten-backed device; `ui.MapEntity` gained one field. `pkg/game` gained the sound
archive, the slot registry, the per-class slot table and the swing emission, and `cmd/againrom`
gained `-sound` and `-volume`. Nothing reached `pkg/sim`; no byte-form version was taken.

## What I actually heard: nothing

**Nobody has listened to this build.** I cannot hear, and no test in this tree can observe a sound.
That is the single largest gap in this story's evidence and it is stated first rather than last.

What is established instead is every link of the chain up to the speaker, and the chain is short:

1. **The assets resolve and decode, on both installs.** A throwaway census (not committed) drove
   `game.OpenSounds` and `game.LoadUnitSounds` over each lawful root, resolved every slot every unit
   class names, and decoded each one. Both roots gave the identical answer: 34 classes, **every one
   with exactly five elements**, **153 nonzero elements and 17 zero**, **65 distinct slots, 0
   unresolved, 0 that decoded to silence**, minimum RMS 1625, every sample arriving at the device
   rate. That reproduces `REG-SFX-057`'s own 153-of-153 census through our loader rather than
   quoting it.
2. **The index mapping is right, and it is the one thing that was easy to get off by one.** The same
   census counted zero elements per index: **[0]=3, [1]=10, [2]=2, [3]=2, [4]=0** over 34 classes.
   That is exactly `ANIM-SND-022`'s own census — three classes with a silent swing, ten with a
   silent unchanged-health grunt, two Catapults silent on both wound grunts — so the story indexes
   0, 2 and 3, which is what the claim names. Element 4 is nonzero in all 34 classes and read by
   nothing; it stays unused.
3. **The device opens on this machine and takes a real sample.** A throwaway program called
   `ui.OpenAudio` (`player=true, err=nil`), decoded the swing slot a real class names (slot 110,
   22050 Hz, 6352 frames, 288 ms), and issued three plays. `audio.Stereo` produced exactly
   `frames*4` bytes each time, and the placements were `{10000,10000}` centred, `{10000,550}` at
   eighteen cells left, `{550,10000}` at eighteen right.
4. **The wiring is live in the real windowed game, and so is the seam.** A throwaway traced build
   of `cmd/againrom` (also not committed) printed, on `-mission 10` against the EN install:
   `SetAudio player true bank true`, `setSwingSound classes 34 play true`, and then once a second
   `listener (17,67) true entities 36 sound0 [120 200 220 223 240]`. So the viewer holds a real
   device and a real bank, the swing emitter holds all 34 class rows, **`listenerCell` answers**
   rather than refusing, and a real entity crosses the seam carrying five real slot ids. That
   listener trace is the one that mattered: a listener that would not resolve makes `playSlotAt`
   return silently on every call, and the whole game is then mute with every test still green.
5. **The real game survives with audio wired.** `-mission 10` ran windowed for 90 seconds and
   `-mission 20` for another 90 without a crash or a line on either stream, and `-check` prints its
   usual summary on both roots.

**No trace of an actual play was produced**, and the reason is not the sound path: no fight starts
on its own. Ninety seconds of each of two missions produced zero entities entering the charging
phase, because the party stands where it is placed until a player orders it. Reaching a blow needs
the mouse.

What is left unverified is the last hop alone: that the bytes handed to the playback library become
audible, at the right moment, at a sensible volume. **That needs a person, the build in
`builds/0126-sound/` and its README.**

## The gate

Run on the committed tree from the worktree, with `bash`.

```
go build ./...   -> no output
go vet ./...     -> no output
gofmt -l $(git ls-files '*.go') pkg/game/soundseam_test.go
                 -> no output   (the second argument is deliberate: git ls-files
                    does not list a file added in the same change)
go test -trimpath -count=1 ./...   -> 32 packages ok, 4 with no test files, 0 FAIL
bash scripts/check-no-game-assets.sh -> check-no-game-assets: clean (tree scan)
bash scripts/check-doc-budget.sh     -> every artifact ok; spec 11422/13312,
                                        plan 9038, tasks T1 1173 / T2 1070 / T3 1116
bash scripts/check-sdd-audit.sh      -> FAIL set empty for 0126 once this file exists
```

`check-sdd-audit`'s note and warning **counts** are meaningless from a worktree, which has no
`builds/`; only the FAIL set is comparable. Its two 0126 FAILs before this stage were "there is no
verification.md" and "plan.md accounts for no: FR3 FR9"; the second was a real omission and the plan
now cites both.

## Reverting to learn what is witnessed

Five lines were broken, the suite run, then restored. Four were already witnessed. **One was not,
and finding it is the reason this instrument is used instead of reading the assertions.**

| Line reverted | Result |
|---|---|
| `gruntSlot`'s floor, `hp <= GruntFloor` to `hp < GruntFloor` | `TestGruntSlot/at_the_floor_plays_nothing`: `ok = true, want false` |
| the throttle window, `< GruntThrottle` to `< 0` | `TestFR3ThrottlesGruntsPerVictim/100ms_apart`: `plays = 2, want 1` |
| the swing tick, `== AttackDelay` to `>= AttackDelay` | `TestSwingSoundFiresOncePerRun`: `14 emission(s), want exactly 1` |
| `stepSound` unhooked from `stepNumerals` | five tests fail, covering FR-1, FR-2, FR-3 and FR-6 |
| **`push`'s fill of `MapEntity.Sound`, replaced with `nil`** | **the whole suite stayed green** |

The last one is a defect in the story as the executors left it. Every grunt in the game runs through
that one expression — the drawing tier reads the slots by index and can decide nothing without them
— and nothing asserted it. `pkg/game/soundseam_test.go` now does: the slots reach the seam whole, in
order, by the entity's **own** class id (a second class row is present in the fixture so a lookup
keyed on anything else would show), and a class the table does not name reaches it as `nil`. With
that test in place the same revert fails: `Sound = [], want [11 0 22 33 44]`.

A sixth revert was written after the fix rather than before it: `audio.Stereo`'s new master clamp,
removed, fails `TestStereoClampsTheMaster`.

## The second reconciliation finding

`audio.Stereo` was not total over its master volume, which arrives from a command-line integer with
no range of its own. `-volume -100` did not quieten the game: it inverted every sample and played at
the same loudness in opposite phase. `-volume 500` amplified into the sample clamp, which is audible
clipping rather than a louder game. Both are now folded into `[0, MasterUnit]` in the one code path
FR-8 already names. Refusing the value was rejected: FR-10 and P-2 make every failure here degrade
to silence, and a startup refused over a volume would be the only thing in this story able to stop a
game from running.

## The fog gate, decided during the merge and named as ours

The fog hotfix (`637e513`) landed under this branch, in the file this story hooks into, carrying a
structural fence that refuses any function naming `v.entities` and not classified in `fogWalkTable`.
`stepSound` is one, so the merge asked a question this story had not: **does a blow inside the fog
make a sound?**

Nothing in the pin answers it: whether the original's engine sends a health level for a unit the
client cannot see is a network question no claim reaches. So this is **ours**, decided by one
argument — `ingestDamage`, the same walk over the same values one file over, is gated because the
owner ruled that no enemy indicator survives the fog. A numeral suppressed and a grunt played at the
same instant hand back exactly what the suppression withholds, and a positional sound carries the
direction too. Gated, therefore.

It is **one rule reached twice**, and the second reach is worth recording. The swing never passes
through `stepSound` — it arrives from the tier owning the attack-run counter as a bare cell. A first
attempt gated `playSlotAt` on the **cell** alone, which silenced a player's own unit in a cell the
plane called dark while the numeral for that blow still drew; the test caught it. `PlaySlotAt` now
carries the sounding unit's **owner**, so both sounds take `fogGateEntity`'s exception for the local
participant. `pkg/ui/soundfog_test.go` witnesses both: removing `playSlotAt`'s gate fails
`TestASwingInTheFogIsSilent`, removing `stepSound`'s fails the hotfix's own fence.

## Where each acceptance criterion, property and scope claim is answered

Every AC below is a test that exists and passes; the reverts above are what say the tests are
attached to the code rather than beside it.

| Spec | Answered by |
|---|---|
| AC-1, AC-2 | `pkg/ui` `TestFR1PlaysTheAtOrAboveHalfGrunt`, `TestFR1PlaysTheBelowHalfGrunt` |
| AC-3 | `TestFR1AHealthThatDidNotFallPlaysNothing`, unchanged and higher |
| AC-4 | `TestFR2RefusesAtTheFloorAndPlaysOneAboveIt`, both arms |
| AC-5 | `TestFR3ThrottlesGruntsPerVictim`, 100 ms and 1600 ms |
| AC-6 | `TestFR4TheFirstSightingPlaysNothing` |
| AC-7 | `TestAC7AShortOrZeroClassArrayPlaysNothingAndDoesNotFail` |
| AC-8 | `TestFR6SoundIsNotGatedByTheNumeralToggle` |
| AC-9 | `pkg/audio` `TestPlaceLeftRightAndOnTheListener`, `TestPlaceRefusesAtOrBeyondFalloff`, `TestPlaceClampsGainToTheUnit` |
| AC-10 | `TestStereoMuteAndZeroMasterYieldNoPlay`, `TestStereoHalvingMasterHalvesAmplitude`, plus the new `TestStereoClampsTheMaster` |
| AC-11 | `TestAC11ANilDeviceAndANilBankAreSilentAndFailNothing` |
| AC-12 | `TestDecodeWAV16BitMono`, `TestDecodeWAV8BitMono`, `TestDecodeWAV16BitStereoDownmix`, `TestDecodeWAVAcceptsEitherChunkOrder`, `TestDecodeWAVRefusesNonPCMFormat`, `TestDecodeWAVRefusesMissingDataChunk`, `TestDecodeWAVRefusesTruncationWithoutPanicking` |
| AC-13 | `TestResampleHalvesFrameCountAtTwiceRate`, `TestDecodeWAVResamplesToDeviceRate`, `TestResampleIdentityReturnsSameBackingArray` |
| AC-14 | `pkg/game` `TestSwingSoundFiresOncePerRun`, revert-witnessed above |
| AC-15 | the gate: 32 packages green with no install present, `check-no-game-assets` clean |
| P-1 | the same inputs give the same plays — every test above is deterministic and none reads a clock it was not handed |
| P-2 | `pkg/game` `TestOpenSounds` on a missing and on a malformed archive, `pkg/ui`'s nil device and nil bank, `pkg/audio`'s refusals, and the master clamp: no input in this story can fail a start, a map open or a frame |
| P-3 | `pkg/audio`'s resampler returns its input unchanged at the device rate, and `SoundBank.Sample` caches both a decode and a failure, so a repeated play decodes nothing |
| SC-1 | nothing in the story loads ambience, music, interface, spell or death sound |
| SC-2 | the device plays and forgets; no limit, priority or stealing exists to test |
| SC-3 | element 4 is named in the slot constants and read nowhere; the census above shows it nonzero in all 34 classes |
| SC-4 | the two flags are the whole surface; there is no in-game control |

## What was cut, and what was not

Nothing on the brief's cut list was cut. The swing landed, the positional pan and attenuation
landed, the throttle landed. The swing turned out to be the cheap half rather than the expensive
one: `pkg/game`'s `advanceSwings` already keeps the attack-run counter the decoded rule reads, so
the emission is four lines inside a walk that already ran.

Three things are absent by contract rather than for want of time, and each is named in `spec.md`:

- **The unchanged-health grunt** (element 1). The original's client is told a blow arrived; this
  tree sees only health levels, so a blow for zero and no blow are the same observation. Refused
  rather than approximated (DD-8).
- **Element 4.** Nonzero in all 34 classes and read by no hook any claim in the pin names. Left
  unused rather than guessed at (SC-3).
- **Any channel policy** — no voice limit, no priority, no stealing (SC-2). Enough simultaneous
  blows will pile up.

And one timing is a **disclosed divergence** rather than a reproduction. `ANIM-CLOCK-024` says the
swing frame, the swing sound and the damage are scheduled from three different numbers in the
original, and this tree's attack cycle already binds two of them. The counter and the threshold this
story tests are the original's own; the cycle they ride is ours (DD-7). It is stated at the emission
site as well as here.

## Two premises of the brief that were wrong

**This story does add a module.** The brief said it adds no dependency because `ebiten/v2` already
ships `ebiten/v2/audio` in the same download, which is true of the source and not of the module
graph: the first import of that subpackage pulls `github.com/ebitengine/oto/v3 v3.4.0` in as a new
**indirect** requirement, absent from `go.sum` at the branch point. It is Apache-2.0, its licence
text was already under `LICENSES/`, and `THIRD_PARTY_NOTICES.md` gained a row — which
`internal/notices` cross-checks on every run, so it could not have gone unnoticed.

**`plan.md`'s `SetAudio` signature was unbuildable.** It gave the viewer the settings, but
`audio.Player.Play` takes none, so the viewer had no path to spend them. Resolved by giving the
settings to the concrete device — `OpenAudio(st audio.Settings)` — and leaving the viewer's setter
two arguments. FR-8 is unaffected: `audio.Settings` is still the one named place and `audio.Stereo`
still the one code path.

## Open

- **Nobody has heard it.** See the top of this file.
- `OpenAudio` and its `Play` are the one part of this story no test touches, deliberately: a test
  that opened a device would fail on any machine without one. What stands in for one is that every
  guard lives on `playSlotAt`, where AC-11 sees it, and the by-hand drive described above.
- On a map's very first push the entities carry no slots — `setSwingSound` runs after the
  constructor's push — so a blow on frame zero would be silent. None can land there; recorded, not
  fixed.
- The listener is the middle of the visible tile range. The original's is not decoded, and a decode
  of it would move `listenerCell` and nothing else.
- The attenuation is linear in cell distance. The original's is an `FSQRT` distance through a
  `log10` whose constants are not decoded; only the clamp at 10000 is taken. `FalloffCells` and
  `PanCells` are ours and are the two numbers a later decode would move.

# Story `1038` — closure

Map-placed humanoids now resolve their indexed world-body pixels through the faction shade selected
by the placement's decoded owner slot. The production frame passed to the ordinary unlit and lit
blits carries the selected table; classes outside the shared human family retain their own palette.
Stone Curse takes precedence through corpse stage 2 and reaches the existing fixed neutral path
from the un-repaletted base frame. At stages 3 and 4 the effect may remain attached, but the draw
resumes its ordinary frame and owner shade. The as-built behaviour is canonical in `spec.md`.

The pointable result is mission 20 on either preserved install. Entity 12, class 14, owner 3 uses
shade table 3. Its opaque pixel 555 has palette index 124: the base colour `{38,40,85}` becomes
`{86,57,39}` in both `StaticFrame.RGBA` and `StaticFrame.RGBALit`. The runnable headless build and
the command actually used to open that mission are under `builds/1038-faction-shade/`.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `pal.DecodeOwnerTables` accepts the exact 16,384-byte headerless resource and decodes all sixteen 256-entry BGR tables. The exhaustive short-length test and representative overlong cases reject every other supported shape without a partial result. `LoadUnits` carries the registry's exact `Palette == 0` arm; the release census independently parses `units.reg` and finds the same 18 shipped classes on each root. |
| Runtime state | PASS | The already-decoded `alm.Unit.Owner` remains the 1-based type-5 roster slot as it crosses `mapload` into `sim.Entity.Owner`. `mapWorld` adds only a presentation cache keyed by base-frame pointer and low-nibble shade. Owners 1 and 17 share an identity, while owners 1 and 2 do not. |
| Simulation | N/A | No simulation field, command, rule, byte form or digest changes. The selector reads the existing owner value and produces only a client frame. The full repository's digest and import-boundary tests remain green. |
| Player input | N/A | No gesture, hit test, key or command path changes. The existing `ui.MapEntity` frame is changed before the same viewer receives it. |
| AI | N/A | No target, path, order or cadence is read or written by palette selection. Mission-drive movement stays at the baseline. |
| UI/HUD | PASS | `entityDraws` applies the owner table after class, tier, body, corpse and animation selection, at the production seam that feeds the map viewer. `RGBA` and `RGBALit` both produce the independently decoded mission 20 colour. An attached Stone effect retains the base frame and sets the returned presentation flag only at decay stages 0 through 2; stages 3 and 4 return to the ordinary frame and owner shade. |
| Triggers/scripts | N/A | No script opcode or trigger changes. Missions 10 and 20 retain 0 unsupported nodes and the baseline 16/27/12 and 14/15/11 script counts. |
| Inventory/equipment | N/A | No inventory or equipment state changes. The paper doll is a separate compositor under `PAL-FIGURE-014`; `TestAHeroBodyRetainsOwnerShadeEligibility` proves the production body loader's structural copy retains a positive owner-shade flag. |
| Persistence/save-load | N/A | No persisted field or form version changes. `ownerFrames` is derivable cache state, recorded in story 0143's field census. `TestCandidateBodyCacheIsAdoptedOnlyAtCommit` restores a mission through the production candidate boundary and proves the committed/live set retains `HasOwnerPalettes`, two sentinel entries and its owner-shaded body. |
| Campaign/session | PASS | The release witness opens every campaign mission through `MissionOpenerWith` on EN and RU. Owner values survive the real map-to-world path; the story does not add or claim a multiplayer allocation or reconnect implementation. |
| Shipped content | PASS | Each root reports 24 missions, 1,968 map placements, 347 eligible and visible placements and 0 hidden eligible placements. The complete slot census is `[0 8 120 55 12 24 71 52 5 0 0 0 0 0 0 0]` on both roots. |
| Interactions with existing mechanics | PASS | Normal, attacking, dying and bone selection converge on one final application site. Corpse art is judged by the substituted class; tier-palette classes remain excluded; equal palettes retain the base pointer; different shades retain distinct texture identities; indexed pixels and the base frame remain shared and unchanged. Missing or refused cosmetic data returns the exact base frame. Real casts under owners 1 and 2 prove the Stone override for live, attack, fallen and first-bone frames, then ordinary owner shading for both later bones while the effect remains attached. |

No known in-scope GAP remains. The unresolved original global sprite table is not a gap in this
palette result: `DIV-353` records it as an adjacent UNKNOWN and the story does not claim sprite-source
fidelity.

## Production and integration witnesses

`TestReleaseCampaignMapHumanoidsUseTheirOwnerShade` opens all 24 campaign missions through the
production front end. Its expectation does not reuse the production palette decoder: it reads the
raw archive bytes as B, G, R, reserved entries, independently parses `units.reg`, and then limits the
entity census to ids minted by the map's own placements. For every visible eligible placement it
compares the selected frame's complete 256-entry palette with the independently decoded owner table.

The same test finds a changed opaque pixel in real mission 20 and hands that selected frame to both
canonical pixel walks. On both installs the witness is:

| Root | Missions | Placements | Eligible | Visible | Hidden | Mission 20 witness |
|---|---:|---:|---:|---:|---:|---|
| EN | 24 | 1,968 | 347 | 347 | 0 | entity 12, class 14, owner/shade 3, pixel 555, index 124, `{38,40,85}` to `{86,57,39}` |
| RU | 24 | 1,968 | 347 | 347 | 0 | entity 12, class 14, owner/shade 3, pixel 555, index 124, `{38,40,85}` to `{86,57,39}` |

This is a composed-frame witness through the code's production mission opener, frame selector and
canonical blits. No window was launched and no human screen observation is claimed; the workflow
requires the code to be the instrument on the owner's desktop.

`TestStoneOwnerClassesFollowTheResearchedDecayGate` drives real spell-20 casts through the same
`entityDraws` seam, then uses canonical damage commands to enumerate decay stages 0 through 4.
Live, attack, fallen and first-bone selections return `Stone` true and their exact base frame. The
two later bones retain the attached effect in the world but return `Stone` false and the selected
owner frame. Owners 1 and 2 exercise different tables; late-bone frames pass through both `RGBA`
and `RGBALit` and retain one stable cache identity across repeated snapshots.

`TestStoneDecayGateUsesTheSelectedBodyClass` adds a selected hero body and its corpse, a non-owner
corpse and a tier-palette corpse. The first resumes owner shading at stage 3; the excluded corpse
classes retain the exact frame selected before the central gate. The established UI-side
`TestGrayscaleRGBAUsesFixedLuminanceAndPreservesAlpha` pins the neutral transform used through
stage 2.

The runnable `againrom.exe` was built from this branch and run headlessly with the EN root as
`-check -mission 20`. It reported a 144-by-144 mission, 57 entities, party at `(13,14)`, 11 raises,
and a valid successor into chapter 30.

## Mutation proof

The production use-site mutation removed the single call which assigns
`draw.Frame = mw.ownerFrame(...)` inside `entityDraws`. The focused seam test then failed for owners
1, 2 and 17: every composed pixel stayed at the base `{1,2,3}`, both owner-shaded frames retained
the base identity, and different shades aliased. Restoring the call restored the test.

`pkg/game/world.go` had SHA-256
`B05421B73ACE7EF11E1584F6DC3BAE07749117DD225554E4C4190F58F3ED0365` before the mutation and the
same digest after the revert. The test therefore kills removal of the production consumer rather
than pinning a helper in isolation.

The first adversarial correction's mutation applied owner shading unconditionally. Its focused
Stone witness failed immediately because owner 1 crossed with an owner-frame pointer instead of the
neutral base pointer.

The pass-2 correction mutation removed the one decay-stage narrowing statement, restoring the old
effect-only broad guard. Four late-bone cases then failed: owners 1 and 2 returned `Stone == true`
at stages 3 and 4. The hero-corpse, non-owner-corpse and tier-corpse cases failed beside them. The
focused command exited 1; restoring the statement made it exit 0. `pkg/game/world.go` has SHA-256
`40113E73A34D4713994E6BF0F3AB48CD5F4BA1D4F09BB6D5E8CBF597DACD6895`, identical before and after
that mutation.

## Research reconciliation

The build follows `PAL-RULE-021`'s owner-slot mechanism at High and retains its
shipped-single-player reach at Medium. The complete shipped census proves this build's wiring; it
does not raise the claim about ROM1. `PAL-FIRST-022` supports first-writer slot stability, while
`PAL-JOIN-023` supports the low-nibble fallback without promoting its Medium reconnect name.
`PAL-BLIT-024` supplies the High rule that the selected palette reaches all five original
world-body blits.

`PAL-OWN-007`, `PAL-SLOT-015` and `PAL-BAND-016` supply the resource layout, sixteen-slot cycle and
exact `Palette == 0` class arm. The implementation carries their amended state and does not repeat
the retracted missing-writer premise. `PAL-SHADE-012`, `PAL-SHADE-013` and `PAL-FIGURE-014` keep the
selected object and paper-doll boundary explicit.

`MAGIC-STONEDRAW-084` gives Stone Curse's later palette override: for `Palette == 0` the original
replaces the owner selection with one fixed neutral table. Both its frame and palette overrides are
gated on `drawable+0x15a <= 2`. `REG-UNITS-050` identifies that field as the corpse stage: 0 live,
1 fallen, 2 first bone, with stages 3 and 4 the later bones. The gates and arguments are High; the
statement that both decoded forms render grey remains Medium. This build does not promote that
grade: it preserves its existing grayscale compositor, applies the same stage gate to the returned
presentation flag and resumes ordinary owner shading outside it.

`UNIT-SPRITE-042` and its direct dependency `UNIT-PICT-035` establish the original's picture branch
but leave the clear-side global table's content and index Unknown. This build already chooses body
art from the resolved class sheet. Owner shading changes only the palette of that selected indexed
frame, so `DIV-353` records the unidentified sprite-source seam instead of inventing it or claiming
equivalence.

## Divergence and backlog disposition

`DIV-048` is closed because its exact revisit condition is now met: research answers the map-faction
shade ordinal and the ordinary production world-body blit consumes it, with both-root shipped
evidence. Stone's researched fixed neutral override takes precedence through stage 2; the ordinary
owner consumer resumes for stages 3 and 4 without reopening the debt.
`DIV-353` is the only reserved id spent. It remains OPEN UNKNOWN for the unidentified global-table
sprite source; `DIV-354` through `DIV-360` remain unused. No other mismatch was found.

BACKLOG row 12 was re-measured against `e3466269`. Its behavioural gap, "shade decoded but not
applied", is discharged by this story, and its research-open wording is stale under
`PAL-RULE-021`. `pipeline/BACKLOG.md` belongs to the seat outside the implementation repository, so
the landing must remove or rewrite that row there.

## Gates

The clean implementation chain reports `go build ./...` 0, `go vet ./...` 0, `gofmt -l` empty and
`go test -trimpath -count=1 ./...` 0. `go list ./...` reports 66 packages, 43 with test files.
The check-script glob selects two scripts: `check-claim-citations.sh` resolves 1,289 distinct
citations against 1,479 claims and 220 experiments, and `check-no-game-assets.sh` reports a clean
tree. Its explicit `--history` run is also clean.

The gated-test manifest grows from 44 to 45 functions for this story's one release census. The seat
release gate selects 47 executions on each root and runs all 47, with 0 skipped. The scenario gate
selects and passes 15 of 15 on each root. `check-div-claims.sh`, pointed at this worktree, selects all
232 live rows and validates 283 cited claim ids with a nine-cell header. The preserved-install gate
reports 162 files unchanged.

The required one-tick `missionrun` census on EN reports 0 unsupported nodes for missions 10 and 20.
The milestone drive is byte-for-byte at its presentation-independent baseline on both roots:
mission 10 has 16 checks, 27 instants and 12 triggers; mission 20 has 14 checks, 15 instants and 11
triggers; the 240-tick drive has 4 of 36 units moved and 1 fallen. A palette-only story is expected
to leave these numbers unchanged.

## Remaining review surface

Pass 1 returned the owner-dependent Stone grayscale; pass 2 proved that correction too broad by
joining `MAGIC-STONEDRAW-084`'s gate with `REG-UNITS-050`'s field identity. This correction closes
the repeated P class by enumerating its population: live, attack, fallen, first bone, both later
bones, two owners, hero and substituted corpse eligibility, tier exclusion, both canonical blits
and cache identity. The correction delta remaining for pass 3 is the one central stage predicate,
its returned `MapEntity.Stone` value, the owner-frame path outside it and the canonicalized
specification, closure and divergence rows. No other production surface remains unenumerated.

Pass 1's W copy-boundary witness remains closed by the hero-body and candidate-restore assertions.
Its D wording remains closed by distinguishing this build's class-sheet selection from `DIV-353`'s
unresolved original branch. A fresh adversarial pass 3 is still required for the corrected tip;
this closure does not claim its verdict.

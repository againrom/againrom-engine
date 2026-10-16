# Projectile phase sound

## Intent and authority

Earthquake's accepted cell sound follows its common projectile record after a
fresh cast and an ordinary SAV LOAD. This is one bounded C4/C5 result.

Base: `4ad1b8f2ad091b349c71242842dd7b7bd1e75dc3`. Knowledge:
`d2a0c9f428ecfa0e28899950039c332f25623ce8` (k315).

`ANIM-148` gives picture 51 an initial actionphase of -1 and 16 segments.
`ANIM-149` gives its phase counter and sound 551 at actionphase 8. Its
UnionRect amendment leaves the separate prologue callback and destructor
unresolved; the pinned retraction ledger has no separate ANIM-149 entry.
`MAGIC-BURSTLIFE-034` retains stationary burst lifetimes and sound. Its partial
retraction corrects the stage interval to three ticks; it does not change the
local sound boundary.

## Result

The common `sim.stepSavedProjectile` driver reports picture 51's live
actionphase-8 boundary through the existing transient `Report` sink. The game
dispatches that event through `queueSpellSound` with zero delay and the existing
Viewer audio adapter. Staged construction no longer queues a separate first-game
picture-51 timer. Other staged sounds retain their existing route.

The same driver handles a fresh record and a restored SAV record. LOAD before
the boundary reaches the cue; LOAD at or beyond the boundary does not replay it.
A zero-segment removal emits no cue, and a replacement record follows its own
phase. No World field, save field or format changes. Restored owner handling and
the existing cell, fog, visibility and audio admission gates remain in force.

## Proof

The baseline regression sounded a fresh record at actionphase/frame 7, lost the
cue after ordinary SAV LOAD at actionphase 7, and sounded a retired identity's
queued cue while its replacement was at actionphase 1. LOAD controls at
actionphase 8 and 9 remained silent.

Focused GREEN covers a production cast, actual record phases and lifetime,
ordinary SAV/cold LOAD at phases 7, 8 and 9, the last positive segment,
retirement/replacement, paused frames and redraws. Observed and silent runs keep
equal World bytes/hash; the fresh cast also keeps equal projectile stores/hash.

Installed EN and RU App casts each produce 32 distinct phase-8 record cues and
32 admitted `registry:551` backend requests through the existing adapter.
Device players are deterministic; this proves dispatch, not audible output.
Both roots retain the first-game slot and shared-delivery controlled witnesses.

Ordinary `pkg/game` and `pkg/sim` tests pass. Strict architecture, storyguard and
gated-population checks pass. The comment baseline falls by 46 bytes; composition
baselines do not change. `go build ./...`, asset and preserved-install guards,
and divergence-claim checks pass. The sole review, final chain, main landing and
build promotion belong to the root handoff and remain pending.

Private receipts are in `review/c4-c5-completion/1442/`: `focused-red.txt`,
`focused-green-attempt4.txt`, `packages-attempt1.txt`, `strict-attempt2.txt`,
`installed-{en,ru}-attempt1.txt`, `shared-{en,ru}-attempt1.txt`, and the per-root
phase/delivery JSON. The package receipt's only failed leg was the measured
comment fall; `strict-attempt2.txt` verifies the lowered baseline.

Published editor reconciliation preserves every phase-sound Go blob except
the falling comment baseline. `integrated-focused.txt` and
`integrated-ordinary.txt` pass on that combined source: game91.132s,
sim16.299s, archtest20.508s, storyguard3.968s and gatedtests1.615s.

## Open debt

Camera shake amplitude, the separate prologue callback and destructor,
attachment/hash lifetime, transport's extra tick and scorch callback authority
remain Unknown. This result changes no canonical scorch or other spell rule.
DIV-944, DIV-1814 and DIV-501 retain their separate open scopes. Closed
DIV-1786 and DIV-1878 remain closed.

The ledger also removes stale engine-description clauses from DIV-1782 and
DIV-1877. Parent-own array inheritance, tiled custom-radius bursts and visible
canonical scorch already exist. Their original fault recovery, notification
timing and view-hash boundaries remain open. This refresh changes no code or
ROM1 authority.

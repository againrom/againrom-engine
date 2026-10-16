# Target-bound Heal and Drain marks

## Result

A retained positive-count ROM1 projectile of picture 20 or 30 refreshes its
resolved actor's presentation mark. This works after ordinary SAV and fresh-App
LOAD without a CastEvent or legacy Heals residue. The actor carries one element
per picture; its mark follows the current actor pose and outlives the projectile.

## Authority

Knowledge pin `5dd731dce555a554c88c6990eedc9fa60cbd792d` supplies
MAGIC-MARK-059/060/061 and amended ANIM-149. The supported contract is a packed
kind/countdown pair, by-kind replacement at 32 for the resolved 20/30 branch,
decrement/removal during rebuild, distinct Heal/Drain builders and actor-relative
records split around the actor by signed depth. ANIM-149's target lifecycle
Unknown does not establish an elapsed-time cadence or detailed particle recurrence.
ANIM-183..186 bound the selected notification receiver prefixes; concrete
dynamic callback effects remain Unknown. They add no inferred gameplay rule.

## Implementation

The projectile driver reports the resolved target at the production branch.
Constructor calls inside Step use the same report; the existing frontend release
adapter receives the same observation through a short-lived callback. Neither
observation changes World bytes, projectile leaves, IDs, allocator or drivers.
EntityID zero is a valid actor when its runtime binding resolves.

`mapWorld.advanceEffectMarks` remains the countdown owner. It decrements existing
elements, removes expired or stale target bindings, then refreshes observed kinds
at 32. The existing shower builder accepts this countdown and the actor class's
TileSize. `markDraws` sends its records through the existing actor mark passes.
Legacy fixed-anchor Heals use the same geometry builder and retain their old
fields and defaults. A new target-bound producer never also creates legacy Heals. Fresh semantic
Heal/Drain observations, including weapon riders with no projectile, feed the
same by-kind owner. Their current resolved binding is distinct from the binding
captured by the retained projectile branch. Duplicate observations refresh one
element and use the same target-identity seed.

The ordinary NativeActions supplement stores each target ID, packed element and
presentation seed. Existing actor bindings remap IDs on LOAD. No new save producer
or format exists. Missing new fields mean no target marks. A new map owns a fresh
mark list; changed source identity, off-map actors and disappeared targets cannot
inherit old records. Projectile retirement does not control mark expiry.

## Proof

`TestRetainedProjectileTargetMarkColdLOAD` enters through ordinary SAVE and a new
App's LOAD. It checks all World bytes and hashes, the complete 16-leaf projectile
store, IDs, allocator and driver equality. Its five controls cover resolved 20,
resolved 30, zero count, unresolved target and another picture. Resolved branches
produce marks with no commands, CastEvents or legacy Heals. A second SAVE/cold
LOAD retains newly created marks after projectile retirement, all remaining
counter phases, expiry and the next ordinary move command.

Focused simulation tests cover constructor and retained calls, actor zero,
negative controls and byte-identical observed/unobserved Worlds. Game tests cover
same-kind refresh, independent kinds, old-input defaults, corrupt packed values
and stale target replacement/removal. UI tests check current fine-position
displacement and both passes around the actor.

`TestReleaseRetainedProjectileTargetMarksInstalledArt` reads EN/RU graphics through
`OpenContainers`, `LoadProjectiles` and `LoadUnits` into the synthetic front. The fixture maps its synthetic class zero to a loaded actor sheet and reveals
fog. It uses ordinary SAV/cold-App LOAD and checks actual actor-art submissions
after the projectile retires. Matching target sheet-frame identities excludes picture 30's
ordinary projectile sprite as a false positive. It constructs no default audio
device and opens no game window. This is installed-art validation, not evidence
of shipped producer reachability for the synthetic positive-count input.

## Bounds

One rebuild per engine tick, decrement-before-refresh ordering, deterministic
presentation seed and the retained particle policy are engine choices. Detailed
native 20/30 particle recurrence and rebuild-to-seconds cadence remain Unknown.
DIV-3418 records the engine timing policy; DIV-944 retains the geometry boundary.
The render uses the actor pose exposed by the existing unit placement seam; no
new native position field is inferred. Final chains, the sole adversarial review,
publication and promotion belong to the seat.

## Validation

The focused EN and RU selections each passed seven top-level tests and seven
subtests, including both installed pictures. Ordinary package tests passed:
`pkg/game` 3059 tests, 1119 skips; `pkg/sim` 2759 tests; `pkg/ui` 1912 tests,
11 skips; `pkg/render/terrain` passed. Ordinary game/UI used
`AGAINROM_TEST_VOLUME=0`; device initialization was permitted. The game package
ran on the host because the sandbox forbids its synthetic junction/hardlink
fixtures. Private receipts include SHA-256 pins for 3402 package Go files;
none changed during final validation.

## Final-gate correction

The first prepared landing failed four EN/RU release parents: ordinary scenario
flight, Lightning launch and both Prismatic Spray ray witnesses. The producer
and frontend adapter had treated actor ID zero as a homing target even when
its native key was zero; ReleaseCast refused the record before allocation.
One correction preserves fixed aim for this keyless-zero case. An encodable
actor zero still homes, and an absent nonzero target is still refused. The
manual Lightning point-cast witness now states AtCell explicitly and keeps
its geometry assertions. Added loss controls fail before the correction and
check allocation, native key and later movement. Failed prepared evidence
remains preserved; no second adversarial pass is commissioned.

Corrected frozen sources pass all four failed parents and the target-mark
selections on EN and RU. Ordinary sim passes 2760 top-level tests; ordinary
game passes 3060 with1119 skips on the host. SHA-256 verification covers2717
game/sim Go files with no changes during validation. The initial sandbox game
attempt retains its fixture AccessDenied failures and partial sim PASS.

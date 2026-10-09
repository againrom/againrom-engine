# Escort residues

## Intent

Defend heals its charge before cover engagement. Idle AI escorts turn. Close
orders follow the charge between actor decisions. DIV-007 is narrowed to the
bounded choices recorded below. Pursuit search, save production and widgets
are outside this change.

## Authority

The pinned claim tool supplies AI-FOLLOW-112, AI-FOLLOWTAB-113,
AI-FOLLOWGAP-114, AI-FOLLOWRANGE-115, AI-FOLLOWSET-116, AI-FOLLOWAUTH-117,
AI-FOLLOWHEAL-118, AI-FOLLOWDEATH-119, AI-DEFEND-111, AI-ORDER-039,
AI-TICK-008, AI-ACQUIRE-002, AI-BREAK-041 and AI-FILTER-001. Additional
authority is HERO-MP-006 for the mana floor, AI-TURN-104 and AI-RANGE-102
for the idle gate and range, AI-RETAL-056 for the blow flag, AI-395 for the
persistent idle order, and AI-PURSUE-040, AI-371 and AI-417 for the shared
close executor and centred body cadence. AI-RAND-058 identifies the original
generator; DIV-027 retains the engine generator.

## Behaviour

Within escort range, a defender with a book selects spell 6 when its mana
floor is zero and its charge is damaged, or when the floor is less than its
maximum mana plus 3 and the charge is strictly below half maximum health.
The instance cost must not exceed current mana. Equality is admitted. A
missing spell, insufficient mana or refused action falls through to cover.
An admitted one-shot book cast owns the defender through wind-up and
recovery. Follow never uses this fork. Installed spell row 6 resolves through
the existing table and book decoder as Heal.

An empty standing acquisition gives an AI-owned escort a persistent idle
order. Each eligible centred body update tests the blow flag or a 15-bit
draw below 205. A successful gate consumes the flag and requests current
facing plus 33 plus the integer quotient of 190 times a second 15-bit draw
over 32768, truncated to a byte. A flagged update omits the gate draw.
Existing turn scheduling limits the body to one facing step per update.
Human-owned escorts retain their existing ordinary healing route.

A close order reads the charge's current cell on each eligible centred body
update. A changed cell invalidates the old route. Inside the stop distance
the escort stands and faces its charge. The close order survives this stop,
so a moving charge can reopen the gap before the next actor decision.
Transit, imported active motion, Stone Curse, cast ownership and loaded
physical progress retain their existing body gates. The stop test uses the
shared edge distance; the actor arm still uses its Chebyshev range test.
Step-away remains a walk to a fixed cell.

## Persistence

EscortOrder distinguishes close and idle from an ordinary walk.
EscortTurnPending carries the blow flag for fresh native actors. Both enter
typed ActionContinuations and current SAV. Sparse native suffix ESC1 uses
form 116; absence retains the historical bytes and gives both fields zero.
Invalid enum values, flags, actor references, counts and spans fail bounded
atomic decoding. Original escort orders seed the discriminator from their
order byte, never from destination coordinates.

## Proof

The three initial regression tests fail on the base: both positive heal
branches select combat, idle keeps its facing for 640 sub-ticks, and close
retains the old charge cell. Focused tests cover the half-health boundary,
full health, reserve boundary, mana equality, instance cost on a non-mage,
fight fall-through, cast ownership, Follow exclusion, forced idle flag,
transit exclusion, stop and immediate resumption.

TestEscortResiduesSurviveBinaryAndActionContinuations compares native and
typed cold continuations. The historical-form, field-loss and malformed-input
tests cover deterministic zeros, independently missing fields and atomic
refusal. Installed witnesses are
TestReleaseMissionEscortCloseReaimSAVColdLoadAndNative,
TestReleaseMissionEscortIdleTurnSAVColdLoadAndNative and
TestReleasePlayerDefenderInstalledHealSAVColdLoadAndNative. They exercise
mission 100's script Follow and mission 10's player Defend, F2 SAVE, cold
menu LOAD, engine save, continued hashes and SAV field-loss controls on each
release root. Placement, centring and the idle alarm are explicit fixtures;
the map, actors, spell rows, orders and save routes are installed production
inputs. No game window is used.

The handoff records exact revision, commands, elapsed times, release
population, milestone comparison and scenario results. Fixture-only tests
do not establish installed validation.

## Open debt

DIV-2600 records the absent mana floor on historical native actors: explicit
player policy supplies it, otherwise the existing quarter-pool default does.
DIV-2601 records the Medium idle divisor choice of 32768. DIV-027 retains
the deterministic engine generator, which changes the random sequence.
Existing cover radius, preference, standing score, cell-based step-away and
missing-charge rules remain under DIV-558. No original runtime experiment
is claimed. DIV-2602 through DIV-2607 are unused.

# Native campaign save continuation

## Result and dependency

A player starts or enters a mission from the current campaign state, saves,
loads in a fresh process and continues with the same state and next actions.
Mission entry does not construct a SAV Document and import it as a substitute
for native initialization. This story uses story1220's single current-state
producer for every save. It introduces no writer, refusal fallback or AGS path.

The isolated branch started at published checkpoint
e776ba1261b95037169ae01bee8c97f9cb0e911f. Its candidate received the sole
fresh adversarial review and landed as merge `e86bedb`; the forward knowledge
pin correction is `46a49e5`. Later SAV corrections reached local engine main
`345d04a` without publication. The full Go measurement on that exact clean
commit had six `pkg/game` failures. Commit `7524b67` closed those roots,
including the generated city Spell supplement, absent-template kind41
construction and frozen current-envelope roots. The follow-up Human-tail
correction `de16770` now passes the complete clean `pkg/game` package. The
combined candidate `fa54eb9` carries these corrections, the correction pass
`6b1ee15` and the corpus directory skip; it merged to engine `main` as
`bda0711` on knowledge k72 and was promoted. This is a landing, not a
milestone verdict. M7/M8 remain open.

## Owned behavior

- Additional campaign sequences preserve current World, Item, Effect and
  Spell identities across city operations, mission entry, combat and return.
  Common entry initialization and existing witnesses belong to story1220.
- Campaign-specific defects found by this expanded sequence are fixed here
  using the shared producer, never a second source of current state.
- Every Main, Side and Offered campaign mission is exercised from installed
  EN and RU data. The population currently has28 distinct missions per locale.
- Full M8 coverage and remaining M7 acceptance remain beyond this landing. The
  original EN sample is still an owner-kit task after the internal continuation
  passes.
  M10's full EN/RU original matrix and M11's AGS removal remain later work.

## Change boundary

The common entry prerequisite stays in story1220: frontend routing, removal
of the fabricated generated Document, citymissionseed and the shared object
transfer. Their existing generated binding, hero-row, Group/order and combat
tests migrate with that code. Existing all28-mission EN/RU release coverage
and M2 stay required there; this split transfers no failing prerequisite gate.

This branch owned the additional sequence/acceptance population after that
parent: campaign-specific repairs and the current continuation proof. The
copied test migrations were reconciled into the landed common parent. The
candidate received one adversarial review and a serialized landing. The current
post-landing repair clears measured regressions before acceptance; it does not
create a second review pass or claim a completed promotion.

## Proof and open work

The short integration set covers a natural mission, combat with a dropped
Sack, a city operation and city-to-mission crossing, and an older saved world.
Each uses two SAV/cold-load cycles and verifies subsequent actions. Full
acceptance retains all28 missions per locale and all114 saved inputs on both
locales, positive/negative controls and unchanged strict comparison semantics.

The measured correction trajectory is retained because the failure count alone
hid one root. A whole-tree run at `7524b67` inherited an RU-only
`AGAINROM_ASSETS` value; its 488.183-second release failures are invalid as a
regression measurement. With both asset and corpus overrides removed, the
complete `pkg/game` package took 65.614 seconds and exposed two roots: the new
Human-tail carrier was private and therefore absent from gob/JSON clones, and
one spell-training test still expected the pre-LOAD graph before the generated
Spell supplement materialized. Making the carrier a map produced a second
64.859-second measurement with three roots: nondeterministic encoded order and
two validations that incorrectly required detached SAV residue to name a live
roster root. The corrected carrier is an exported, sorted `PartyID` slice
because the legacy AGS envelope remains readable until M11; ordinary SAV edits
rebuild it and the current actor remains authoritative. Retained residue may
outlive its roster root, while application still requires a current member of
the same identity. The resulting clean package run passed with zero failures in 64.521 seconds and `TestLiveTreeClean` passes. The first
paired release run then exposed older roots outside that carrier: AGS byte
determinism, a mission-position baseline mismatch, stale fresh-process camera
expectations, local-SAV dispatch in a structure fixture, and later structure
continuation differences. The baseline, camera expectation and local dispatch
are corrected; the remaining release roots keep M7/M8 open.

Test runner acceleration changes scheduling or redundant setup, never the
population, cold-process boundaries, tick windows or assertions. The runtime
report is review/story1220-test-runtime.md; the durable parallel locale helper
belongs to the seat pipeline. Earlier serial receipts remain preserved.

DIV-1377 through DIV-1380 returned unused; no new divergence is asserted
here. On `bda0711` gofmt, full Go, the asset guard and M2 pass; the M2 corpus
walkers skip `exp<N>-*` research resave directories by owner direction. EN/RU
release fails 44 entries, a subset of the 60 that fail on the previous main
`04a323b`; those 44 are open debt. Receipts:
`review/story1221-land-bda0711/`. The original EN sample has no accepted
result: the owner's generated mission-10 files crashed in original LOAD, and
research SAV-1087..1098 name the fields; story1222 adopts them. No original
acceptance is claimed.

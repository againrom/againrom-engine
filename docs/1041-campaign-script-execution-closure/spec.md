# Story `1041` — campaign-script execution closure: as-built spec

Canonical at landing. `contract.md` records the promised result and the pre-story measurements; this
file records the instrument that was built. It is an execution-evidence specification, not a claim
that every behaviour the instrument observes is faithful. `DIV-387` bounds the places where the
full node population is visited but the semantic oracle is not discriminating enough to prove every
claim clause.

## Domains touched

Assets, Sim Core, Campaign & Scripts, and AI & Orders. The effect oracles read Party, Items & Heroes,
Combat & Magic and Persistence state, but this story changes no mechanic in those domains. The one
production behaviour change is inside Campaign & Scripts: a build-time action now leaves no runtime
trigger slot instead of aliasing to instant zero.

## FR-1 — the shipped denominator

`cmd/scriptcoverage` opens a lawful install through `game.OpenArchives`, loads definitions, enumerates
the campaign archive, and starts every mission with `game.StartMission`. It refuses anything other
than 28 maps and the measured compiled populations:

| Root | Checks | Runtime instants | Accepted triggers | Reachable operation rows |
|---|---:|---:|---:|---:|
| EN | 680 | 759 | 398 | 52 |
| RU | 678 | 759 | 397 | 52 |

The stable node identity is `(root, map, family, authored node index)`. An opcode-6 node also carries
its first parameter as the group sub-command. The CSV distinguishes `node`, `operation` and
`synthetic` rows rather than folding three different populations into one count.

The 52 operation rows are the active `TRIG-CLOSURE-037` join, with the missing instant-13 statement
supplied by `TRIG-TAKEITEM-038`:

| Family | Rows | Operation keys |
|---|---:|---|
| runtime checks | 17 | 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 14, 15, 16, 17, 18, 19, 21 |
| build-time check | 1 | constant `0x10002` |
| runtime instants | 25 | 2, 3, 4, 5, 6, 7, 8, 10, 12, 13, 16, 17, 18, 19, 21, 22, 23, 24, 25, 28, 29, 30, 32, 33, 34 |
| build-time action | 1 | drop `0x10002` |
| reachable opcode-6 sub-dispatch | 8 | 2, 3, 4, 5, 10, 11, 14, 15 |

Checks are reachable because the ordinary pass evaluates every compiled check. Runtime instants and
their sub-commands are reachable only when an accepted trigger names their authored action id.
Authored actions 20 and group sub-command 1 have compiled nodes but no accepted trigger path; group
sub-commands 17 and 18 have no campaign node. They are not among the 52.

The validator requires every expected key and the measured compiled totals. It does not reject an
additional reachable operation whose key is absent from `claimOracle`; that denominator blind spot
is one bucket of `DIV-387`.

### The two builder rows

The constant is accepted into the compiled check list and presets its owned register during world
construction. It remains a build-time row because an ordinary pass must not overwrite a variable an
instant may since have changed.

The drop action is consumed by `mapload.CompileScriptFrom`. It appears neither in `Script.Instants`
nor in any compiled trigger slot. `CompileScriptFrom` now tracks build-time action ids separately and
leaves a slot `ScriptNone` when a trigger names one. The general missing-id rule remains unchanged:
an unknown runtime action id still takes the builder's documented miss-to-zero path when a runtime
instant exists. Thus a build-time drop cannot run unrelated instant zero, while an actually unknown
id keeps its pre-story behaviour.

## FR-2 — a complete return-value observer

`sim.StepTraced` returns one `ScriptTrace` for the same step `sim.Step` performs. `World` has no trace
field, callback or global sink. With a nil trace, the recorders return immediately; with a trace,
they collect these facts at the production sites:

- every check: index, opcode, owned register, dispatch, actual write, before/value, loss-counter
  before/after, or the exact silence reason;
- every trigger: inert, spent once-latch, failed, fired or held; latch before/after; each comparison
  actually evaluated; the first failing comparison;
- every fired instant slot: slot and compiled indices, opcode, optional group sub-command, support
  decision, and the canonical hash immediately before and after dispatch.

A write is recorded at `setRegister`'s call site. Equality of the old and new value therefore cannot
turn a real write into a silence. Checks distinguish the four legacy unresolved/support silences from
the build-time constant, VIP no-value arm, two dead arms and the health-selector silence. The legacy
`Firings`, `VIP` and `Silent` projections remain for existing consumers; the complete records are
additive.

A repeating trigger whose condition remains true is `ScriptTriggerHeld`: it fires its instants again
without claiming a second latch edge. A failed trigger records only the pairs evaluated before the
short circuit. An inert or spent trigger records no invented comparison.

`game.PlayWorld.StepTraced` is the production mission-stage entry for a caller that needs the returned
trace. It still performs the driver's ordinary unsupported-census and announcement sampling. The
private play step delegates to it, so the developer command and ordinary headless play cannot drift
onto different script loops.

## FR-3 — ordinary campaign drives

Each map is driven for 4,000 ticks through `game.PlayWorld.StepTraced`. The deterministic input
strategy issues no command on 63 ticks out of 64. On the remaining tick it chooses the first living,
on-map party actor and issues an ordinary attack against a hostile actor or a bounded adjacent move.
It never writes a register, predicate, latch, entity field or RNG seed directly.

The exact-node join observes 181 reachable instant nodes on each root, up from the pre-story
unattended 147. It reaches 18 of the 25 runtime instant families rather than the unattended 17; it
also observes every runtime check node and every accepted trigger's decision. Natural status belongs
to the exact authored node. A sibling node firing never promotes one that did not.

Natural coverage is descriptive. Seven runtime instant families and three reachable group
sub-commands still have no natural node in this drive, and mutually exclusive mission branches are
not rewritten as though one playthrough took them all.

## FR-4 — the controlled production seam

Every reachable authored node behind the 50 runtime operation rows receives a controlled execution
and observation row, including nodes the ordinary drive reaches: 545 EN / 543 RU check nodes, 614
instant nodes and 89 opcode-6 sub-command nodes per root. The extra witness makes both the operation
result and its full parameter/reference population independent of what happened to occur in 4,000
ticks.

`sim.NewControlledScriptWorld` copies a normally constructed mission through its canonical byte
form, replaces only the program and program-owned execution state, presets constants, and returns an
ordinary `World`. The source mission is unchanged. The harness then:

1. copies the exact compiled check or instant record from that mission;
2. passes it through `sim.NewScript`;
3. for an instant, places it behind one always-true one-shot trigger;
4. advances through `sim.StepTraced` and requires the returned identity to match the exact slot,
   compiled index, opcode and, where present, sub-command;
5. evaluates a separate postcondition over public state and return-copy observers.

The harness never calls `runCheck`, `runInstant`, `cmdGroupOrder` or an effect helper. A seed is
itself applied by a preceding controlled production instant: add item before take, remove from map
before return or swap, seed a carried item before give-all/drop-all, remove the group before group
return, and cast a real effect before a lifetime setter. Whenever preparation changes the world, the
same exact node also runs against the unmodified mission and must satisfy its ordinary/miss oracle.
Every seed is printed in the CSV.

`sim.ObserveScriptGroups` returns copies of the runtime group records needed by group-order oracles.
Changing a returned copy changes neither the world bytes nor its hash.

## FR-5 — claim-backed effect oracles

The dispatch trace says that an arm ran and whether canonical state changed. It does not say the
change was correct. `cmd/scriptcoverage` joins each runtime row to an oracle derived from the active
claim set:

| State boundary | Covered operation rows | Representative authority |
|---|---|---|
| register write, same-value write or deliberate silence | all 17 runtime checks | `TRIG-COND-003`, `TRIG-DIST-014`, `TRIG-PARAM-030`, dedicated check claims |
| comparison, latch and emitted event | triggers and instant 2 | `TRIG-MSG-023` |
| outcome counters and mission variables | instants 3, 4, 5, 8 | `TRIG-END-009`, `TRIG-PARAM-030` |
| relation, formation and purse cells | instants 7, 10, 23 | `AI-FORM-037`, `TRIG-DIPLO-019`, `TRIG-MONEY-028` |
| carried stacks, sacks and ownership | instants 12, 13, 19, 20, 22, 28 | `TRIG-ADDITEM-027`, `TRIG-TAKEITEM-038`, `TRIG-DROPALL-024`, `TRIG-GIVEALL-025` |
| map presence and bounded return | instants 16, 17, 18, 32, 33 | `TRIG-OFFMAP-041`, `TRIG-RETURN-042`, `TRIG-MAPGROUP-043` |
| pending casts, tails and effect lifetimes | instants 21, 24, 25, 29, 30 | `TRIG-CASTACTOR-044`, `TRIG-CELLTAIL-035`, `TRIG-EFFECTTIME-034`, `TRIG-CELLEFFECT-045` |
| entity or structure field | instant 34 and check 21 | `TRIG-PROPERTY-036`, `SAV-BLDG-037` |
| group order, commanded cell or per-member order | instant 6 and eight reachable sub-commands | `TRIG-GROUP-005`, `AI-GROUPCMD-020`, `TRIG-GRPARM-047`, `TRIG-GRPLIMIT-048` |

Message instant 2 is a deliberate canonical no-op, so its complete oracle also samples the real
`game.Announcer` with the exact raise join produced by the builder and requires the authored event.
Effect setters 29 and 30 cover an absent match and one matching effect. Return, swap, give-all,
drop-all and group-return seeds make their positive postconditions non-vacuous. These seeds do not
prove duplicate all-match behaviour, and the presence/group oracles omit further untouched-state and
reset clauses; `DIV-387` carries the complete weak-oracle population.

The release gate runs every exact node through its current oracle, then aggregates those results into
the 50 runtime operation rows. This prevents an unvisited reference or parameter shape from standing
in for its siblings; it does not make a coarse oracle discriminate every effect. Under those current
oracles, instant 30 alone is `DIVERGES`, `DIV-385`, and neither exact node is counted as an effect
PASS. `DIV-387` prevents the other PASS labels from being read as stronger semantic proof than the
instrument supplies.

## FR-6 — synthetic cases stay outside shipped closure

The synthetic section has nine rows per root:

| Population | Rows | Result |
|---|---:|---|
| authored-unreachable instant 20 | 3 | PASS through its carried-container/sack oracle |
| authored-unreachable group sub-command 1 | 1 | PASS through its group-order oracle |
| campaign-absent group sub-command 17 | 1 | `DIVERGES`, `DIV-386` |
| campaign-absent catalogue literal 18 | 1 | PASS as the original's deliberate inert no-op |
| group-10 target veto | 1 | PASS with a real air-domain target and vetoed real members |
| group-11 byte-zero range coercion | 1 | PASS with authored value 256, whose low byte is zero and becomes three |
| group-15 byte-zero range coercion | 1 | PASS on the same boundary |

The first four are complete corpus populations, not representative examples: three exact opcode-20
nodes, one exact group-1 node, and one absence row for each catalogue literal. The last three are
claim-proved unreachable semantic branches built from exact compiled group records with only the
unreachable parameter replaced. None contributes to the 52-row denominator.

Literal 17 and literal 18 are deliberately separate. `TRIG-GROUP-005` gives 17 a real runtime case
and 18 none, while `AI-GROUPCMD-020` and `AI-ROAM-025` give 17 its group state and behaviour. The
current dispatcher treats both as unsupported no-ops. That is a fidelity defect only for 17.

## FR-7 — durable refusal and disposition

`TestReleaseCampaignScriptExecutionClosure` skips without `AGAINROM_ASSETS` and names the missing
variable. `pipeline/check-release-tests.sh` discovers it in the measured gated population, then runs
it with every other install-gated test on EN and RU. It refuses:

- any map or compiled-count drift, or a missing expected 52-row key; it does not yet reject an extra
  reachable unknown key (`DIV-387`);
- a runtime row without controlled production dispatch and a claim-backed effect PASS or typed
  divergence;
- a build-time row without its builder postcondition;
- an accepted check or trigger not observed in an ordinary pass;
- natural exact-instant coverage at or below the standing 147-node baseline;
- a synthetic row missing from its named population or carrying the wrong disposition.

The command emits the same matrix outside the test runner as CSV, or the root summary with
`-summary`. Lawful assets and derived map bytes are never written to the repository.

## Design decisions

### DD-1 — observation is returned, never stored

Trace data, group snapshots and coverage bookkeeping are caller-owned return values. No new state is
serialized or hashed. Traced and untraced copies are compared after every step by both bytes and
digest.

### DD-2 — controlled means real mission plus exact record

The controlled seam starts from a production mission, crosses `NewScript` and `StepTraced`, and
prints every replacement or precondition. A helper call or a hand-built miniature world cannot close
a shipped row.

### DD-3 — dispatch and semantics are different evidence

The trace owns identity and dispatch outcome. The command owns the claim citation and independent
state postcondition. Before/after hash equality is sufficient only where the active claim specifies
a no-op. `DIV-387` records the postconditions that are presently too coarse or incompletely mutated.

### DD-4 — roots never merge

EN and RU are loaded and reported separately. Their operation sets agree; their check and trigger
populations do not. Neither root supplies a missing row for the other.

### DD-5 — synthetic evidence never promotes a shipped row

Presence, acceptance, reachability, natural execution, controlled execution and semantic effect are
separate CSV fields. A synthetic seed can demonstrate an arm but cannot change reachability.

### DD-6 — builder-owned forms stop at the builder

The constant and drop rows are not forced through runtime dispatch. Their postconditions are the
owned register preset and the absence of a runtime instant/trigger slot respectively.

### DD-7 — no new byte-form state

`formatVersion` is unchanged. The controlled seam reuses the existing canonical form; the observer
adds no section, flag or field.

## Fidelity findings, not 1041 mechanics

`DIV-385` is a shipped, player/state-visible pre-existing defect: both exact instant-30 nodes in
`90.alm` write `Entity.SpellFX`/`SpellFXSpell`, a presentation mark, while the canonical attached
effect's hashed `Remaining` value merely takes its ordinary tick from 1461 to 1460. The authored
durations are 60000 and 1. Story 1041 records both rows and leaves the Combat & Magic repair to a
bounded hotfix.

`DIV-386` is a campaign-absent pre-existing defect: group sub-command 17 is reported unsupported and
does not install order `0x11` or Roam behaviour. It is synthetic-only and does not reduce the shipped
52-row result. Literal 18 is a separate PASS.

`DIV-387` is one witness-fidelity class, not a mechanics finding. Independent pass-1 inspection found
the mechanics behind every listed bucket correct and produced no P finding, but the current command
cannot discriminate all of the claim clauses it reports as PASS. The ledger row owns the full
population and the one remediation boundary.

## Deliberately unchanged

No effect, inventory, order, combat, persistence or UI rule is repaired because its mismatch became
visible here. `pipeline/check-milestone.sh` remains a support census and is not relabelled execution
closure. No matrix output, lawful asset or generated campaign map is checked in.

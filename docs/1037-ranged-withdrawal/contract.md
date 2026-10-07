# Story `1037` — ranged-monster withdrawal

**Contract, lane, 2026-08-24.** Base
`27e994c5c9f4b6bb5953a8ee62aeca21299394b0`, research pin
`d7ee0c62cfa4a16083d356f24b1870035e1f0209`. Branch
`story/1037-ranged-withdrawal`, worktree `wt-story-1037`.

`DIV-345` through `DIV-352` are reserved to this story. The lane stops and asks
the seat if the range is spent. It does not take another id from an unmerged
worktree.

Simulation form version 60 is used because both thresholds decide the next
hashed tick and must survive save/load. Form 59 remains the immediate readable
predecessor; the story does not choose another version.

## Result

A ranged monster whose authored health threshold is reached disengages under
the original group-AI tail. It chooses the same hostile population, targets
the same cell three steps away from the mean living position, and hands that
cell to the existing movement and route machinery. The rule runs after the
group's ordinary order dispatch. Patrol, guard, attack and commanded groups
therefore make their ordinary decision first, and a successful withdrawal may
replace that decision.

Classes without the decoded thresholds do not gain them. Humans and
mercenaries do not withdraw. Save and load preserve the next withdrawal
decision, including any runtime threshold overwrite this build can identify
without promoting a Medium command name.

The pointable result is a real withdrawing placement in a campaign mission on
each lawful EN and RU install. The witness prints the actor, threshold, hostile
population, pre-tail decision and replacement destination produced through the
campaign loader and ordinary simulation tick.

## Research authority

The pinned research submodule is the only authority for ROM1 behaviour. The
following active rows were read in full through `research/tools/claim`:

- `AI-WITHDRAW-026` (High) gives the per-member post-dispatch tail, the full-tick
  placement and the ordered health gates.
- `AI-WITHDRAW-027` (High) binds `Withdraw` and `Wimpy` to streamed slots 34 and
  35, establishes absolute-health comparisons, and gives the class and shipped
  placement census.
- `AI-WITHDRAW-028` (High) gives the hostile mean, living filter, three-cell
  geometry, zero-axis substitutions, edge clamps and route boundary.
- `AI-CLASS-029` (High) gives the spawn classifier and proves that construction
  reaches it under the base vtable, so it writes nothing and preserves the
  streamed thresholds.
- `AI-CLASS-030` is High for both threshold writers and their three modes, and
  Medium for the outer command name. The build does not promote that name.
- `SESS-PARAM-017` (High) identifies the outer boundary independently: opcode
  `0x46`, selector parameter 3, the mode at `cmd+0x0e`, player scope and the
  exact calls into `R0186` and `R0187`.
- `UNIT-STREAM-001` is High for the 38-slot stream and Medium for the tail-slot
  names. Together with `AI-WITHDRAW-027`, it establishes the slot 34/35 names
  and destinations at the required threshold.

The boundary is known through `SESS-PARAM-017`; this build has no generic
session-command producer through which to expose it. Story `1037` does not add
an isolated setter that bypasses opcode `0x46`'s player scope. `DIV-346` records
that absent producer.

## Measured starting state and population

At the base, `pkg/data/unitdef.go` consumes slots 34 and 35 without storing
them. `pkg/sim` contains no withdrawal decision. The existing movement,
engagement, group-order and route-substitution systems are consumers to extend,
not parallel systems to replace.

The research census contains 56 parameterised classes. Twelve carry positive
`Withdraw`: four tiers each of `Goblin_Sling`, `Orc_Bow` and `Bat_Sonic`. All
twelve are ranged. Twenty-four carry positive `Wimpy`, including melee and
Dragon rows. `AI-WITHDRAW-027` published a placement census before
`UNIT-GATE-033` corrected the EN Units-versus-Humans resolution. The as-built
witness therefore regenerates the complete population from both preserved
roots and reconciles both claims instead of pinning the stale EN count.

## Behaviour

### B1 — data reaches every producer

Slots 34 and 35 become named signed values on `data.UnitDef`. Sentinel cells
retain the constructor default. Every placement, clone, ghost, party or resume
path that can create a non-hero entity carries both values or proves that its
population cannot own them. The values do not exist only in the ALM loader
while another entity producer silently creates zeros.

### B2 — one post-dispatch tail per full tick

Every participating living member of every AI group is offered the withdrawal
tail once per full simulation tick, after that group's ordinary order dispatch,
whatever the current group order. Entity and group iteration remain canonical.
Dead and off-map records do not manufacture a decision.

The tail may replace the destination or attack decision written earlier in the
same tick. It does not create a second AI scheduler and does not run once per
render frame.

### B3 — the health gates retain their order

Current health and both authored thresholds use signed absolute-health
comparisons. The `Wimpy` arm runs first and uses its decoded hostile
collection. Only when that arm does not take the member does the `Withdraw` arm
use the literal radius-two collection. Equality, zero and sentinel thresholds
follow the decoded comparisons; the implementation does not substitute a
`threshold > 0` enable rule.

The classifier described by `AI-CLASS-029` is not recreated as a working
derived classifier. The original construction order makes it write nothing and
preserve the streamed values.

### B4 — flee geometry is exact

The chosen hostile set is filtered to living actors. With no living hostile,
the decoded ordinary-acquisition fallback runs. Otherwise the target is three
cells directly away from the arithmetic mean of the hostile fine positions.
The zero-axis substitution, larger-axis parametrisation, integer conversion and
map-edge clamps follow `AI-WITHDRAW-028`.

The picker does not read the block plane. It submits an ordinary move target to
the existing canonical route search. The route search's decoded substitute rule
handles an unreachable chosen cell.

### B5 — command and interaction ordering is preserved

Player commands, script group commands, attack pursuit, patrol, guard and
escort retain their existing dispatch. A successful later withdrawal write may
replace their decision. A failed gate does not clear or alter the earlier
decision. Death, Stone, invisibility, sight, command-group identity and group
membership remain owned by their existing systems.

Opcode `0x46` parameter 3 is the decoded runtime overwrite. Its mode at
`cmd+0x0e` selects uniform 0, 10 and 30 percent maximum-health thresholds and
its writers run in the decoded player scope. This build has no session-command
producer, so none of those writes is reachable; `DIV-346` records that known
production debt rather than calling the command unidentified.

### B6 — persistence and determinism

Every value needed for the next tick's withdrawal decision is canonical
simulation state. Hashing, binary encoding, validation and round-trip tests
cover it. If this changes the byte form, form 60 appends the state without
reusing or renumbering form 59. Every supported older form migrates
deterministically to the state its class and available producer data can
establish. Any irrecoverable loss uses the existing upgrade disclosure.

No clock, map iteration, client state or noncanonical cache enters the
decision.

The six headings are one vertical result. B1 without B2 through B5 would leave
parsed values unused. B2 through B5 without B6 would make a save or hash erase
the next decision. Splitting the shipped witness from this state would leave
the production path unmeasured. The slice therefore remains one story despite
combining hashed state with five domains.

## Domains

The story touches five domains from `docs/DOMAINS.md`:

- **Assets**: the two streamed unit-definition slots and both-root census;
- **Sim Core**: canonical entity state, tick ordering, movement target, route
  handoff and hashing;
- **AI & Orders**: group dispatch, health gates, hostile collection and flee
  decision;
- **Campaign & Scripts**: placement producers, supported group commands,
  mission loading and any identified threshold writer;
- **Persistence**: byte form, validation, migration and resumed-world
  equivalence.

The reviewer walks the interfaces between all five domains.

## Required witnesses

- An independent slot walk around slots 33 through 37, including sentinel and
  signed values.
- Exact `Wimpy`-first and radius-two `Withdraw` cases, boundary equality, zero
  thresholds, sentinel thresholds and a failed gate that retains the earlier
  decision.
- Multiple living hostiles whose mean distinguishes sum/count from nearest
  hostile behaviour, dead hostiles excluded, both zero-axis cases, all four
  edge clamps and an unreachable target that exercises the existing route
  substitute.
- A group order whose ordinary decision is replaced after dispatch, and one
  whose failed tail remains byte-identical.
- Player-commanded success and failure; group orders 0, Guard, Swarm, Stand
  Ground, Move and Swarm 2; actor Patrol, Defend and Follow; dead, off-map,
  owner-zero and Stone exclusions; sight, invisibility and off-map hostiles;
  and form-60 equality of the next decision.
- Hash sensitivity and byte-form/migration proof for every added canonical
  field.
- A campaign mission witness on each preserved root, plus the complete
  56-class and all-map placement census. The mission witness prints the
  ordinary decision immediately before the tail. The census resolves each
  positive row's EquipItem and exact attack classification, prints its
  denominators and positive counts, and bounds maximum placed health overall
  and among threshold-active actors.
- A mutation at the post-dispatch production call site and a mutation of the
  three-cell geometry. The committed witnesses kill both mutations.

## Twelve-aspect closure

| Aspect | Required result |
|---|---|
| Data | Slots 34 and 35 plus the complete EN/RU class and placement census. |
| Runtime state | Exact thresholds and any runtime overwrite live on the authoritative actor/order state. |
| Simulation | Tick ordering, health gates, hostile selection, flee geometry and route handoff. |
| Player input | Player orders retain their dispatch and are superseded only by a successful later tail. |
| AI | Every group order reaches one common tail; no duplicate scheduler exists. |
| UI/HUD | N/A unless the existing presentation of the resulting move changes. |
| Triggers/scripts | The decoded opcode `0x46` boundary is stated exactly; its absent session-command producer is `DIV-346`. |
| Inventory/equipment | N/A, with the existing resolved EquipItem attack classification measured rather than changed. |
| Persistence/save-load | Hash, form, validation, migration and next-tick equivalence. |
| Campaign/session | Mission transition and resumed-world entities retain the rule. |
| Shipped content | Both roots cover every class and placement, with a real mission outcome. |
| Interactions | Group commands, attack, death/off-map, sight/invisibility and blocked-route substitution. |

A known in-scope GAP fails the story.

## Expected divergence rows

`DIV-345` records the base's parsed-but-unused withdrawal family and closes only
when the decoded data reaches the production AI decision, form and both-root
witness. `DIV-346` records the absent implementation producer for the known
`SESS-PARAM-017` session command. A form-59 migration loss, the decoded
dispatcher gates and the signed-WORD versus canonical-int32 health boundary use
separate reserved rows when this build cannot reconstruct or represent their
source state. Additional rows are created only for measured mismatches.

## Exclusions

- Building the generic session-command producer needed by opcode `0x46`.
- Recreating the spawn-time classifier as a working derived classifier.
- Adding a second route search, hostile relation model or AI scheduler.
- Changing ranged weapon or equipment classification.
- Changing screen drawing or adding a new player control.
- Widening a save form below the existing readable floor.

## Review ceiling and stopping condition

The adversarial ceiling is exactly **three passes**. Hashed simulation state and
the five touched domains do not raise it.

The initial review surface is the complete data population, every entity
producer, tick/order convergence, both health gates, hostile population,
geometry and route boundary, command interactions, form/hash/migration and both
shipped roots. The chain stops at the first pass with no P finding after this
remaining-surface list is empty. Only P returns the story. W becomes a typed
witness row and D is corrected in place without another pass. A repeated P
class is closed by enumerating its complete population. Pass three cannot be
extended; any remaining P surface becomes a separate defect story.

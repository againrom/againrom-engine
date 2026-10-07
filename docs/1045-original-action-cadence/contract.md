# Contract — story 1045 original action cadence

This is the canonical Stage 1–3 contract. Its commit advances the research pin from
`49b154de5a0d81cde33313cd03c86ee57211bc25` to
`23daf74f6ee83e5fee5474981faf7c276680cc9f`. No production cadence or canonical-state code may
start until story `1044` completes review, lands, and this branch merges that exact master tip.
Story `1044` currently reports that simulation form 61 remains current. That report does not
allocate a byte-form version to this story.

## Outcome

Retained physical attacks and spell orders use one canonical per-actor action lifecycle. Each
application occurs after the decoded charge, recovery, random, equipment and scheduler terms.
Insufficient mana retains the cast order and its progress. Mana restored later admits the same
order at its next scheduled attempt. Death and invalid targets cancel or refuse only at the
established lifecycle boundaries.

The player-visible discriminator is a repeated cast after mana exhaustion. A completed caster with
insufficient mana attempts admission on three actor ticks, skips one tick when the stale completion
state is consumed, and repeats. Restoring mana causes the retained cast to proceed without another
player command. Physical attacks, ordinary book casts, weapon-diverted mage casts and fighter
weapon riders keep their separate cadence identities.

## Research baseline

The pinned research contains the action-cadence claims published from `EXP-0234`. This contract
consumes the claims, not the experiment report.

- `HERO-CADENCE-112` is High for the retained physical interval and its two scheduler-boundary
  ticks.
- `HERO-CADENCE-113` is High for the Humanoid gate and penalty formula. Its shipped-population and
  effect-reachability clauses are Medium and do not define canonical arithmetic here.
- `HERO-CADENCE-114` is High for the common armed executor and the mage-diversion versus fighter-
  rider distinction. Its broader upstream ownership-negative clause is Medium; this contract does
  not use it to constrain every order producer.
- `HERO-CADENCE-115` is High for actor-death cancellation and the named physical target checks. It
  leaves external order replacement and spell-specific target loss Unknown.
- `MAGIC-CADENCE-126` is High for the Complication gate, pointer-clear ordering and cadence
  formulas. Its shipped Complication distribution is Medium and is witness scope only.
- `MAGIC-CADENCE-127` is High for insufficient-mana progress, completion and retry timing. It
  leaves external replacement of the retained order Unknown.
- The active, amended `HERO-CADENCE-023` supplies the three-phase countdown, charge/relax sources
  and physical ranged-extra term. Its retracted complete-period formula and worked intervals are
  not used.

All in-scope facts that enter hashed state use the High portions above. The two Unknowns remain
explicit bounds; they are not filled by inference.

## Domains and interfaces

The story touches six domains:

- **Assets** supplies the Humanoid classification, runtime weapon weight, charge/relax values and
  Spell Complication value as typed inputs.
- **Sim Core** owns the tick order, random draw, canonical lifecycle, command intake, hash and
  binary form.
- **Combat & Magic** owns physical application, book application, mage weapon diversion, fighter
  rider application and recovery.
- **AI & Orders** owns retained orders, progress, completion, re-arm and cancellation.
- **Party, Items & Heroes** owns equipped-weapon identity, runtime weapon weight, Reaction and the
  Humanoid predicate.
- **Persistence** owns exact save/load, hash participation and legacy migration.

The Client and Campaign & Scripts interfaces are checked because they submit commands and consume
events. Neither receives new cadence state or a new rendering rule.

## Behaviour

### Canonical inputs

1. Every actor carries an explicit canonical Humanoid classification. It is supplied by every actor
   producer, survives save/load, and participates in the world hash. It is not inferred from mana,
   ownership, type-id ranges or the presence of equipment during a tick.
2. Every Spell rule carries its decoded `Complication Level`. The typed Assets-to-Sim conversion,
   canonical spell table, save/load and hash preserve it. It is not looked up from install data
   during simulation.
3. Reaction and the equipped primary weapon's runtime weight use their existing canonical sources.
   No second weight formula or equipment resolver is introduced.

### One action lifecycle

4. A retained attack or cast uses one per-actor lifecycle containing the active action identity,
   phase, countdown, retained order progress and completion state. A tick performs at most one
   lifecycle transition and at most one application. A repeated explicit unit or cell cast cannot
   replace that record during recovery, either scheduler boundary or re-arm.
5. The lifecycle runs in canonical entity order and consumes deterministic random values. Refused
   admission for insufficient mana writes no recovery, draws no recovery jitter and does not erase
   the retained order.
6. The final two ticks in a repeated interval are order-machine boundaries. They are not added to a
   stored cooldown: one tick consumes completion after recovery, and the next re-arms the retained
   order and loads charge.

### Physical and fighter cadence

7. An uninterrupted retained physical order has this application-to-application interval:

   `charge + rangedExtra + relax + U[0,3] + humanoidPenalty + 2 actor ticks`.

8. `rangedExtra` is the decoded physical ranged-swing term. It applies only where that existing
   term applies. It is not added to ordinary book casts or mage weapon diversion.
9. An equipped Humanoid uses
   `clamp(IDIV(runtimeWeaponWeight + 5*(30 - Reaction), 12), 0, 12)`. Signed `IDIV` truncates toward
   zero. An unarmed Humanoid and every non-Humanoid use zero. The penalty is added to recovery before
   relax and `U[0,3]`.
10. A non-mage weapon Spell remains a rider inside the physical strike. It does not replace the
    physical action and does not start a second recovery interval.

### Book and mage-weapon cadence

11. The current owner-authored visible-swing floor remains. For an ordinary book cast,
    `bookCharge = max(charge, 8)`. This is an againrom choice, not a decoded registry value.
12. An uninterrupted retained ordinary book order has this application-to-application interval:

    `bookCharge + relax + U[0,3] + humanoidPenalty + ComplicationLevel + 2 actor ticks`.

13. A mage with a Spell-bearing weapon diverts from the physical strike into the cast route. The
    weapon-borne release is not subject to the eight-tick book floor or `rangedExtra`. Its pointers
    clear before the Complication gate, so its interval is:

    `charge + relax + U[0,3] + humanoidPenalty + 2 actor ticks`.

14. Player-armed and AI-armed actions converge on this lifecycle after their existing order
    selection. This does not claim that every upstream player and AI producer has identical
    selection rules.

### Retained insufficient-mana progress

15. Insufficient mana is a failed admission, not a failed-cast cooldown. It does not change phase,
    countdown or the stale completion state.
16. When completion is set by the preceding successful cast, the retained order attempts admission
    on three actor ticks. The progress-3 tick consumes completion, clears active progress and skips
    admission once. The next tick re-arms the same order and begins the pattern again.
17. When completion is not set, insufficient-mana admission is attempted every actor tick. It does
    not acquire the one-tick skip.
18. Mana restored later admits the retained order on its next allowed attempt. The caster keeps the
    same spell, target form and order position. No new click, AI selection or script command is
    required.
19. An offensive or player-armed restorative autocast is one-shot selection. Insufficient mana
    returns it to its producer without storing a target or a pending record. The unarmed idle-Heal
    affordability gate remains the producer-side exclusion.

### Cancellation and revalidation

20. Actor health at or below zero cancels an in-flight action before order or action dispatch. The
    actor applies nothing and starts no recovery afterward. All lifecycle residue is cleared or
    normalized to the canonical cancelled state, including `CastWait` written by a one-shot or a
    moving retained cast.
21. A physical attack rechecks actor/target presence, linkage and current reach at start and at
    application. Target health alone does not refuse the strike. The retained attack ends when its
    existing group/order teardown rule removes or replaces the target, not merely when target health
    becomes non-positive.
22. Existing book-cast admission and release revalidation remains unless a cited claim above changes
    it. `DIV-022` continues to own againrom's cross-producer priority, admission facing and current-
    perception checks. The timing change does not weaken those checks or add another busy predicate.
23. `DIV-028` continues to own movement during a pending book cast: the destination is admitted,
    movement waits for the cast to release, and neither the cast nor the move is discarded.
24. No broader rule is added for an external command replacing progress 1 or 2, and no one target-
    loss rule is generalized across all Spell arms. Those behaviours remain Unknown under
    `HERO-CADENCE-115` and `MAGIC-CADENCE-127`.

### Canonical form and migration

25. Every lifecycle value that can change a later application or retry is canonical. It participates
    in the hash and round-trips exactly through save/load at charge, application, recovery,
    completion-consumption, re-arm, insufficient-mana attempt and skip boundaries.
26. Form 62 refuses a non-retained pending book record and cast recovery on an actor that is not
    alive. The constructor normalizes the latter to zero. No new form field or version is required.
27. Fresh map, hero, original-save and raised-actor producers provide the canonical Humanoid input.
    Every Spell-table producer provides Complication through its existing typed seam. No tick reads
    a lawful install.
28. Legacy readable forms migrate deterministically. Existing attack, book-cast and recovery state
    is preserved where the old form carries it. Humanoid, Complication or retry facts absent from an
    old form are not guessed from unrelated bytes. The migration reports any resulting fidelity loss
    in the save-form loss notice.
29. If the exact reviewed story-1044 tip lands with form 61 still current, and no other landed branch
    has consumed the next byte-form version, story 1045 conditionally owns form 62. Otherwise the
    lane stops and asks the seat to reallocate. No code, test or document may present form 62 as
    current before that prerequisite is measured on landed master.

## Reconciliation and divergence handling

- The current eight-tick book floor remains by the owner's visible-swing directive. The landing must
  add or update one typed divergence row for that preserved difference unless an existing row is
  found to own it. No divergence id is allocated during this preparation stage.
- The current physical target-health precheck conflicts with `HERO-CADENCE-115`. The implementation
  removes that precheck from physical application and retains the claim's presence, linkage and
  reach checks.
- `DIV-022` remains the authority for againrom's owner-authored cross-producer priority and
  perception snapshot. `HERO-CADENCE-114` establishes convergence only after an order is armed.
- `DIV-028` remains OPEN because the arriving research does not answer external movement/order
  replacement during a cast.
- Any additional mismatch discovered after story `1044` lands requires a seat-allocated id. The
  lane does not select one from its worktree.

## Witnesses

- A synthetic interval matrix fixes charge, relax, ranged status, Humanoid status, equipment weight,
  Reaction, Complication and random output independently. It measures application ticks for physical,
  ordinary book, mage weapon-diversion and fighter-rider routes against literal expected intervals.
- A retry witness covers retained and one-shot producers. It proves three attempts followed by one skip when
  completion is set, attempts on every tick when it is clear, no recovery or jitter on refusal, and
  later mana admitting the retained order without a new command. It covers insufficient mana at
  admission and after charge. It also proves armed offensive and restorative one-shots return to
  current selection while idle Heal keeps its affordability gate.
- Cancellation witnesses kill the actor before, during and after application through command,
  later-ID combat, equipment and derived-health producers. They cover book, autocast, mage-diverted
  and fighter-rider self-kill, both recovery representations, target removal, unlink and reach loss.
  They observe the production application and event paths.
- Producer/reader population tests cover every actor constructor, every Spell-table converter, every
  player/AI/script action producer, both weapon-Spell routes, every lifecycle reader and every clear
  path. The population statement names what the instrument cannot see.
- Repeated explicit unit and cell commands cover all four jitter results, the Humanoid penalty,
  Complication and both scheduler boundaries without target replacement.
- Two worlds differing only in Humanoid classification, Complication, progress, completion or a
  lifecycle boundary hash differently. Dead cast-recovery residue normalizes to the same hash and
  form as zero and is refused when decoded. Each lawful boundary round-trips byte-identically and
  resumes with the same next application tick and random position.
- Legacy migration witnesses cover every readable version and every loss-notice class. The new form
  refuses malformed phase, countdown, progress, completion, target and ordering combinations.
- A headless real-campaign mission on each lawful root exercises one physical Humanoid, one ordinary
  book cast and one insufficient-mana retry through production map loading and command paths. No GUI
  or game window is launched.

## Twelve-aspect scope

| Aspect | Contract scope |
|---|---|
| Data | Humanoid classification, runtime weapon weight and Spell Complication inputs. |
| Runtime state | One per-actor lifecycle, retained progress and completion. |
| Simulation | Phase order, countdowns, formulas, jitter, retry, cancellation and revalidation. |
| Player input | One cast command remains effective through temporary insufficient mana. |
| AI | Armed AI orders use the common lifecycle; upstream selection remains source-specific. |
| UI/HUD | No new drawing. Existing cast/attack events remain the client boundary. |
| Triggers/scripts | Existing command producers are swept; no new script opcode or priority is added. |
| Inventory/equipment | Equipped primary weapon identity and runtime weight select penalty and rider/diversion routes. |
| Persistence/save-load | Hash, exact round-trip, malformed-state refusal and legacy migration apply. |
| Campaign/session | No new session state; real campaign missions provide the integration witnesses. |
| Shipped content | Both roots supply the Humanoid, weapon and Spell populations for the headless sweep. |
| Interactions with existing mechanics | Busy priority, movement during casts, perception, mana regeneration, death, target teardown and random order are preserved or explicitly reconciled. |

## Bounds

Do not change global tick speed, client animation cadence, spell effects, damage formulas, target
selection, mana regeneration, movement routing or campaign state. Do not invent external order-
replacement or spell-specific target-loss behaviour. Do not launch or control the game GUI. Use only
static and headless checks.

No new divergence id or serialized byte-form version is allocated before story `1044` lands. No
production file is changed during this preparation stage.

## Story cohesion

This story reaches hashed state and touches more than three domains. It is not split because failed
insufficient-mana retry consumes the same progress and completion state that successful application
writes. Splitting cadence, retry and persistence would leave an intermediate canonical lifecycle
whose failed path cannot reproduce the successful path's state, and would require a second byte-form
migration for one action.

## Review ceiling and stopping condition

At most three fresh adversarial passes. Hash and domain count do not raise the ceiling. Only P
returns the story. Stop at the first pass with P=0 and an empty remaining-surface list; apply W and D
without another pass. The first review brief enumerates every lifecycle producer, reader, writer,
clear path, save/load field, migration and event consumer.

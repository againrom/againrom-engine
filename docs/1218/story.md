# Current actions survive SAV

Ordinary SAVE retains unfinished book, scroll and script actions, typed targets,
accepted movement, clocks, queued menu choices and visible spell continuations.
Fresh LOAD resumes the next action; a second changed SAVE remains a mission SAV.
SAVE does not advance time, finish an action or spend a resource.

Base: `72db0bfb0a92473eb2545f5acb734ffd9dfe5a82`. Knowledge k68:
`a950c62defb1fac82b5acc7d9249d0230edd56eb`. This is the final M6 current-action
family under `pipeline/SAV-ENDGAME.md`. M7 owns whole-world assembly without a
source document. Group ordering and DIV-1368 remain unchanged.

## As built

The existing producer projects current actor/order targets, cast phases, timers,
completion latches, deadlines, motion and Book parameters into native fields.
Book membership and positional Spell references follow the current book; shared
Spell children are copied when their consumers require different values. This
repairs Fergard's real range loss in `122323`.

A bounded `/CurrentState/AgainromActions` registry value supplements fields whose
exact continuation has no established original encoding. It contains typed
records and explicit object bindings, never an AGS or serialized World. Current
native fields are still projected; loaded bytes are fallback evidence only.
Source-free component tests use different donor values and the same producers.

Book payment, retries, charge, release, recovery and completion boundaries remain
distinct. Scroll reservations keep complete current items and shared children
until release or cancellation/refund. Script entry remains queued. Actor zero,
self, removed actors and structure targets have distinct bindings. Terminal
actor continuation is retained separately from ticking actors.

Movement preserves admitted stride, fractions, turns and routes across later
speed changes, including unrated and off-map states. Unknown clocks preserve the
first-tick policy and the separate zero-regeneration case. Menu commands retain
order, retreat thresholds and auto-healing presence across opposite-profile LOAD.
Bolts, delayed explosions, Heal/Drain showers and cast runs preserve age and
expiry. Stable visual actor labels and allocation state preserve future seeds
and path tags across births, removals and repeated reminting; native initial
seed behavior and point-cast endpoints are unchanged.

## Proof

Seat `review/story1218-witness/README.md` and `manifest.json` identify commands,
seeds, direct current-state expectations, hashes and retained failures. These
are engine witnesses, not original runtime evidence or a release verdict.

- Three registered EN/RU tests cover ordinary inventory scroll reservation and
  release/refund, a real cell-entry script queue, and paid book windup. Each
  compares 65 uninterrupted samples with a separate process after the first
  SAV, then 61 samples after a changed second SAV at elapsed four. Actor,
  Spell and Order wire checks do not use carrier bindings as their oracle.
  EN: `action-wire-final-en.txt`; RU: action subtests in
  `action-focused-ru.txt`, plus corrected `options-final-ru.txt`.
- `current-book-cast-witness/candidate1218-second` passes frozen EN/RU current
  expectations: unchanged distance nine, spell7 spends15 mana once and releases
  nine cloud cells at elapsed nine. Base SAVE reduced Range9 to8 and refused
  that cast. A changed second SAV also preserves the cloud continuation.
- `current-options-queue-witness/runs/candidate1218-second` passes six real radio
  clicks, opposite preferences, exactly-once application and a second cold cut.
  The durable preferences test also checks the initial two-command ordinary
  SAV and64 later option states, alongside its legacy AGS regression.
- `current-motion-cut-witness/runs/working1218-first` passes both61-sample first
  and57-sample second continuations from untouched `122323`. Base lost target
  and stride, differing in47 of61 position samples. Components additionally
  cover changed speed, turns, unrated/off-map state and wrapping/unknown clocks.
- `current-native-visual-witness/working1218-second` passes unchanged frozen
  EN/RU Fire Ball and natural auto-Heal traces through both cold cuts and expiry,
  including actual draw positions/frames and future Heal seed1238302146.
  `working1218-first` retains the intermediate new-Heal seed failure caused by
  reminted actor IDs. Source-free tests cover future visual allocation too.
- Unchanged owner-M20 inputs20/21 pass EN/RU action-component recovery and40
  exact next ticks in `owner-pair/{en,ru}-bound.txt`. Their strict current SAV
  exports also succeed. This is component proof, not another fresh-process
  whole-world claim. Failed raw-clone controls lacked the mandatory loader
  arithmetic rule and never advanced; their receipts are preserved.
- Focused sim/SAV packages, game components, architecture, story and gated-test
  inventory pass. Three new release names are registered. Neutral shared
  helpers preserve legacy entry points; identifier debt falls5952 to5950 and
  comment bytes fall7719960 to7719416. Asset and preservation guards pass.

The observable result is ordinary SAVE/LOAD continuation, not a script-census
claim. No script-node implementation changed. The seat owns the sole review,
full final chain, corpus census and exact-main promotion; none is claimed here.

Landing checks preserve presentation-only residue without a simulation World.
Historical snapshot comparisons independently check native visual labels and
allocation defaults before excluding those additions; every historical field,
producer fixture and continuation digest remains fixed. Current descriptor
digests account for the five new visual fields. Group transaction checks now
prove supported authored orders and rollback after a malformed final patrol.
All seven repaired test families pass, and comment bytes fall to 7711496.
The sole adversarial pass accepted candidate `1da10cd`; its independent EN/RU
owner-save and changed-speed cold continuations remain in the seat report.

Release correction binds the action supplement before late graph projection.
SAV reindex and retirement relocate its archive references once, retaining
opaque action values and detached reservation values. New objects may carry
zero keys until final remint; nonzero collisions still fail. Five affected
item, mixed-spell and Book families pass on EN/RU, with independent zero-key,
permutation, retirement and malformed-input controls. The mover oracle now
reads the actor's changed target, and the original-mission dialog case proves
mission SAVE and two fresh-process continuations instead of requiring its
retired city fallback. Both route witnesses pass on EN/RU. Identifier debt
falls to 5948 and comment bytes to 7711226; final gates remain the seat's step.

## Authority and remaining debt

AI-ORDER-039 and MAGIC-221 identify cast order target, Spell and range words;
MAGIC-CADENCE-126/127 distinguish payment and recovery. The partial idle-turn
retraction is not cast authority. SAV-UNITPROG-156 establishes raw serialization;
SAV-ACTORINPUT-547 limits pointer repair to its stated stage. ITEM-CASTSTATE-056
retains its Spell/borrowed-Item distinction and retraction. SAV-TOKENPOS-074 and
SAV-REGENORDER-531 bound position and deadline fields.

REG-099/102 establish the registry mechanism, not original acceptance of a new
leaf (SAV-914/916). DIV-1369 records chosen action encoding, the unproved original
reserved-scroll graph and transient script persistence (TRIG-CAST-033).
DIV-1370 records visual/seed encoding debt; ANIM-CLOCK-001 does not establish
original drawable serialization. Delayed audio cues are not persisted by this
visual component. Original acceptance and full native/original lifecycle
identity remain M10 work, never an inference from successful engine reloads.

Existing family rows were amended; only reserved DIV-1369/1370 were added.
Both seat allocation sweeps passed with next DIV1371 unchanged. M7 source-free
whole-world assembly, M8 city state and later acceptance remain ordered work.
No knowledge pin or original evidence changed.

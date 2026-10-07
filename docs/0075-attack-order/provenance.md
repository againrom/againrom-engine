# Provenance — the player can order an attack

Every claim id below is a row of the `research/` submodule at its pin. Nothing is cited from an
experiment folder.

## What the contract rests on

| Spec anchor | Claim | Confidence | What it carries |
|---|---|---|---|
| FR-1 | `AI-CLICK-051`, `AI-PANEL-053` | High | Ordering an attack takes **two** clicks: a panel arms mode 1 and only a later click aims it. Guard and aggressive issue on the spot, attack does not. |
| FR-1 (a gate exists) | `AI-PANEL-053` | High | `view+0x99c = mode` is reached only when the gate holds; otherwise the routine returns having armed nothing. *(That row's own capability clause is struck through in it.)* |
| FR-1 (no class test) | `AI-PANEL-060` | High | The routine the struck clause pointed at **never looks at the selection**: twenty instructions over one view flags word, returning `0xef` — every mode but cast. **Attack is armable for any owned selection whatever classes it holds.** |
| FR-1 (the gate IS ownership) | `AI-PANEL-061` | High for the bit's set site and its guard | `view+0x144` bit `0x4` is set when the **primary** selected object's `CPlayer` (`obj+0x14`) differs from the local participant's record (`view+0x9b4`), and only then; the panel is disabled outright for a selection the local participant does not own. |
| FR-2 | `AI-CLICK-050` | High | The attack cursor is the only one producing **two different orders**: opcode `0x19` carrying the target when the hover state holds, else `0x16`, a plain move to the cell. |
| FR-2 (one-shot) | `AI-PANEL-053` | High | The armed mode is one-shot — `R0211` clears `view+0x99c` after every consuming click. |
| FR-3 | `AI-CLICK-050`, `UNIT-HOVER-020` | High | The click-time test is `R0214(view+0x98c, L00620)` — `CObject::IsKindOf`, a runtime-class test. **No diplomacy runs at click time.** |
| FR-3 (no owner test) | `AI-CURSOR-052` | Medium | The hostility bit exists but is decided at HOVER, off a view-side 32-entry row, and its only effect there is which cursor is shown. |
| FR-3 (the primary) | `AI-PANEL-061`, `AI-SELECT-065` | High / Medium | The comparison is made on ONE object, `view+0x138`, not over the set; and the set itself is a `CMapWordToPtr` keyed by unit id rather than an array. |
| FR-4 | `AI-CLICK-051` | High | What the order carries: opcode, the issuing player index, **the target actor id as a `u16`**, then every selected unit by id. The target is addressed by an **id**, not a pointer. |
| FR-6 | `AI-CMD-054` | High | `0x19` sets, per member, `actor+0x50 = 3` with `ord+0x0c = target` and `ord+0x14 = actor+0x12c`. |
| FR-6 (the stop distance) | `AI-GUARD-007` | High | Arm 3 is `R0009`, which sets `order+8 = 5`, `order+0xc = target`, `order+0x14 = actor+0x12c` — *"a target with a stop distance, which is the only construct in this area that survives being farther away than reach."* |
| FR-6 (reach is the stop) | `UNIT-COMBAT-006` | High | `actor+0x12c` is reach: constructor default 1, moved only by an equipped weapon's `@.range - 1`. This tree equips nothing, so the stop distance is 1. |
| FR-6 (the metric) | `TRIG-DIST-014` | High | Distance in this engine is **Chebyshev** — `R0162` returns `max(|dx|,|dy|)` with no multiply and no `FSQRT`. |
| — (not built) | `AI-RETAL-056` | High | Being struck issues no order and produces no target. Retaliation is out of this story because the game does not have it. |
| — (not built) | `AI-ARBITER-057` | High | One arbiter, not two; ownership is a local clause inside shared routines. Nothing here forks a behaviour tree by owner. |

## What is OURS by choice, and where the seam is

- **The arming input is a key, not a panel.** `AI-PANEL-053` decodes the panel's tables and its
  arming gate; it decodes no geometry, no art and no key. This tree has no command panel at all.
  **AUTHORED**, on the map screen's existing letter register (an action takes a letter, a diagnostic
  takes a function key — 0058 DD-11). The seam is one input flag and one `Viewer` method.
- **The consuming press is the SECONDARY button.** In the original the armed mode changes what the
  primary click does, because the primary click is that engine's act button. In this tree the
  primary button selects and the secondary orders (0028 FR-3). Taking the secondary press keeps
  *which button acts* where this tree already put it and reproduces the decoded property that
  matters — one press, two possible orders, chosen by what is under the cursor. **DISCLOSED
  DIVERGENCE.**
- **The arming gate compares ownership, and ONE OPERAND OF THAT COMPARISON IS NOT ESTABLISHED IN
  THIS BUILD.** The gate's content is a seat correction (`analysis.md` §Seat correction): the
  routine `AI-PANEL-053` pointed at tests no unit class and compares the primary selection's player
  against the local one. The *structure* is reproduced — an ownership comparison made in the view
  tier, over the entity's own owner word (`UNIT-OWNER-009`, carried into this tree by 0071).
  What is **not** established is our side of it: `AI-PANEL-061`'s local operand is a view-side
  participant record, and 0071 states at the field itself that which roster slot a human participant
  holds is made by the session-join path and not by the map. This tree has no session join. So the far side pushes **no local participant**, the comparison has
  nothing to compare against, and the control is open for every selection.
  **AUTHORED, with the seam named:** the local participant is one value pushed across the seam;
  zero is "none established" — which is the same zero `UNIT-OWNER-009` and 0071 already give a unit
  owned by nobody, and roster slots are 1-based, so no slot is shadowed. A story that establishes a
  participant pushes a nonzero value and the comparison becomes live with no structural change.
  **What is deliberately NOT built is a unit-class test**, because the retracted clause is the one
  that would have produced one, and a control greyed out by class reproduces a rule the game does
  not have.
- **The primary selection is the lowest present id.** The correction names *the primary selection's*
  player; this tree's selection has no primary. **AUTHORED** as the lowest id the snapshot still
  holds, the tie rule the tap and the emission order already use.
- **The three cursor gate globals are not modelled.** They are now closed — `AI-KEYMOD-059` reads
  them as modifier-key latches — and this story still models none of them: the arm is a flag with no
  cursor and no modifier behind it, which asserts nothing either way.
- **No cursor art changes.** `AI-CURSOR-052` binds the attack cursor to
  `graphics\cursors\attack\sprites.16a` at High. Drawing it is a different tier's story; the armed
  state is stated on the debug readout instead. **AUTHORED**, and it asserts nothing about the
  original's cursor.

## What this supersedes

`0064 spec FR-2` — *"an attacker holds a victim or a destination and never both"* — is superseded in
one direction by FR-6. It was written when nothing could issue an attack order, so no order had ever
had to survive being out of reach; `AI-CMD-054` and `AI-GUARD-007` say the original's block carries
both. The other half of that clause — a move order ends a fight whole — stands unchanged and is
still tested.

## The fan-out ceiling — examined, and diverged from

The engine's command record stops appending selected ids at **253** and the 254th is a **silent
no-op** (`AI-FANOUT-064`, High); and the container behind the walk is a `CMapWordToPtr` rather than
an array (`AI-SELECT-065`), so an oversized selection is ordered as an **unpredictable subset**
rather than as its lowest ids.

This build does not impose it, and the reason is not that it was overlooked (FR-9, DD-14). The
ceiling belongs to the shipped command record's **shared** id append — the move builder's as much as
the attack builder's — so imposing it inside this story alone would put half a rule in one story and
leave the move press, whose contract already says *every* present member is ordered (0030 FR-4), on
the other half. What this build does instead is deterministic and total where the engine's is
neither, and it is stated rather than assumed.

## What is NOT claimed

That the surface this story builds is the only producer of an attack order. `AI-SURFACE-063` counts
exactly **three** input surfaces reaching the order-builder family, and `AI-MINIMAP-062` reads the
second's gates as not the map click's. Nothing here builds either of the others, and nothing here
says they are absent.

That the pursuit re-aims the way the original re-aims. The original re-runs its dynamic search on
every completed cell transit and its static one far less often; this tree has no period on its
staleness tests, so a moving victim buys a fresh far search per victim step. That is a **cost**
disclosed in `plan.md`, not a decoded behaviour.

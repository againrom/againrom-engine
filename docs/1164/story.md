# Story1164 — independent areas and Poison continuation

Fire and Poison keep independent area clocks when another cast replaces their
painted cells. Poison applies the established signed-resistance damage and actor
phase. Ordinary native SAVE and fresh LOAD preserve these changes. The branch
reconciles accepted engine base `e021d390bb78bb784985046959d7c679e632334c` and adopts
public knowledge k12 `43397f0f0e8982cd9170c3f3ceebb7263d267f68`.

## Behavior and authority

| Surface | Current behavior and authority |
|---|---|
| Clock and order | First paint does not pulse. Each positive cloud counter decrements, pulses on the new multiple of16 including zero, then cleans up on the following call. Original retained roots precede fresh casts; fresh objects retain creation order across anchors. `MAGIC-CLOUDCLOCK-154`, `MAGIC-CLOUDOWNER-155`, `MAGIC-POISONPHASE-159` |
| Ownership and scan | Six spell layers hold only their newest owner. Overwritten objects keep their clocks, even with zero owned cells; cleanup never reveals an old owner. Each pulse scans the full x-then-y square and gates on any current same-spell layer, using its own payload. Repeated occupied footprint cells are visited independently. `MAGIC-CLOUDOWNER-155`, `MAGIC-CLOUDVISIT-156` |
| Conflict and retiming | Fire/Poison conflict clears cell ownership without removing old objects or actor attachments. Instant29 retimes the current owner at the addressed covered cell. The six-layer width no longer caps typed area objects; legacy mode0 keeps its former slot behavior. `MAGIC-MAPLAYER-040`, `MAGIC-AREACELL-039`, `MAGIC-FIREPOISON-160` |
| Poison input and HP | Only the installed Token8 health/continuous row shifts parsed duration by4, giving128 ticks for the shipped numeric8. Magnitude alone scales. Token8 uses signed water Protection[1]: nominal damage is negative magnitude; nonzero protection truncates `(damage*(100-p)+50)/100` toward zero. Computed zero skips HP and source award; HP has no potion derive or MaxHP clamp. `MAGIC-POISONINPUT-157`, `ANIM-074` |
| Attachment phase | Refresh changes Remaining only, retaining magnitude, mode and source. Areas precede actor attachments. The old positive counter is tested modulo8 before decrement; old1 expires without a zero pulse, and counters above9600 freeze. Current retained actor action0x10 skips this phase. Negative source HP clears attribution, zero HP remains eligible, and awards use computed damage. `MAGIC-POISONREFRESH-158`, `MAGIC-POISONPHASE-159`, `MAGIC-AREASOURCE-161` |
| Current cells | Fresh and restored layers share presence and cleanup. Current52-byte Cell payloads, retained layers, count, planes and Blocks stay coherent. Empty cells restore baseline cost/static with bit4 retained and keep recomputed dynamic. Missing owners or recompute dependencies refuse before partial painting/retirement. `MAGIC-WALLBLOCK-045`, `MAGIC-CLOUDEND-163` |

Production changes are in `pkg/sim`, the narrow installed-rule producer in
`pkg/mapload/spell.go`, and the original Document availability notice in
`pkg/game/savworldeffects_project.go`. Rendering uses the existing explicit
`CellEffect.Cells` list; a live zero-owned area draws zero cells. Story1165 owns
renderer changes separately.

## Save disposition

Form92 changes ordering and ownership semantics without widening form91's byte
layout. It validates unique current cloud ownership across retained and native
containers and persists zero-owned clocks. Earlier forms migrate retained roots
before native objects, then use stored native order. Replacing an older current
pointer updates both52-byte cell copies, retained coverage, planes and Blocks;
it preserves the older clock. A later expiry cannot reveal the replaced owner.
Previously deleted clocks and true historical creation order cannot be
recovered; a conditional migration notice states that loss. Frozen predecessor
bytes and digests remain unchanged. Old attachment counters and serialized
spell payloads remain retained, including legacy Poison duration8 rules.
This affects subsequent NEW Poison casts in an existing pre92 mission AGS,
until a newly constructed mission supplies the installed duration128 rule.
The registered App controls observe first counters7 for old8 and10 for custom11.
A custom duration8 row can be byte-identical to the old installed producer;
no provenance marker identifies its intent. The migration therefore preserves
both rather than silently overwriting custom payloads. This known player-facing
producer debt is DIV-1131; existing phase/resistance fixes still apply.

The sole review returned two native saved-state defects. Signed Poison now
uses the existing native health-gain transition when HP crosses from nonpositive
to positive: clear decay/dwell and restore death-halved defence once. It does
not reconstruct prior actions, membership or loot. This is a bounded native
policy, not evidence of original Poison revival (DIV-1134). Form91 migration
also validates the complete old World/Document pair before refreshing only its
changed cells and planes on a detached Document. The old envelope remains an
old pair until mission restoration. Current form92 conflicts and corrupt old
cell owners are rejected rather than repaired.

Supported retained Light and bare Poison use exact saved operands and clocks.
Existing retained ring and projectile consumers remain intact.
Effect_DirectDamage still lacks an established EDD48 byte-to-damage bridge and
remains unarmed. AE48 is not guessed casting Power. PE44 meaning,
SpellTransport scheduling, unsupported direction/altitude arms and complete
original reference lifetime remain Unknown.

Fresh native areas and new attachments have no allocated original SAV graph
identities. Their World state saves and reloads, while current Document
projection reports `WorldEffects.Unavailable` or `ActorEffects.Unavailable`.
No original object identity is invented (DIV-1132). An independent HP oracle
detects deliberately stale Document Health; native admission does not reject
this general scalar mismatch (DIV-1133). World HP remains authoritative.

Knowledge adoption closes DIV-039's cadence question. DIV-437 separates known
mover construction/derive producers from incomplete coverage. DIV-956 names
the known Player Diary writer and notification route separately from unproved
whole teardown reachability and actor-owned purpose. Those unrelated producers
are not implemented here. DIV-528 belongs to story1165.

## Proof and limits

`verification.md` records focused controls, installed EN/RU App casts, current
World/Document comparisons, ordinary menu SAVE, private source removal and
fresh LOAD. Authentic game0125 is checked before any controlled setup. Its
untouched expiry proof is separate from the Light pulse proof, which relocates
one real actor into the unchanged source's cloud, and from the explicitly
constructed retained Poison payload.

This establishes supported native local continuation. It does not establish an
original world-SAV writer, original-runtime HP/RNG streams, complete
interleavings or generic original actor/target lifetime.
`MAGIC-AREAOVERLAP-162` retains those bounds. The seat owns one fresh review
and the final merge gates.

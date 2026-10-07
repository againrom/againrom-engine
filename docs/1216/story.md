# Released spells continue through ordinary SAV

Ordinary mission SAVE now writes released native spell deliveries and retained
SpellTransport/PointEffect graphs. Cold LOAD keeps the current countdown,
selected target, frozen payload and root order; delivery happens once, retired
roots disappear, and another ordinary SAVE succeeds. This is the released
delivery family in M6, governed by the Rules for every story in the seat's
`pipeline/SAV-ENDGAME.md`. Unfinished book, scroll and script cast actions remain
the separately ordered current-action family.

Base: `a126ccf1310af376a8d33b4e28d96e9ba13ca9f5`.
Knowledge advances from k65 to published k68,
`a950c62defb1fac82b5acc7d9249d0230edd56eb`.

## Current behavior

One causal order covers retained spell roots, native standing areas and native
deliveries. MAGIC-187 establishes the advance-before-Tick list walk; MAGIC-197
joins transport handoff to that registrar, and MAGIC-225 joins the native
admission call. Thus a sole/tail transport's child waits for the next pass;
an earlier transport can append a child that runs later in the same pass.
Signed decrement and expiry follow MAGIC-189 and SAV-CASTCONT-1006.

Stable graph node IDs preserve root aliases and shared children. Retirement
clears incoming links, updates area-root bindings, and removes unreachable
spell/payload records through the existing archive remappers. Shared payloads
stay while another object references them. Dormant prepainted cloud children
keep their cell owners through handoff or discarded-fallback cleanup.

An unexecuted native cloud writes AE48[0]=0. Its first Area execution paints
and sets that byte without decrementing or pulsing; subsequent calls decrement,
pulse at each new multiple of16 including0, then clean up on the next call
(MAGIC-CLOUDCLOCK-154). Handoff alone never paints. An initialized cloud with
empty current coverage does not repaint. Pre-form92 native cloud drivers were
already running; their existing area-ownership migration marks that phase
initialized while preserving current owners and clocks. Frozen bytes are intact.

SAV-1010 and MAGIC-TARGETID-182 bind PE44 to the target. Point application uses
the frozen ordinary operands or established EDD48 damage bytes19..21, without
reconstructing a cast or charging mana again. Dead or removed targets receive
no new application. SAV-1011 and SAV-1054 establish the absent serialized caster;
SAV LOAD therefore keeps the existing absent-caster convention. Registered SAV
projectile records retain current graphics coordinates, clocks, IDs and expiry.

Current deliveries create real SpellTransport, Point/Area and Effect records.
Point type/picture and flags follow MAGIC-225; transport common bytes follow
SAV-1069. SAV-1068 distinguishes default Point Token construction from the
Position-argument Transport path. Its corrected Position assignment is not
described as target registration. Unknown first-native-SAVE fields use the
explicit bounded policy in DIV-1185, not a research-backed constructor claim.
No opaque simulation blob enters SAV and no fallback format was added.

Optional native form97 retains graph identity and mixed ordering. Historical
forms95/96 retain deterministic decoding, and frozen AGS fixtures are unchanged.
Legacy inert graphs bind only on explicit current SAV projection. Exported gob
field names remain unchanged. The loader's source-arithmetic callback survives
the optional form, so restored human worlds continue advancing.

## Proof

Focused simulation controls cover signed counters0/1/2/4/7/10/32767/32768/32769/
65535, tail and non-tail boundaries, simultaneous and shared deliveries, ordinary
timed payloads, area handoff, prepainted cloud cleanup, missing/dead targets,
fatal damage without invented caster credit, exact-once retirement and native
byte/hash continuity. Synthetic archive controls exercise shared nodes and
equal-valued distinct payloads. Corrupt graph cycles fail atomically.

The installed EN and RU witnesses drive the ordinary SAVE path, load in a fresh
process, compare actual target HP and frozen fields against uninterrupted play,
take the next consequence, then SAVE/LOAD again and check no replay. They cover
all four current installed delivery2 rows; production dispatch remains data
driven. Current witness results are:

| Spell | Current counter | Impact pass | Target HP before -> after |
|---|---:|---:|---|
| 1 | 7 | 8 | 131 -> 123 |
| 2 | 4 | 5 | 131 -> 118 |
| 13 | 10 | 11 | 131 -> 121 |
| 14 | 10 | 10 | 40 -> 29, 40 -> 22, 40 -> 20 |

Sole deliveries also SAVE the handed-off child before its application. Mixed
controls combine a retained transport, native standing cloud and native delivery
on one living target. They compare four consecutive states plus a second cold
LOAD and three further ticks, including ordered root clocks, HP, death stage and
credit fields. Non-tail expiry reaches HP111 on the first pass; tail expiry
leaves HP131 and a Point child, then reaches HP106 on the second pass.

Original `game0018.sav` is separate: runtime157 already has HP-57. Its current
counter2 and projectile segment1 survive cold LOAD, both roots and graphics
retire, and the second SAV does not replay damage. It is not a positive oracle.

Review artifacts are outside Git under the seat's `review/story1216-witness/`,
with `en/` and `ru/` directories containing spell1/2/13/14, both mixed states and
original0018-current and cloud-before/handoff SAV/JSON pairs. Emission requires
an explicitly supplied absolute `AGAINROM_SPELL_WITNESS_DIR`; ordinary tests
write only temporary files.
These changed files and their HP/root sequences are the observable result; the
previous producer refused pending deliveries and the retained transport fixture.
No desktop GUI or original-game acceptance is claimed.

The sole adversarial return R1 exposed two cloud handoff cuts: ordinary SAVE
changed cell counts0/9 before tail expiry and9/0 after handoff. Both exact EN/RU
reproducers now read0/0 and9/9. The durable release regression admits spell7 with
only Delivery=2 and EffectSpeed=256 changed, then uses vanilla installed tables
for both fresh-process LOADs. It compares uninterrupted cells, first-paint byte
and clocks through402 ticks, including the next pulse, zero pulse and cleanup.
The second ordinary SAVE is at tick2; its raw World.Cells owners and AE48/AE4C
are also compared. Clock398 stays unchanged during first paint. Simulation
controls exercise counters0/1/17, positive damage, non-tail execution, prepainted
fallback cleanup and initialized empty clouds. The frozen form91 migration
control retains its running, partly overwritten wall instead of repainting it.

The focused sim/game and internal architecture, story and gated-test receipts
are `review/story1216-correction-focused-final.txt`. Eight selected EN/RU release
tests are recorded in `review/story1216-correction-release-{en,ru}.txt`; the exact
reviewer reproducer is in `review/story1216-correction-repro-{en,ru}.txt`.
The manifest includes the cloud continuation witness. The seat owns the final
full test/release/M2/census chain, exact-binary drive and publication after this
single correction pass. This lane does not claim those final gates,
milestone closure or a change to `builds/current/`.

## Remaining limits

DIV-1185 retains unused native Token/Position values, transport type, unused
EDD48 bytes and original-game acceptance. DIV-939 retains arbitrary original
alias/destruction semantics and unsupported raw payload components. DIV-938 and
DIV-1271 retain original target teardown and fine-position distance debt.
Unserialized caster attribution can differ from uninterrupted native play.
Frontend-only native spellBolt visuals have no current SAV projectile producer;
this continuation preserves registered SAV graphics records.

The longer mixed fixture reaches the existing object62 `unknown=8000` item-merge
refusal at its next SAVE. Its failing receipt remains
`review/story1216-mixed-en.txt`; the next object/Group story must rerun and remove
that refusal. The shorter mixed proof retains the same delivery boundary and
second SAVE without changing items or clearing Unknown flags. Source-free worlds
remain M7, and unfinished cast actions remain separate M6 work.

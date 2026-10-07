# Spell delivery survives a save

New games and original SAV imports use the installed delivery columns. Book
mana is paid at admission. Firebolt, Fireball, Lightning and Prismatic Spray
prepare effects that land only when their simulation counter expires. Rendering
has no authority over damage timing. Prismatic prepares its capped, primary-first
fan at admission; ordinary spells prepare after charging.

Authority: MAGIC-DELIVERY-170, MAGIC-CASTCLOCK-171, SAV-CASTCONT-1006 and
MAGIC-SPRAY-135/136/137 in knowledge k42. The counter narrows to u16, decrements
once and delivers on signed expiry. Lightning and Spray use ten ticks.

Form95 preserves paid charging state, prepared rule/power/IDs and the remaining
counter. Reopening an AGS neither charges twice nor restarts a flight. Original
SAV output refuses pending native children instead of omitting them. Older AGS
forms retain their existing embedded timing rules and unpaid charge semantics.

Focused proof covers admission and cancellation, observed/unobserved equality,
charge and flight SAVE/LOAD, signed counters, area aim after target movement,
caster loss, terminal target loss, spray cap/order and script source distance.
Malformed or noncanonical state is rejected before publishing the decoded World.
Historical literal byte/hash fixtures remain frozen behind independent form
adapters. TestReleaseSpellDelivery1183InstalledRulesAndNativeSave reads all28
installed rows and exercises real rules1/2/13 through commands, file SAVE, a new
reader and delivery; it also tests the original-output refusal and its retirement.
The seat owns final Go, paired EN/RU release and original-save acceptance gates,
relevant mission scenarios, and the exact-main binary witness.

Open debt: DIV-1271 keeps fine-position distance, original scheduling and target
lifetime; DIV-1272 keeps the secondary group-sight/rank bridge; DIV-1273 keeps
migration of future casts from pre95 AGS. DIV-1132 retains original PE44/EDD48
binding and full original transport adoption. No native original pixel/audio or
full original-session equivalence is claimed. Reserved DIV-1274 remains unused.

The owner requested immediate local integration and deferred push. One fresh
adversarial pass reviews the exact local candidate. Remote equality verification
remains due when publication resumes.

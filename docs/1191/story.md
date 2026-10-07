# Global autohealing

Game Options and Ctrl+U expose No, Standard and Often. AutoCasting persists the
choice for fresh missions. Live choices use the canonical command queue, survive
paused AGS SAVE/LOAD, and produce Player+58 and actor mana floors in active-map
original SAV output and settled-town SAV output. Imported and carried parties
retain their own policy and stored floors unless a profile explicitly selects
a different mode. New characters alone receive the Standard default.
Explicit selected-spell autocast remains independent.

## Authority and implementation

TOWN-AUTOHEAL-458 and SAV-726 establish mode0/1/2 to100/50/0 percent, direct
percentages3..100, invalid-value retention and the signed-word mana-floor fold.
A global heal requires a book, nonzero mana and current mana strictly above the
floor before spell6 lookup. Ownership handover applies the new player's policy.

Corrected SESS-PARAM-017 places original retreat at selector1 and healing at3.
Historical native retreat retains3; native healing uses4. No old queued retreat
command changes meaning. DIV-1302 records this transport difference.

The optional form96 suffix carries sorted bounded player policies in the world
hash. An absent policy keeps exact form95 bytes and the historical quarter-pool
rule. Original Player aliases remain uncovered and cannot change a shared
policy. Profile write failures and unsupported source derivations leave the
world and pending queue intact. Document F58 disagreement is refused before
adoption. Old test fixtures peel only the additive policy suffix; frozen
predecessor bytes and hashes are unchanged.

## Proof

Focused simulation checks cover all modes, signed floors, invalid retention,
strict equality, explicit autocast, actual healing, native successor hashes,
legacy absence and corrupt suffix refusal. Game checks cover profile failure,
ambiguous players and document disagreement. The game, UI and simulation package
checks pass. The release manifest adds one installed witness.

EN and RU installed tests exercise the radio, held-world queue, opposite-profile
cold AGS continuation for64 ticks, ownership joins, active-map original SAV
write/read and Ctrl+U. Both option panels were rendered and inspected. Exact
candidate runtime, sole adversarial review and final landing gates are recorded
by the seat after candidate publication.

## Open debt

DIV-331 retains local patient selection, combat eligibility and scheduler
behavior; the threshold claim does not prove those consumers. DIV-1303 records
the authored fresh Standard default, deterministic old-save policy, and custom
percentage UI. DIV-1304 is closed: city output writes the same current Player
percentage as its returned Human floors. Town preferences apply to the next
fresh mission. DIV-1305 remains unused and retired.

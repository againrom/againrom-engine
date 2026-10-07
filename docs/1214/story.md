# A new corpse survives SAV and continues decaying

## Result and boundary

SAV-ENDGAME M6's first family is one complete continuation: a newly dead actor
writes the ordinary mission SAV, a fresh process loads its body and current
tuple, play advances its decay, and the next SAVE writes that changed state.
A party member or hireling without an authored map unit has the same result.
The rules in the seat's `pipeline/SAV-ENDGAME.md`, Rules for every story on this
track, govern this lane: no new refusal or AGS fallback, current state first,
unknown original facts named as debt, and unchanged corpus successes.

## Authority

The engine uses knowledge k65. `HERO-DEATH-026` and `HERO-DECAY-069` distinguish
the dead-list phase and cadence from still-live dying actors. `SAV-DEADLOAD-126`
stores stage, HP and timer independently; `SAV-DEADLOAD-128` gives the terminal
boundary. `SAV-DEADLOAD-125` does not require an authored map-unit join.
`SAV-DEADLOAD-132` leaves runtime occupancy outside the load chain Unknown;
an absent live entity is not evidence that the dead-list clock stops.

## Implementation and proof

Held dead records advance on the same phase-12 dead-list cadence as mapped
corpses. Their source tuple stays immutable; their current tuple advances
monotonically and remains readable through the existing native binary form.
Imported terminal tuples stay literal, including HP -10007. A new transition
below -600 uses stage 5, HP -10001 and runtime ID 0. No living entity, occupancy,
loot, reward or RNG event is created by that continuation.

The late-corpse export refusal is removed. Virtual bodies resolve T0E through
the unit registry, with HERO-APPEAR-041/042's hero-axis and dying-body law for
hero wire types. Player This/Slot resolves their owner under SAV-OWNER-048.
Explicit AGS reload reconstructs presentation from the admitted document rather
than losing the body. The native save format gains no field or version.

The existing DIV-1187, DIV-1344, DIV-997 and DIV-1353 rows carry the remaining
representation and original-runtime limits. No new divergence ID is used.

Focused simulation tests cover signed phase boundaries, independent stage/HP,
legacy and source clocks, terminal nonresurrection, immutable source, no replayed
gameplay events and invalid native tuple rejection. The regression failed on
the frozen pre-change tuple. Both full simulation and game package tests pass.
EN/RU installed witnesses cover real lethal damage, ordinary SAVE, child-process
LOAD, body/owner/position and purse/sack preservation, 128 steps changing
HP -16 to -20 and stage 2 to 3, and another SAV. Fighter and mage routes use
the same production producer. The imported hireling's body also survives an
explicit native save. Final merge gates and corpus census are pending the sole
review; original-runtime acceptance remains M10's owner-run matrix.

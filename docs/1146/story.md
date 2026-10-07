# Idle regeneration

Living actors recover health and mana at three times their ordinary rate when
signed32(current subtick minus action-end subtick) exceeds 80. Movement, turning,
weapon attacks and book/scroll casts write a simulation deadline. Merely holding
an order, failing a command or standing near an enemy does not write it.

Authority: `HERO-REGEN-021`, `SAV-REGENORDER-531`, `ANIM-CLOCK-001` and
`MAGIC-CASTTICK-030`. Native widened arithmetic and original-current signed-word
arithmetic keep their existing respective domains, gates and dispatch schedules.
Rate is selected once before both pools, including the original health callback
boundary. The bonus uses game ticks and therefore follows pause and game speed.

Form86 adds an ID-ordered deadline suffix. Forms50..85 that were already readable
remain readable; their absent clocks start on first execution after any known
remaining action. Original LOAD uses the same explicit bootstrap because the
original action-end origin is not recovered. New native SAVE/LOAD retains exact
deadlines, including uint32 wrap. New actors receive their own starting clock.

`DIV-738` retains arithmetic/callback and complete original-load debt. `DIV-1016`
records deadline-production and bootstrap limits: represented run durations are
used without inventing original animation bytes or post-load chronology. The
original clock notification uses base attack/cast wind-up plus recovery; later
randomized recovery does not rewrite that deadline.

Focused tests pass for strict 80/81 and signed wrap, both arithmetic consumers,
six action producers with literal deadlines, rejected commands, old-save
bootstrap, corrupt suffix refusal and 192-tick native continuation. Extreme
native int32 modifiers retain widened arithmetic without rate3 overflow.
Frozen predecessor forms keep their original bytes and hashes after peeling
only the added suffix; old elapsed idle age is not fabricated.

The registered `TestReleaseIdleRegenerationAfterManualCast` uses installed map
and spell rules with ordinary App pointer input. Its controlled Teleport ends
the run at12; health reaches26 and mana gains36 over260 ticks, while the cursor
stays armed. EN focused witness passes. Final Go, paired EN/RU release,
milestone-2 compatibility and build witnesses belong to the landing record.
No original-save population is intentionally added.

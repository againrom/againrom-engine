# Saved mission outcomes

## Result and contract

An original world SAV restores the human participant's reported mission outcome
and the separate session WIN/LOSE counters. A completed mission reaches the
ordinary Victory-to-town route; campaign defeat remains terminal and permits
reload only. Loading never replays spent notices, rewards, or trigger actions.

The import validates the unique human participant, outcome domain, session
lengths, and latch values before publishing front-end state. Native save/reload
preserves each counter and the outcome independently, including a lost outcome
with no LOSE increment. A city SAV starts a fresh mission rather than carrying
the previous mission's terminal latch.

## Authority and surfaces

SAV-FLAG-027 and SAV-PLAYER-028 identify Player outcome separately from session
counters. SAV-SESS-031 fixes their wire offsets. SAV-RECON-268 establishes the
hybrid restore and excludes blanket saved result-register import. TRIG-END-009
requires loss-first, exact-one reporting and terminal loss. Existing campaign
defeat behaviour follows story 1091 and owner direction.

Touched surfaces: SAV session extraction, deterministic session import/native
validation, original-load orchestration, terminal mission UI, and regression
witnesses. No byte-form extension, new command, trigger-register restore,
unknown session-field interpretation, or original-runtime claim is planned.

## Proof and open debt

Synthetic tests cover outcome 0/1/2, independent counters 0/1/greater-than-one,
simultaneous counters, transactional rejection, detached inputs, first tick,
native round trips, and no repeated rewards. The lawful save666 world source
has independently measured Player outcome 1, WIN 1, LOSE 0, and spent latch 7.
EN/RU App witnesses must load it, preserve the terminal state through a native
save/reload before acknowledgement, and take Victory into town exactly once.
A synthetic defeated source is not evidence of a lawful lost SAV. The saved
terminal outcome opens its panel at tick zero; original panel reappearance
after Continue remains Unknown under DIV-407.

Full world SAV writing, pending orders/effects, and original EN/RU read/resave
compatibility remain outside this slice. Verification results follow here when
measured in [verification.md](verification.md).

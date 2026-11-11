# Divergences -- Second game roots -- Blow

2 row(s). Part of the split described in `docs/DIVERGENCES.md`: read that file first for what a row means, what each column holds, and how a row is found.

## Divergences

Implementation and researched ROM1 behaviour differ, or the owner ruled against a claim.

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-4842 | ROM2 blow / the range draw's arithmetic | Hotfix brief: the arithmetic that consumes the generator matches the original; one seed | `R2-ENGINE-1150` (High for the counts as a function of x, Medium for the full period): the blow's range helper returns (x(n + 1)) >> 15 for the 15-bit output x of the generator state x 214013 + 2531011. `R2-ENGINE-1149` (High): the helper returns 0 for n = 0 without calling the generator; Unknown whether all of the blow's draws share one generator state | A second-game World runs the seeded stream only (DIV-2748). The blow draws through the World's one bounded draw, which in seeded mode answers the high 64 bits of a 64-bit SplitMix64 output times n + 1. The two mappings take different generator outputs, so no value is equal by construction. The blow makes no call at a spread of 0 (`R2-ENGINE-1152`), so its count and order of draws are the original's | FIDELITY-DEBT | The bounded draw serves every World draw of the second game; the other consumers' draw forms are not read here, and the second game's start seed and reseeds have no claim | Claims for the second game's start seed, reseeds and the other World consumers' draw forms, then the second game on the original stream as DIV-2748 names | OPEN |

## Authored where research is silent

No ROM2 claim answers the question, so the implementation authored one.

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-4843 | ROM2 blow / the steps the claims leave thin | - | `R2-ENGINE-1148` (High): before any draw the resolver returns 0 for a null block and for a target whose byte +0x4c has bit 0x08 (`R2-ENGINE-477`: bit 8); no claim read here names the writers or meaning of the bit. `R2-ENGINE-1149` (High): the magic part draws when it is "present and admitted"; the presence test is not stated, and kinds outside 1..5 reach a default arm that `R2-ENGINE-1152` leaves Unknown. Bless and Curse are read only for an attacker whose dword +0x14 is nonzero (`R2-ENGINE-1149`); the unit strike requires an attacker with an owner before it calls the resolver (`R2-ENGINE-481`) | The second game's blow refuses before any draw when the target is out of reach or its maximum health is 0 or below; no unit state stands for bit 0x08. The magic part is present when its base or spread is nonzero, and its selector names one of the five protections. Bless and Curse are read for every attacker | UNKNOWN | The claims give the test but not who carries bit 0x08, nor the magic part's presence test or default arm; implementing either would guess | A claim naming the writers and meaning of target +0x4c bit 0x08, and a read of the magic part's presence test and default arm | OPEN |

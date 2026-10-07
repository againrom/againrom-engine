# Provenance — the step cost readout

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 (the rate is the movement law's `v`) | `TERR-MOVE-056` — `R1088`, the multiply, the height tilt, the byte-add mean, the signed divide and the `[1,63]` clamp, each with its address | High for every term and its arithmetic form; the row's own Medium half is the corpus **range** figures, which nothing here asserts |
| FR-2 (the transit in ticks) | `TERR-MOVE-056` — `mover[0xaa] = ceil(256 / max(...))` over the per-axis steps | High |
| AC-5 (source and destination are not interchangeable) | `TERR-MOVE-056` — `d = clamp((i8)(height[src] − height[dst]), −32, +32)` and `v += (v*d) >> 6` **arithmetic**, so the sign of the difference survives the shift | High |
| AC-6 (a diagonal transit is the longer) | `TERR-MOVE-056` — the per-axis steps are scaled by the `0.707` double at `L06908` when `dx*dy != 0`, and the cell to cross is not | High |
| AC-4 (the group term replaces the unit's own speed) | `TERR-MOVE-056` — the raw speed is `actor->[0x70]->[0x3c]->[0x44]` when that byte is nonzero, else `(i16)actor->[0x8c]` | High |
| FR-7, first bullet (the multiplier's source) | `TERR-MOVE-056` — `SpeedMultiplier` is `world+0x58db4`, from `data/map.reg [Path Finding]`, shipped value 8 | High |
| FR-7, first bullet (that the constant is right on what ships) | `MOVE-PARAM-006` — `world.res:data/map.reg [Path Finding]` ships all seven scalars equal to the code defaults, **identically in the EN and RU roots**, `SpeedMultiplier` among them | High |
| DD-8's rate row (what the bare integer means) | `MOVE-STEP-010` — movement is sub-cell at **1/256 of a cell per axis** | High |
| FR-7, second bullet (the cost accessor is not a pure read) | `TERR-MOVE-056` — `cost(cell)` is `R1087`; on `block[cell] & 0x20` with a nonzero byte at `record+0xe` of the `world+0x540b8` table it stores `cost >> 2` back, at `L05498`/`L05499` | High |

Every arithmetic term FR-1 and FR-2 rest on is already implemented in this tree and is not
re-derived by this story; the claim is cited because the story's contract asserts **which** terms
those are and **which two are missing**, and that assertion is only checkable against the row.

## Ours by choice

Nothing below is asserted by any source. Each is engineering that can change later without
contradicting anyone.

| Choice | Note |
|---|---|
| That a debug readout exists at all, and that these two values belong in it | the readout is AUTHORED end to end; nothing published describes a parameter box in the original |
| The **pair** the law is evaluated on — the mover's own cell to the cell under the cursor | the reading of «относительно персонажа» this story owns; the alternative readings are settled in `plan.md` |
| Both rows' existence, their labels, their order and their field numbers | no source ranks or names these fields |
| The marker word for a pair more than one cell apart, and that the values are still stated under it | |
| The four absence causes and which of them omits a row versus states a marker | |
| That the rate is stated as the bare integer the law produces, with no unit suffix | the sub-cell denominator is a simulation constant and stating it would put one in the drawing tier |
| The shape of the seam — a query of builtins installed once, rather than a value pushed per frame | forced in kind by the tier's permitted imports; the particular signature is ours |
| That an unrated mover states the marker rather than the number a total law yields | see *Open*, first row: this is a choice about **our** law's totality, not a claim about the original's |

## Open

| What | Why it is left undecided |
|---|---|
| Whether the original computes anything at all for a mover of zero effective speed | `TERR-MOVE-056` transcribes the arithmetic but not the caller's own guard, so what the original does with a zero speed is not established. This tree's advance does not rate such a mover, and the readout follows the advance rather than guessing the original |
| The movement multiplier as a **map parameter** | the key is decoded and named and both shipped roots carry the default, so the constant is right for everything that ships; what is open is a **customised** map, which this tree has no `data/map.reg` reader to notice. Deliberately assigned no map-varying meaning here |
| The cost accessor's conditional rewrite | the block bit, the table and the write-back are decoded; this tree reads neither the bit nor the table and rewrites no cost byte. The stated figure is therefore taken from an unmodified plane, and no attempt is made to model the modified one |
| The direction table and the facing byte that pick a step's destination | decoded in the same row; this tree derives a step from its route search and the readout takes the destination as an input, so neither is exercised |

## Removed

| Dropped | Why |
|---|---|
| A row stating the mean cost byte the law divided by | it is computed inside the rate function, after a byte-width add that can wrap. Surfacing it means either recomputing it beside the authority — the exact failure the readout's own rule names — or widening the law's return for a diagnostic. Neither is worth a row the rate already implies |
| A per-cell marker for the cost accessor's write-back | it cannot be made honest: deciding whether a cell is one the original would rewrite needs the block bit and the table record this tree does not read, so any marker would be a guess dressed as a measurement. Disclosed in prose instead |
| Stating the raw cost byte of the cell under the cursor beside the mover-relative figure | it is the number the story exists to **not** state; two rows where one is per-cell and one is per-mover invite the per-cell one to be read as the answer |

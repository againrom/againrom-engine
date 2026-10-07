# 1015 — contract

## What this story settles

The owner played the school room in `builds/current/` on 2026-08-18 and reported that the placement
of the skill sprites is wrong for the mage and for the fighter, with two screenshots. Four separate
defects produce what the screenshots show, and a fifth was found beside them while reading the room:

1. **The fighter never gets its own column face.** The room's background picture bakes one face of
   the training column into `trnhall.bmp`, and it is the mage's. The fighter's five weapon patches
   are drawn over the mage's engraved symbols.
2. **The mage's five icons are drawn at the fighter's five rectangles.** `schoolSkillRects`
   (`pkg/ui/townshell.go`) is one array of five, indexed by slot and shared by both classes, so all
   five mage icons land in a vertical stack in the middle of the column instead of at the five
   scattered positions the mage face has.
3. **Pure black is drawn opaque.** The fighter's `sword` patch is a 24-bit DIB whose canvas is pure
   black around the blade; `readChargenBMP` gives every pixel alpha `0xff`, so the black canvas is
   painted over the column. This is `DIV-013`'s defect class at a second screen.
4. **A click selects a different skill from the one it lands on.** `schoolMaskSlot` reproduces the
   original's own visual-to-stored permutation `1,2,4,3,5` (`TOWN-GENERAL-106`) while the draw side
   was re-ordered to detailed character generation's `1,2,3,4,5` under `DIV-121`. The two halves
   disagree at slots 2 and 3 on **both** classes: a click on the Bludgen row selects Pike, a click on
   the Air symbol selects Earth, and each spends the other's price.
5. **A party-picker step keeps a pending skill selection and its quoted price.** `TOWN-138` (High,
   `EXP-0193`, pinned at `040688c`) reads the original's own step routine: it writes `-1` into
   `view+0x34c` and `0` into `view+0x354`, the selected-slot and price fields `TOWN-GENERAL-107` and
   `TOWN-GENERAL-108` name. `shopStepMember` (`pkg/game/shopview.go`) does not touch `schoolCell`.

## What will work after this story

- The column shows the face that belongs to the selected member's class: the fighter face for a
  fighter, the mage face for a mage. Both faces are shipped art (`interface/training/column/`), and
  the room background's own baked face is no longer what decides which one a player sees.
- Each class's five skill icons are drawn at that class's own five positions, so a lit icon replaces
  the engraved symbol underneath it rather than covering a different one.
- No black rectangle appears on the column. Pure black is cleared to zero alpha on the school's
  patches, by the same `keyBlack` rule `DIV-013` records for the shop.
- A click on a skill selects that skill, on both classes. The Train button then quotes and spends
  that skill's price.
- Stepping the party picker clears the pending selection and its quoted price, per `TOWN-138`.

## The observable result

`builds/current/`, in the school room, on both preserved roots: two columns that match the shipped
art rather than one column wearing the other's face. Concretely, and measured rather than asserted,
by a new developer tool `cmd/schoolcheck`:

- the rotation frame the room background bakes, and at which rectangle, printed as a match fraction
  against the next-best candidate over a search window, not as an assertion;
- the ten skill patches' own placements inside their class's face, each cross-checked against the
  bounding box of its own colour code in that class's `mask.bmp` — two instruments, and the tool
  fails when they disagree;
- the pure-black fraction of each of the thirty patches, so the keying rule's population is a number
  and not a claim.

`pipeline/check-milestone.sh`'s script-gap census is not expected to move — no script opcode is
touched — and is measured in `closure.md`, not assumed.

## Claims this story is built on

| Claim | Confidence | What it supplies |
|---|---|---|
| `TOWN-067` | High | `+0x31c==0` is the fighter panel and `==0xf` is the mage panel, closed by matching each panel rect to its class mask's dimensions. The two panel rects: mage `(188,188,288,308)`, fighter `(192,192,284,312)`. |
| `TOWN-061` | High | The school's own paint routine `R1487` and its eight-step order. Step 4 draws a state-indexed decorative sprite chosen from `+0x30c` by `+0x31c` — the column face — and step 5 the active panel's five skill icons. This story builds steps 4 and 5. |
| `TOWN-068` | High for geometry | The five fighter icon rectangles: `(200,196,280,228)`, `(200,216,280,252)`, `(200,272,280,288)`, `(200,248,280,276)`, `(200,288,280,308)`. Its shared-array clause is a conflict this story records rather than builds; see the divergence allocation. Its own unresolved clause — why the array's index order and its visual y-order diverge — is answered by `TOWN-GENERAL-106`'s permutation together with the five patch heights. |
| `TOWN-017`, `TOWN-018`, `TOWN-019` | High | The two-panel raster-mask hit test, the three art states per skill per class, the per-slot enabled flag. |
| `TOWN-GENERAL-106` | High | Visual indices 0..4 map to stored slots `1,2,4,3,5`. This story does not build that permutation — `DIV-121` is the owner's accepted decision to use `1,2,3,4,5` — but it is what identifies defect 4 as an inconsistency rather than a choice. |
| `TOWN-138` | High | What the original's school picker step resets. Defect 5. |

## Aspects that apply

| Aspect | Applies |
|---|---|
| Data | Yes — the column's rotation frames are a shipped archive series this build has never read. No format changes. |
| Runtime state | Yes — one session value: which face is showing. No new persisted field. |
| Simulation | No — `pkg/sim` is untouched. No hashed field, no serialized form, no `formatVersion` change. |
| Player input | Yes — the mask hit test's own answer is defect 4. |
| AI | No. |
| UI / HUD | Yes — the whole subject. |
| Triggers / scripts | No. |
| Inventory / equipment | No. |
| Persistence / save-load | No. |
| Campaign / session | Yes, narrowly — the pending school selection is session state and defect 5 is when it is discarded. |
| Shipped content | Yes — every rectangle in this story is measured against both preserved installs' own art, and the tool prints both roots. |
| Interactions with existing mechanics | Yes — the Train button reads the selected slot, so defect 4 changes which skill is bought and defect 5 changes when the quote is void. Detailed character generation reads a separate art set (`interface/chrgen/`) through its own origins table and is not touched. |

## Domains touched

**Client** (`pkg/ui/townshell.go`, `pkg/game/townschoolart.go`, `pkg/game/townshell.go`), and one
line of **Town & Economy**'s session state in `pkg/game/shopview.go` for defect 5. One domain and a
seam, which is why the ceiling below is the ordinary one.

## The ceiling this contract sets

**Three adversarial passes.** Five behaviours under one contract, no reach into hashed simulation
state, one domain touched. `AGENTS.md`'s rule: a contract states its own ceiling before the lane
opens, and reaching it is a scoping diagnosis rather than a failure.

The four defect classes are related by one subject — what the column draws and what a click on it
means — and by one file. Splitting them buys four doc stacks and four landings for one screen the
owner looks at once.

## Divergence allocation

`DIV-142` through `DIV-146` are reserved. Expected disposition:

- `DIV-142` — the rotation between the two faces is not animated. The install ships sixteen frames;
  this build cuts between the two rest frames. `TOWN-061` establishes that a state-indexed sprite is
  chosen by `+0x31c` and `TOWN-017` names only two values of it; whether anything advances it, and at
  what cadence, is `EXP-0195`'s open question. **FIDELITY-DEBT.**
- `DIV-143` — defect 5's residual, if the fix is partial. Returned unused if the reset is total.
- `DIV-144` — pure black keyed to zero alpha on the school's patches. Read off the shipped art; the
  original's own blitter is undecoded, exactly as `DIV-013` records for the shop. **UNKNOWN.**
- `DIV-145` — the mage panel's five icon rectangles are measured from the shipped art, not read from
  `.text`. `TOWN-068` publishes one shared array serving both classes; this build uses two.
  **CONFLICT**, and the row must say plainly that the claim's own scope note — that the shared-array
  conclusion came from the paint loop's index — is why the conflict is recorded rather than resolved
  here. `EXP-0195` is open on exactly this question and its answer is the authority.
- `DIV-146` — spare, per the standing rule that a limit met at its ceiling is raised.

`DIV-121` is amended, not replaced: it records the owner's choice of detailed-generation order
`1,2,3,4,5` for the visible order, and must now say that the choice binds the mask hit test too. The
row as written is why defect 4 could stand — it named the draw side and not the click side.

## Concurrency

`EXP-0195` (`exp-0195-school-column`) is open in the research repo and asks what the original's own
paint routine does for steps 4 and 5, per class, at instruction level. It is deliberately not told
anything this contract measured, and its answer is the authority over every rectangle here. This
story proceeds on the owner's directive rather than waiting: an owner-directed feature proceeds on
his intent even where research is missing or partial, and the gap becomes a ledger row rather than a
hold. The reconciliation belongs in `closure.md`.

No implementation lane is open. The submodule pin is at research `040688c` and is frozen for the
duration of this story.

## Out of scope

- The rotation animation itself (`DIV-142`). The frames are shipped and the cadence is not decoded;
  adding a timed animation without one is authoring a number nobody can check.
- The original's own visual-to-stored permutation `1,2,4,3,5`. `DIV-121` is the owner's accepted
  decision and this story keeps it, on both sides instead of one.
- The school's price, Train command and General slot (`TOWN-GENERAL-106`..`108`), beyond defect 5's
  discarding of a stale quote.
- Detailed character generation's own skill column, which reads a different art set through a
  different origins table and shows the defect in neither screenshot.
- Whatever else `interface/training/` ships with no consumer here (`EXP-0195` question 6).

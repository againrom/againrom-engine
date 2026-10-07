# Verification — 0057-tier-palette

Toolchain: the `go.mod` pin. Roots: `gameversions\en` and `gameversions\ru`, read-only.
Tests carry no game data; every fixture below is synthesised in test code.

## The gate

```
(go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
 sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
 sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))")
EXIT=0
```

FAIL set **empty**, byte-identical to the branch point's, which was also `EXIT=0` with an empty
FAIL set. `check-sdd-audit` reports notes and warnings and none is enforced; `check-doc-budget`
reports the ten pre-existing declared overruns, none of them this story's.

```
sh scripts/check-no-game-assets.sh --history
```

clean over the full log — the gate that backs golden rule 1, run because this story handles
colour tables.

Deletion set of the landing:

```
git diff --diff-filter=D --name-only <branch point> HEAD   ->   (empty)
```

## AC by AC

| AC | How | Result |
|---|---|---|
| AC-1 | `pkg/formats/pal`: a synthesised `0x436` stream with three channels that disagree at every index | the 256 colours are the bytes at `0x36` read `[B,G,R,X]`, `X` dropped; entry 0 and both channel extremes carried |
| AC-2 | every length below `0x436` (0…1077, the whole domain) and all **65 536** two-byte prefixes | 1077 short-stream refusals naming the length; **exactly 1 of 65 536** prefixes accepted; the 16-table shape refused; no input panics |
| AC-3 | `pkg/data`: a class on a stored path with mixed case and backslashes, byte-for-byte | `palette.pal`, `palette2.pal`, `palette3.pal`, `palette4.pal` in the sheet's directory, `palette5.pal` past the count; `""` at 0, -1, -4096 and for a `File`-less class; a separator-less base answers the bare name |
| AC-4 | the clamp over -4096, -1, 0, 1, 2, 3, 4, 5, 7, 2^30 | 0, 0, 0, 1, 2, 3, 4, 4, 4, 4 |
| AC-5 | `pkg/game`: a synthetic container carrying a sheet and four tables — equal, differing, absent, refused | tier 1 is the base slice **by identity**; tier 2 carries the second table over the **same backing array** of pixels; tiers 3 and 4 are nil and fall back; 2 of 4 fell back |
| AC-6 | the selector at -4096, -1, 0, 3, 4, 5, 2^30 over a class with four tier slots | every one answers the base slice; the in-range non-empty tier answers its own; nil receiver and frameless class answer nil |
| AC-7 | a synthetic map placing one class at tier columns 1 and 3 and once through the npc arm, pushed to the draw seam | the two carry different frames whose colour tables differ and whose pixel indices and geometry are equal; the npc-arm placement states no tier and draws the sheet's own |
| AC-8 | the same map's world against one `mapload.FromALMWith` builds directly | entities equal one for one, digest equal, `MarshalBinary` equal |
| AC-9 | `terraintool tiers` on each root | below |
| AC-10 | `terraintool tiers -class N -out` | five stills written outside both repositories |
| AC-11 | a class with no tiers, and the untiered class of the AC-5 bundle | its frames at every tier from -1 to 6 are its own, by identity |

## The properties

| # | How it is witnessed | Result |
|---|---|---|
| **P-1** | AC-5's shared-backing-array assertion: a recoloured frame is a struct copy, so no line can write through the base pixels; the deep-copy mutant is the same assertion read the other way | holds |
| **P-2** | AC-5 and AC-6 as identity, not as equal contents: an equal table yields the base slice itself, and the corpus run reports 16 of 16 classes taking it at tier 1 | holds |
| **P-3** | AC-1 decodes entry 0 like every other and no code on the path reads, reserves or rewrites it; the frame's own per-pixel flag stays the only answer to which pixels are holes | holds |
| **P-4** | `TestTheTierIsFixedAtMapOpen`: two builds with no advance between draw the same frames and the lookup does not grow | holds |
| **P-5** | AC-8, plus AC-11 and the untiered arm of AC-5: a map with no tiered placement resolves no tier, and every class with none draws its own frames by identity | holds |

## The success criteria

| # | Where it was run | Result |
|---|---|---|
| **SC-1** | `pkg/formats/pal` — the exactness case, the 1077 short streams, the 65 536 prefixes, the totality sweep | pass |
| **SC-2** | `pkg/data` — the byte-for-byte name cases and the ten-value clamp | pass |
| **SC-3** | `pkg/render/terrain` — the selector at seven out-of-range tiers, a nil receiver and a frameless class | pass |
| **SC-4** | `pkg/game` — the synthetic container in all four tier states, plus the absent-sheet case | pass |
| **SC-5** | `pkg/game` — `TestTheTierColumnReachesTheDrawnFrame` and `TestTheTierSurvivesTheCorpseSubstitution` | pass |
| **SC-6** | `pkg/game` — `TestTierResolutionReachesNoWorldState`: entities, digest and byte form | pass |
| **SC-7** | `terraintool tiers` on both roots, and the discriminating check below | pass — 55/55 loaded, 0 fell back, 16/16 sheet-own at tier 1, 4/4 on the published word |
| **SC-8** | the five stills under `review\0057-tier-palette\` | pass |
| **SC-9** | the gate above and the deletion set | pass — `EXIT=0`, FAIL set empty, deletion set empty |

## The corpus run — AC-9, both roots

```
terraintool tiers -assets <root>
...
class 64 "Goblin": 4 tier(s), 4 loaded, 0 fell back, 248 frames, tier 1 the sheet's own
class 71 "Dragon": 4 tier(s), 4 loaded, 0 fell back, 164 frames, tier 1 the sheet's own
class 72 "Death Star": 1 tier(s), 1 loaded, 0 fell back, 154 frames, tier 1 the sheet's own
tiers: 16 of 34 classes carry tiers, 55 declared, 55 loaded, 0 fell back, 16 take the sheet's own table at tier 1
```

The two roots print **the same 17 lines, token for token**. So on both:

- **55 of 55** built addresses exist, are at least `0x436` bytes, begin `BM` and decode.
- **0 fell back** — no absent entry, no refused stream, no missing sheet.
- **16 of 16** classes take the sheet's own table at tier 1, i.e. tier 1's 1024 bytes equal the
  sheet's embedded palette on every class that has one. Research measured 13/13 over the tiered
  classes; this is the same fact over the three single-tier classes as well.
- The key's distribution is the shipped one: 16 classes of 34 carry tiers, three of them one tier
  and thirteen of them four — 3 + 52 = 55.

**The discriminating check.** Re-executed through this build's own committed path —
`pal.Decode` → `terrain.SpriteChannel` at the shipped daytime row, tint 0 → the RGB565 pack — for
one class, one palette entry and one row, against the four words the research spec publishes:
**4 of 4 agree on the EN root and 4 of 4 on the RU root.** A negative control passing one wrong
word three times scored 1 of 4, so the comparison discriminates. The probe was a throwaway and
took the expected words on its command line: neither it nor this file carries a colour, because
a decoded palette entry may not enter the repository even as evidence.

## The map — AC-10's subject

The tier distribution of a shipped map, from the existing definition-table dump. Its second key is
the tier:

```
classdump -databin <root>\world.res <root>\Forester.alm   |   grep 'key 0x0040/'
     36 key 0x0040/0x0001
     37 key 0x0040/0x0002
     45 key 0x0040/0x0003
      7 key 0x0040/0x0004
```

Identical on both roots. `Forester.alm` therefore carries **all four tiers of the Goblin class**,
and ten further classes on it carry all four as well. It ships in both roots, which is why it is
the build's named map rather than `Cross.ALM`, which has a fuller spread and is EN-only.

## The deliverable

`builds\0057-tier-palette\` holds `againrom.exe`, `terraintool.exe` and the run note naming the
map, the invocation and where to look. `againrom -check` on each root:

```
EN: againrom: 38 map rows, 8 of 8 buttons have a mask region
RU: againrom: 34 map rows, 8 of 8 buttons have a mask region
```

Owner-review artifact: `<seat>\review\0057-tier-palette\` — five stills, one per
class, each that class's tiers side by side in ascending order: Goblin (64), Ogre (66), Fat troll
(68), Dragon (71), Orc with sabre (80). Outside both repositories. The troll's four read green,
blue-grey, purple and olive; the goblin's orange, yellow-green, brown and grey-blue.

## Green-but-hollow audit

**One hollow pass was found and fixed inside T5, by the mutant it was supposed to die to.**
`TestTheTierSurvivesTheCorpseSubstitution` was green against a build whose corpse arm read the
base slice, because the fixture class's animation descriptor carried **no dying block**: the death
selection refused it, execution fell through to the live arm, and the assertion was reading the
live arm's tier all along. The fixture now carries a dying block, and the test asserts both that
the drawn class is the substituted one and that the death selection accepts that descriptor — so
the arm cannot be skipped without the test saying so. With that fixed the mutant fails as it
should.

Mutants run, each against the tests named for it:

| Mutant | Outcome |
|---|---|
| the equal-table arm copies the base slice instead of returning it | killed — "tier 1 is not the base slice by identity" |
| the recoloured frame deep-copies the pixel slice | killed — "does not share the base frame's pixels" |
| the per-tier memo keyed on the sheet address alone | killed — "tier 2 answered the base slice" |
| the corpse arm left on the base slice | killed — "the corpse drew mark 0, want 3" (after the fixture fix above) |
| a non-stat-arm placement given tier 1 | killed — "entity 6: present = true, want false" |

**One named mutant is not reachable and is recorded as such.** T5 names "the lookup keyed by loop
index rather than by the minted id". The world builder mints id `i` for the `i`-th record, so the
two are the same integers by contract and no fixture can separate them. What guards it is that
contract, not a test, and the code cites it.

## Two things measured and not asserted

- **`palette_.pal` ships beside every `palette.pal`** — 16 further nodes on each root — and matches
  no name the construction builds. Not read; research lists it as open.
- **The three 1086-byte `palette.pal` nodes** at `units/heroes`, `units/heroes_l` and
  `units/humans` are not in any class's sheet directory, so no built name reaches them either.

## What was not verified

- **That the game window draws it.** The gate is headless and no test opens a window. What is
  verified is the seam — the frames the draw hands over — plus the stills, which are the render
  tier's own lit blit of the same frames. The picture on screen is the owner's acceptance test.
- **Team colouring**, which this story does not build (spec, Out of scope).
- **RGB565 packing**, which this build does not do at all; the discriminating check above packs
  once, in the throwaway probe, to compare against a published word.
- **Whether a modded `palette.pal` differing from its sheet is honoured.** The code path is the
  same one tier 2 takes and is covered synthetically (AC-5), but no shipped root exercises it —
  all 16 are equal.

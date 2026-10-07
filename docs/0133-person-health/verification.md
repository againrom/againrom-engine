# verification — 0133

Environment: Go 1.26.1 (the toolchain `go.mod` pins), windows/amd64, 2026-08-10. Both lawful roots
read: `gameversions/en` and `gameversions/ru`. Every figure below is output this tree printed; none
is copied from a source or computed by hand.

## The gate

```
go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -count=1 -trimpath ./...
```

Silent on the first three; every package `ok` on the fourth, on a clean working tree. Run again after
each of the two implementation commits and once more on the final tree. `scripts/check-no-game-assets.sh`
(tree scan) clean, `check-doc-budget.sh` clean, `check-hotfix-ledger.sh` ok, `check-sdd-audit.sh`
zero FAIL.

## Acceptance criteria

| AC | Evidence |
|---|---|
| AC-1 | `TestAWorldBuiltWithATableCarriesTheResolvedHealth`, `TestOnlyAMatchedUnitsEntryCanYieldANonGroundMover`, `TestAPlacedPersonCarriesHisDerivedHealthRateAndCadence` — a person whose row states 30 is minted at 70 and 70 |
| AC-2 | `TestAC2ZeroBodyDerivesNoHealth` |
| AC-3 | `TestAC3HealthColumnDoesNotMoveTheMaximum` |
| AC-4 | `TestAC4NonzeroBodyDerivesPositiveWithNoHealthColumn` |
| AC-5 | `TestTheDifficultyDoesNotReachAPerson` — one number across all three settings |
| AC-6 | `TestAWorldBuiltWithATableCarriesTheResolvedHealth`'s creature and unresolved arms, unchanged |
| AC-7 | `TestAZeroProfileMemberIsByteForByteTodaysMember`, `TestAZeroProfileTrainedHeroDoesNotHealFromExperience` |
| AC-8 | `TestAWorldRefusesAPersonRowItCannotStream` |
| AC-9 | the developer run below |
| AC-10 | the census below |
| AC-11 | `TestAC11DerivedMaximumIsIdempotent` |
| P-1, P-4 | AC-1's three tests, plus the census: every person-arm placement of both campaign maps reads a derived number |
| P-2 | AC-8 |
| P-3 | AC-11 |

AC-8's first draft asserted the refusal for **any** row too short to stream, and the test written for
it built a world instead: a row truncated below the search's own key column is reached by nothing, so
it is no match rather than a refusal and takes the provisional value. The criterion and P-2 were
narrowed to the row a search reaches, which is what the code does and what the test now pins.

**The gate is witnessed by reverting it, not by reading the assertion.** With the health arm's gate
put back to the column flag and every test left standing, `TestAC2ZeroBodyDerivesNoHealth`,
`TestAC3HealthColumnDoesNotMoveTheMaximum`, `TestAC4NonzeroBodyDerivesPositiveWithNoHealthColumn`
and `TestAZeroProfileTrainedHeroDoesNotHealFromExperience` all fail.

## AC-9 — mission 20, both roots

```
go run ./cmd/restool cat <root>/scenario.res 20.alm > <tmp>/20.alm
go run ./cmd/classdump -databin <root>/world.res <tmp>/20.alm
```

The person placements, before and after, EN and RU **byte-identical to each other on both runs**
(the two roots' extracted lines hash the same):

```
before   49 key 0x000b/0x0000  arm server-id entry   58  healthMax    20  speed   17  domain ground
before   50 key 0x000b/0x0000  arm server-id entry   58  healthMax    20  speed   17  domain ground
before   51 key 0x000b/0x0000  arm server-id entry   58  healthMax    20  speed   17  domain ground
after    49 key 0x000b/0x0000  arm server-id entry   58  owner    5  healthMax   120  speed   17  domain ground
after    50 key 0x000b/0x0000  arm server-id entry   58  owner    5  healthMax   120  speed   17  domain ground
after    51 key 0x000b/0x0000  arm server-id entry   58  owner    5  healthMax   120  speed   17  domain ground
```

The rest of the map's people move with them: entry 116, placed ten times, 90 → 66; entry 118 once,
126 → 98. `almtool roster 20.alm` names the five roster records — slot 1 `Self` owning **0**
placements, slot 5 `Merchant` owning 4 — so the three placements above are the merchant's, and the
map places nobody for the player at all.

## AC-9 — mission 10, both roots

Same commands against `10.alm`. Again EN and RU are identical:

```
before    0 arm server-id entry  203  healthMax    15      after  owner 3  healthMax    10
before   22 arm server-id entry  100  healthMax   108      after  owner 2  healthMax    82
before   25 arm server-id entry  101  healthMax    90      after  owner 2  healthMax    66
before   29 arm server-id entry   58  healthMax    20      after  owner 2  healthMax   120
before   33 arm server-id entry   99  healthMax    45      after  owner 2  healthMax    31
before   34 arm server-id entry  181  healthMax    45      after  owner 2  healthMax    31
```

The direction is not uniform and is not meant to be: eleven of the mission's fourteen resolved people
come out **weaker**, three come out six times stronger.

## AC-10 — the whole person collection

```
go run ./cmd/classdump -databin <root>/world.res
```

```
EN   humans health: 210 row(s), column sum 20539, derived sum 31478
       derived above column 123, below 86, equal 1; zero column 0, zero derived 0
       largest ratio: "M151_SuperAxemen" derived 607 against column 30
RU   humans health: 210 row(s), column sum 20539, derived sum 31428
       derived above column 123, below 86, equal 1; zero column 0, zero derived 0
       largest ratio: "M151_SuperAxemen" derived 607 against column 30
```

**Against `HERO-HP-072`, which publishes the same census over 215 rows.** It reports column sum
20 809, derived sum 31 828 (EN), the largest ratio 20.2× on `M151_SuperAxemen` at 607 against 30,
equality on 1 row of 215, 128 above and 86 below, and EN differing from RU on exactly one row,
`M120_BrigandChief`, 654 against 604.

Four figures agree exactly and independently: the largest-ratio row **and both its numbers**; the
below count, 86; equality on **exactly one** row; and the EN−RU derived-sum difference, **50**, which
is `M120_BrigandChief`'s own 654 − 604.

The whole of the remaining difference is five rows, and it reconciles to the byte:

* the collection holds 215 written entries of which **210 carry parameters** (the tool's own
  collection table, `Humans 215 / 28 / 210`). The five paramless rows are refused by this build
  wherever a map could place them, so the census does not walk them. At the constructor's defaults
  each derives 70: 5 × 70 = **350** = 31 828 − 31 478, and each derives above its column, which is
  128 − 123 = **5**.
* those five rows' health columns account for 250 of the 270 the column sums differ by, at a default
  of 50 rather than the 30 this tree uses — and the residual **20** is one row: `ManMage_Unarmed` is
  the only censused row leaving its health cell empty (measured by instrumenting the census loop
  temporarily and reverting it), so it takes 30 here where `HERO-HP-072` reports its column as 50.
  250 + 20 = 270, exactly.

That residual is a **divergence this story does not close and does not create**: a person's own
constructor runs a second defaults routine — health 30 → 50, Mind and Spirit 20 → 30, speed 10 → 16 —
and this tree streams a person over the *creature* constructor's defaults. It touches no derived
number in the census (the derived sums reconcile exactly), only a column this story stopped reading.

## Success criteria

| # | Outcome |
|---|---|
| SC-1 | met — `TestAWorldBuiltWithATableCarriesTheResolvedHealth`: the fixture's row states 30 and the entity is 70/70 |
| SC-2 | met — `TestAC2ZeroBodyDerivesNoHealth` |
| SC-3 | met — `TestAC3HealthColumnDoesNotMoveTheMaximum` |
| SC-4 | met — `TestAC4NonzeroBodyDerivesPositiveWithNoHealthColumn` |
| SC-5 | met — `TestTheDifficultyDoesNotReachAPerson` |
| SC-6 | met — the creature and unresolved arms of `TestAWorldBuiltWithATableCarriesTheResolvedHealth`, unmoved |
| SC-7 | met — `TestAZeroProfileMemberIsByteForByteTodaysMember` |
| SC-8 | met — `TestAWorldRefusesAPersonRowItCannotStream`, on the narrowed AC-8 |
| SC-9 | met — `TestAC11DerivedMaximumIsIdempotent` |
| SC-10 | met — the two developer runs above, both roots, missions 20 and 10 |
| SC-11 | met — the census above; four published figures reproduced exactly and the rest reconciled |
| SC-12 | met — the gate at the top of this file |

## Limitations

* **The NPC arm is unresolved in this tool.** Two placements per mission take it, and the report
  builds its table from `world.res` alone, without the scenario registry the arm searches — so they
  read the loader's provisional 100, identically before and after. The game's own table resolves
  them; this instrument does not, and did not before this story either.
* **The owning slot is exercised at 0 only** by the synthetic fixture, because the test map builder
  has no owner field for a placement. Its wiring is witnessed there and its nonzero values only by
  the developer runs above.
* **A person's mana maximum is still his row's column** — out of scope, and so a placed caster's two
  pools now come from two different places.
* No `builds/` directory was produced; this story ships no new runnable and the brief named none.

## Conclusion

Every AC has evidence. The arithmetic is not merely self-consistent: it reproduces, independently and
exactly, the four figures of the published census that discriminate between the readings this story
could have taken — an inverted class multiplier or a gate in the wrong place misses both sums by
thousands and moves the largest-ratio row. The confidence claimed is therefore *the placed person's
health maximum is the number the original derives*, over both lawful roots and the whole shipped
collection, with the five refused rows and one empty health cell named rather than absorbed.

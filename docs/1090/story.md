# 1090 — consumables

## Player result

Potions and scrolls retain distinct ordered instances from installed source and
merchant through use and one-unit consumption. Mission pack double-click or doll
drop uses potions; scrolls arm a map target without learning or mana cost. The
accepted action reserves one complete scroll until release. Cancellation before
release returns it. Town potion use shares the same effect operation; scroll
targeting remains mission-map-only.

Initial base `f136861b8f8ce338f06c9a36afe1f17c6f9f7f18`; reconciled implementation
`6c74bbf215d4375c562de5333fc1a13af8646bf7`; promoted evidence pin
`ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`.

## Authority and state

`ITEM-USE-112`, `ITEM-USE-113`, `ITEM-USE-114`, `ITEM-CASTSTATE-056` and
`MAGIC-CAST-003` separate display arming, reservation, action and completion.
`SHOP-GEN-005` gives six named potion stocks. `SHOP-CONSUME-073` distinguishes
scroll power from quantity, price-window admission and book exclusions.
`ITEM-VALUE-115` corrects first-effect book and separate scroll valuation.
`ITEM-EFFGRAM-070`, `ITEM-EFFMODE-073`, `ITEM-WEAR-058` and `ITEM-MAGVAL-090`
provide parsing, kind identity and potion value.

`MAGIC-CONSUME-142`, `MAGIC-CONSUME-143`, `MAGIC-CONSUME-144` supply all thirteen
shipped potion effects, permanent-stat caps, id-zero cross-kind replacement and
timed expiry. Earned gains are distinct from equipment modifiers; derived combat
and character values refresh after use. One active potion attachment carries
kind/mode/magnitude/remaining. Town has no simulation clock. Campaign carry
applies the remaining attachment once to a new actor; native restore never
reapplies an already-applied effect.

Inner save form 68 appends four earned gains and four derived headroom words per
entity, plus a bounded reserved-scroll section. Old forms default to no gains,
no timer and no reservation; a matching mission actor repairs old headroom from
its installed definition. Party carry adds an optional potion attachment,
default nil in old gob envelopes. Historical pins retain explicit peels.

## Proof

Focused potion/scroll, bounded malformed-save, stale-slot and delayed-tap tests
pass. The pre-review candidate passed full `go test -trimpath -count=1 ./...`,
gofmt, diff whitespace and the tracked-tree no-game-assets guard. The first Go run caught a
nil-shop epoch projection and the unregistered approach destination writer;
both were corrected before the clean run. The writer is a genuine commanded
approach; existing Move arrival and cast-busy guards retain its ownership.
EN/RU production-App mission witnesses exercise healing 2→1→0; Body gain and
derived maximum health; timed replacement; installed scroll art, display cancel,
real pointer targeting, reservation and damage without learning/mana use.
Exact native checkpoints cover before/after use and in-flight scrolls, plus 33
resumed ticks. Both town App source/use/purchase/native/next-mission witnesses
pass, as does the existing shop-book purchase/read/carry regression. Five new
`TestReleaseConsumable1090*` tests are registered in the release population.
These are Againrom App/CPU-composition witnesses, not desktop GUI or ROM1 runs.

The paired EN/RU 21-step `scenarios/1090-consumables.json` imports the actual
666 potion stack, drives pointer double-use 2→1, reloads native state, then
uses 1→0. The read-only owner M71 save loads at tick 56569 with three party
members, ten entities, six commanded and two legacy-disclosure pages; current
hash is `1ef640eff1bf6fb2`. No resave or playthrough is claimed.

EN missionrun tick-one census remains M10 16/27/12 and M20 14/15/11
checks/instants/triggers, matching `pipeline/milestone-baseline.txt`; both have
zero UNSUPPORTED lines. This slice changes the visible consumable cycle, not
the script population. DIV-465 closes; DIV-582..584 retain the integration
limits. Pre/post allocation sweeps report 32 namespaces, zero missing answers.
The divergence guard reports 275 live rows and 68 existing retraction matches;
no new row relies on a retracted clause. The seat owns correction verification
against the unchanged independent overlay and final merged EN/RU release gates.

## Single review correction

The sole review returned two P2 sheet mismatches: earned Body was omitted on
native reopening, and town absorption omitted the accepted Antipoison timer.
Mission character initialization now folds canonical earned gains into a copied
entry member before seeding the sheet/cache. It performs no simulation
recompute. Shared town display folds the carried attachment through pure effect
arithmetic, without storing a result or advancing its timer. Spawn remains
base-only, so mission entry still attaches once. Native form and pins stay 68.

Focused potion/scroll, raised-stat, shop and envelope checks pass. The paired
EN/RU App regression `TestReleaseConsumable1090AppRestoredSheets` proves all
four permanent attributes on the first restored frame and after eight ticks,
with byte-identical opening and canonical continuation. Owned Antipoison
double-use goes 2→1, shows +50 in shop and town, retains +50/480 across native
town load, and enters the next mission at +50 once. Read-only projection tests
cover absorption and both regeneration kinds, mage/fighter gates and expiry.
The four earlier consumable App witnesses also pass both roots. Reviewer
source/overlay was not edited; no second review or broad clean-chain repeat
was performed by this lane.

The final merge release gate exposed one stale independent price oracle, not
lost loot: M81's `(8,8)` scroll retained code 3600, kind 4 and operand 6488069,
but the old oracle expected 515029 while production held 512029. The 3000
difference is its MagicItems base. `ITEM-VALUE-115` gives Scroll a separate
kind-41 sum and uses that base only when the sum is zero. The fixture now
implements that branch independently; full item identity and price equality
remain mandatory. A discriminator checks nonzero value 800 rather than
800+37+17, zero-sum fallback 37, and unchanged additive equipment valuation.
The focused producer census passes EN/RU after this test-only correction.

## Integration limits

Exact item target-cursor table values and town-covered-map release are not
published. Mission dispatch uses fog-visible actor hit versus installed point
admission. Unsupported custom effects refuse as a whole. All pre-release
cancellations refund, including started wind-up: owner-directed safety is
broader than the bounded original cancel path. Cast timing/approach use existing
simulation mechanisms, not a new ROM1 timing claim. Potion terminal lifetime
uses the existing targetability boundary.

Consumable pack taps in town wait 500 ms to distinguish double-use from staging
for sale. Pending taps are cosmetic, never saved; item identity/quantity,
member, page, tab, character mode, stock refresh and screen changes invalidate
them. Equipment trade timing is unchanged. No original-SAV writer expansion,
global spell rewrite or untouched fidelity claim is included.

Native fixture pins retain old envelopes and default fields. The combined
form-68/MissionLost/PotionEffect current envelope digest is
`4a2a76beb69651b2f8a5a6edbd96ffa3bb5c254a44ae2a06f5caacbf4a6be17d`.
Touched surfaces are item/source valuation, sim actions/effects and native
upgrade, party/character derivation, and mission/shop App input.

# Story1162 -- retained world effects continue

Original SAV LOAD previously retained AreaEffect and Prj records without ticking
them. This slice connects their established local consumers to current World,
ordinary menu SAVE and a fresh native LOAD. Knowledge stays at k6; source saves
and installs are read-only. It does not write a new original-world SAV.

## As built

- Exclusive AreaEffect roots with zero reference fields resume their saved
  counters and templates. Mode bit2 selects ring, otherwise bit1 selects cloud
  (native enum3), otherwise blast. Cloud decrements once per sim tick, pulses on
  a new positive multiple of16, and removes on the tick after zero. Pulse and
  removal use the saved radius and own layer identity; cleanup recounts occupied
  layers in both residual records and the current52-byte Cell payload. It
  recomputes cost/static/dynamic planes and current Blocks from retained
  baselines, occupants and surviving layers. Instant29 reaches these pointers.
- Ring4 uses two fixed shells; ring9 uses six oriented stages; ring21 uses32
  stages and two draws per stage from the existing native RNG policy. Stages run
  on ticks0,3,6; final stages remove immediately. Blast executes once. Bare
  supported permanent/duration/continuous Effect payloads reuse existing actor
  application/refresh/expiry. LOAD never reapplies saved modifiers or treats
  AE48[0] as casting Power.
- Complete unambiguous zero-altitude Prj rows resume signed travel/current-target
  reads, attached position snap, half/direct phase clocks,34/36 stationary ramp,
  positive-segment progress and following-tick completion. Runtime IDs bind to
  actual native entities, including materialized OriginalDeadActors whose
  current tuple differs from their immutable source tuple. Current installed
  sheets use the persisted position/phase. The bounded missing-target policy
  captures the latest point before native removal; see DIV-1115.
- Form91 adds an original-only driver footer; all prior formats migrate to
  absent/inert drivers. Old native Documents never restart historical timers.
  Stable bindings project changed area counters/stages, exclusive retirement,
  layers, projectile leaves and ordered/multiple IDs. Only bound Prj sections
  are deleted. ASCII spelling, high ID bits, unbound/defaulted items and unused
  sections remain transported unchanged. Native adoption rejects missing
  bindings, stale known fields, lost IDs and introduced area references.

The governing pinned claims are MAGIC-AREATICK-036, MAGIC-AREAPULSE-037,
MAGIC-AREACELL-039, MAGIC-MAPLAYER-040, MAGIC-RING-048, MAGIC-AREAAPPLY-038,
MAGIC-EFFMODE-009, amended MAGIC-ATTACH-016, TRIG-CELLEFFECT-045,
MAGIC-WALLBLOCK-045, TERR-STRUCT-078, ANIM-PROJ-025, ANIM-PHASECLOCK-028
and ANIM-BOLTRAMP-035. Original source
identities remain in the Document; the footer is a native binding, not an
original pointer layout.

## Proof

The registered TestReleaseWorldEffectsContinue1162 reads the Light source from
archive bytes independently:18, mode1,57 layer4 cells and kind19/mode1/E40
00100001. It observes17 then pulse16, refreshes an existing actor effect,
performs ordinary SAVE at a changed cut, removes its private source copy,
loads native in a fresh front end, then checks every next counter and removal
against the independent source oracle. It checks current Document values
separately from equal World hashes.

The same registered release also loads unchanged game0125, checks its raw
Cell payloads and saved Block overlays, runs19 simulation bodies, and makes
ordinary SAVE after retirement. It removes the private source and checks fresh
LOAD before re-projection, then executes damage1. A private controlled copy
changes the cloud to spell19/layer3/countdown1 and supplies one empty
baseline-cost8/static0 cell; after two bodies its planes25/25 become20/20 and
cost8. This is a cleanup witness, not an original wall-constructor observation.
Each case checks all57 cleared current payloads, residual records and the
direct current World/Document oracle. Four stale/loss controls per case reject
layer, count, plane and missing-cell corruption atomically. Sim controls also
preserve occupants, Sack/residue, bit4, other layers, baseline blocking, another
wall identity and out-of-radius residue through native LOAD and routing.

TestReleaseMilestone2Projectiles1157 now observes actual change rather than
freezing the source for20 ticks. Prj266 targets runtime157/archive152/MapID92,
a native corpse with ID38; the first point is19465,28086, action3/phase1 and
segments2. A moved-corpse App witness separates the current entity from source
coordinates, checks installed frames21/22/22, performs changed-cut SAVE/fresh
LOAD and checks completion. A private controlled final-stage corpse proves
removal captures the latest position before SAVE and still completes after
native LOAD. Both release witnesses pass on EN and RU.

Synthetic full App tests cover ring4, ring9 even/odd, ring21 and blast with
opaque EDD48 payloads; they prove geometry/timing/retirement, not guessed damage.
Permanent controls reject suppressed drivers/bindings, stale counters/phases,
lost target/IDs, introduced references, corrupt lengths/counts/modes and
premature/lost detachment. Separate controls cover signed division, clocks,
untimed32-bit magnitude, ground-only cloud occupancy, radius/layer isolation,
instant29 full-width spell/word duration, raw YA1 boundaries and legacy absence.
Frozen prior envelope bytes remain unchanged; only current descriptor pins and
independent outer-footer adapters change.

The registered `TestReleaseMilestone2CurrentProducers1162` joins the finite
milestone's P1-P3 proof gaps for already-existing producers. Each installed
subcase checks untouched game0021 Session, Unit/combat, Building and cell-plane
state against raw readers before setup. P1 executes two one-shot triggers:
Won/Lost become2/3, latches17/918 fire and49->48 diplomacy changes. P2 explicitly
sets a controlled current actor basis, isolates the opponent, runs damage and
scheduled regeneration, then a fighter attack with the installed weapon rider:
HP37->30, fractional bytes13->37/27->35, target50->38, skillXP slot3 0->4,
base skill0->1 and aggregateXP+4. The zero-mana fighter endpoint is setup, not a
claim that the original mage changes class. P3 first runs the untouched source's
natural20-tick movement, then positions the attacker for Building4's physical
blow1000->985. It compares the full current block scan and52-byte cell nodes.
Each subcase directly compares current World owners to Document, makes an
ordinary menu SAVE, removes its private source and checks a fresh source-free
LOAD before re-projection, then advances again. Twenty-three stale/loss
controls must be caught by those direct oracles; they do not claim every stale
field is also refused by native-envelope validation. This is proof of existing
producers, with no production changes or extension of the effect scope.

## Remaining disposition

DIV-939 and DIV-944 now name the remaining boundaries instead of calling all
local consumers Unknown. Known unfinished producers include EDD48 application,
unsupported ordinary kinds/modes, structure payloads, mixed fresh/restored
layer conflicts, picture13 tile notification, attached target/slot callbacks,
picture51 quake and36 link/target-local presentation. Their local clocks and
known shapes already proceed where bound; no missing callback is claimed done.

Actual Unknowns remain AE48[0] Power attribution, EDD48 field-to-damage bridge,
complete SpellTransport/PE44 scheduling and shared reference lifetime, exact
original direction/altitude arithmetic and post-removal target lifetime. The
existing native RNG and DIV-1092 current actor-motion projection boundary are
unchanged. Newly created/replaced actor attachments retain DIV-1107's explicit
current-Document limitation. Unsupported payloads stay unarmed and expose
WorldEffects.Unavailable; equal hashes never certify those retained raw rows.

The lane hands off a clean pushed candidate after reconciliation and focused
installed, gofmt and asset receipts. The seat owns the single fresh review,
then the final full Go, paired release, milestone2, landing and43-tool rebuild.

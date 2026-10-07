# 0139 — tasks

**Kinds:** every task below is `impl`.
**Legend:** `Covers:` are the upstream ids the slice answers. `Fences:` bound it. `Done when:` is
the whole exit condition.

---

## T1 — `impl` — the attachment becomes an id-shaped pair (`pkg/data`)

**Files:** `pkg/data/itemparse.go`, `pkg/data/weapon.go`, `pkg/data/hero.go`,
`pkg/data/foldweapon.go`, `pkg/data/unitdef.go`, and their `_test.go` siblings.

**Covers:** FR-1, FR-1a, FR-1b (the parse half only — this task resolves no id), FR-13; D-1, D-2,
D-3; AC-2, AC-3, AC-15.

`Weapon` and `Combat` carry the **token** (as written, underscores intact) and the level; the
token-to-id lookup is T3's and does not appear here. `takeCastSpell(name) (token string, level
int32, ok bool)` sits beside `takePrefix` and is the only place the attachment's grammar is stated.

**Fences:** do not change `ResolveWeapon`'s signature, its refusals, or any number it already
returns for a name — a name with an attachment and the same name without one must agree field for
field apart from the new pair. Do not touch `pkg/data/recompute.go`'s armour fold. Do not add a
`Spells` argument anywhere.

**Done when:** a weapon resolved from a name with a well-formed attachment carries the token and the
level; every malformed shape named in FR-1a carries neither and still resolves; `FoldWeapon` assigns
both on the ranged arm as well as the melee one; `UnitDef.Combat` emits them; each is witnessed by a
test that fails when the assignment is reverted.

---

## T2 — `impl` — the attack cycle can cast (`pkg/sim`)

**Files:** `pkg/sim/combat.go`, `pkg/sim/spell.go`, `pkg/sim/world.go`, `pkg/sim/rearm.go`,
`pkg/sim/binary.go`, and their `_test.go` siblings.

**Covers:** FR-2, FR-2a, FR-2b, FR-3, FR-3a, FR-3b, FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, FR-10,
FR-11, FR-12, FR-15; D-4 through D-10; AC-4 to AC-12, AC-14.

`Entity` gains `WeaponSpell uint16` and `WeaponSpellLevel int32`; `CombatBlock` and `SetCombat` gain
the pair. The record is 187 bytes wide today and the pair goes at its end. **Take `formatVersion`
41 — 39 and 40 are allocated to other lanes.** The two version tests read the constant and are
already version-free by name: do not spell a number into a test name.

**Fences:** `inReach` and the exported `InReach` keep their signature and meaning. Do not touch
`payExperience`, `resolveBlow`'s tail past the health subtraction, or `SkillXP`. `castSpell`'s eight
refusals keep their order and their draw count exactly — a replay must be unchanged for a world with
no spell-carrying weapon in it.

**Done when:** every pinned digest that moved has moved because the record widened and no other
reason; a caster with a spell-carrying weapon never enters `resolveBlow`; and each of FR-3a's two
directions, FR-9's two halves and FR-10's five refusals has a test that fails when its own line is
reverted.

---

## T3 — `impl` — the wiring and a door to walk through it (`pkg/mapload`, `pkg/game`, `cmd`)

**Files:** `pkg/mapload/spell.go`, `pkg/mapload/start.go`, `pkg/mapload/fromalm.go`,
`pkg/mapload/spawn.go`, `pkg/game/hero.go`, `pkg/game/world.go`, `cmd/missionrun/main.go`, and
their `_test.go` siblings.

**Covers:** FR-1b (the lookup half), FR-13, FR-14; D-11, D-13, D-14; AC-1, AC-13; SC-1.

The token-to-id lookup lives here, where the `Spells` collection is. Every actor-minting site that
already copies a combat block's fields copies the new pair beside them.

**Fences:** `pkg/game/inventory.go` and `pkg/mapload`'s container work belong to other lanes — do
not open them. Keep the edit to `rearm` to the weapon choice and the two added fields; change
nothing else in it. Do not add a high-tier mage literal.

**Done when:** every `castSpell=` token in both installs under
`<seat>/gameversions/{en,ru}` resolves to a row, measured by a test or a tool run
and not asserted; `missionrun -mage -mission N -attack A:B` drives a generated caster; and a
generated mage's staff reaches his entity carrying a nonzero spell after the first tick, not only
before it.

---

## T4 — `impl` — a release trains the staff's own school (`pkg/sim`)

**Files:** `pkg/sim/spell.go`, `pkg/sim/weaponspell_test.go`.

**Covers:** FR-16; D-4 (amended), D-15; AC-16.

The award goes at the **end of `releaseWeaponSpell`**, past every refusal, naming the released row's
own `School` with the victim as source entity. Reuse the sink and the amount the commanded cast
beside it already uses; write no second formula and no second refusal.

**Fences:** do not touch the byte form, `formatVersion`, `Entity`, `CombatBlock` or any pinned
digest — this task adds no state, and needing one means the shape was misread. Do not move the
commanded cast's own award, change `applySpellDamage`'s signature, or edit the sink.

**Done when:** an applied release raises the school the staff's spell names by one and leaves every
other slot — including the one a blow would credit — untouched; a refused release raises nothing;
and both halves fail when the call is deleted or its named slot is changed.

---

## Traceability

| FR | AC | Task |
|---|---|---|
| FR-1, FR-1a | AC-2, AC-3, AC-15 | T1 |
| FR-1b | AC-1 | T1 (grammar), T3 (lookup) |
| FR-2, FR-2a, FR-2b | AC-4, AC-5, AC-6 | T2 |
| FR-3, FR-3a | AC-4, AC-10 | T2 |
| FR-4, FR-8 | AC-4, AC-12 | T2 |
| FR-5, FR-12 | AC-7 | T2 |
| FR-6, FR-7 | AC-8 | T2 |
| FR-9 | AC-9 | T2 |
| FR-10 | AC-11 | T2 |
| FR-3b | AC-11a | T2 |
| FR-11 | AC-8 | T2 |
| FR-13 | AC-13 | T1, T3 |
| FR-14 | AC-13 | T3 |
| FR-15 | AC-14 | T2 |
| FR-16 | AC-16 | T4 |
| SC-1 | — | T3 |
| SC-2 | AC-5 | T2 |

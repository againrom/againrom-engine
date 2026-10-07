# Skill bonus above 100

## Intent and authority

Owner direction: item and spell skill bonuses lift a skill above 100 in the
base game. Training stays capped at 100. The skill-cap mod raises only the
training cap. The SAV stays loadable by the original: it writes the current
level clamped to 100, with base and experience unchanged, and the engine
derives the effective level on load with no new field.

The original clamps slots 1 to 5 to 0..100 twice in one recompute
(`HERO-SKILL-009`, `HERO-ORDER-014`). The original corpus holds level 100 for
base 100 with a +10 modifier. The difference is DIV-2217, DIV-2218, DIV-2219.

Owner case: Danath in `exp-skillcap-danat/quick-save-3.sav` has Sword base 100,
modifier +10 in the modifier block, and level 100 in the ordinary level block.
The bonus source is the worn Armor (code 0xEC7E, slot 12) whose Effect object
has kind 27 (Sword), mode 0, operand 10. The save carries a mod mark with an
empty domain.

## As built

- `pkg/rules`: `EffectiveSkillBound` (255) and `EffectiveSkill(base, bonus,
  cap)`: base held to the training cap, plus the bonus, as a signed 16-bit
  word, within [0, max(255, cap)]. Three readers bound 255: the 16-bit level
  word, the three-digit readout, and the derived damage byte (a fifth of the
  level, at most 51 against 20 at level 100).
- `pkg/data` recompute step 6a uses it. Experience awards still stop at
  `rules.SkillCap()`, compared with the trained base for every class: a source
  Human reads its base block, a native hero its levels less the worn bonus
  (`wornSkillBonus`). A hero at base 95 with a +10 item trains to 100 and
  fights at 110.
- Native heroes (class 0): `e.Skill` is the effective level. The engine tracks
  the bonus folded into it (`skillBonus`), so a rearm, a carry between
  missions and the roster join subtract the worn bonus before reading the
  trained base. Without this the bonus would be counted twice on every rearm.
- Source-backed Humans (class 2): the source block stays in the original
  domain, so SAVE writes the original's bytes. The live level, to-hit and
  damage base are lifted on LOAD (`liftEffectiveSkills`).
- SAVE: `projectOriginalAttack` takes the above-100 terms out of to-hit and
  damage; `clampOriginalLevels` writes slots 1 to 5 at most 100. A save
  without a mod set gains no `AgainromMods` leaf. Under a mod set the mark
  keeps the true levels (DIV-1801) and the ordinary words are clamped as well.
- Display: the character panel prints the effective level with `%d`; 110 and
  the three-digit width fit in EN and RU (screenshots in the review folder).
  The town school shows the trained base.

## Proof

- `pkg/sim`: repeated non-raising awards keep a lifted Human's to-hit and
  damage base fixed; a raise after them lands on the right values; a native hero
  at base 95 with +10 trains to 100 and reads 110.
- `pkg/rules` table of `EffectiveSkill`; `pkg/data` recompute above 100 and at
  the bound; `pkg/sim` lift tests.
- `TestReleaseSkillBonusLiftsPast100AndTheSaveStaysOriginal` (EN and RU):
  Sword 100 plus +10 gives 110 live and in the panel row, SAVE writes level
  100, base 100, modifier 10 and no mod mark, a cold LOAD shows 110, the next
  ticks keep it and a second SAVE writes the same fields. The no-bonus control
  stays 100.
- The existing skill-cap mod release witnesses pass unchanged.

## Open debt

- The effective bound 255 is an engine choice. No original observation exists
  above 100.
- The town school and the town roster show the trained base; whether the
  original shows the bonus there is Unknown.
- `TrainedSkills` recovers a native hero's base by subtracting the bonus from
  the bounded effective level, so it is wrong at the 255 bound and at the zero
  floor (base 100 with +200 recovers 55).
- The spellbook range byte written on SAVE uses the lifted level; it differs
  from the original only for a mage with Mind below about 30.
- No producer of a spell skill bonus exists besides worn items; the spell part
  has no witness.
- `wornSkillBonus` picks the fighter or mage effect family by mana, while the
  session uses the profile's fighter flag; they differ only for a fighter with
  mana.
- A spell bonus above 100 is covered by the same recompute path; the release
  witness uses a worn item only.

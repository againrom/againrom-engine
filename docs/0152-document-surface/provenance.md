# Provenance

## Research pin

The returned research source is commit
`ae9dae83688145a6244e8c0ac177175bbd5c5386`. It contains
`PARTY-MONEY-024` and the amended `SESS-START-034` used for the fresh purse,
alongside the other published claims listed below.

## Research-derived behaviour

| Spec anchors | Published basis | Confidence and boundary |
|---|---|---|
| FR-1, FR-2 | `SAV-HERO-059`, `HERO-NAME-079`, `HERO-TYPED-080`, `TEXT-NAMEIN-024`, `PARTY-FLAG-003`, `PARTY-CULL-004`, `PARTY-MERC-007` | High for primary-character identity, typed names, and the player-character distinction. The source does not establish restoration of town progress. |
| FR-3 | `PARTY-ADDHERO-017`, `REG-SCN-098`, `PARTY-INSTALL-012` | High for mission-30 activation, NPC22, companion classification, construction source, and persisted companion state. The English name Reniesta is install-localized; a Russian installation's lawful NPC22 display name is not a distinct companion. In-memory campaign continuity is Medium. |
| FR-4, FR-5 | `PARTY-MONEY-016`, `PARTY-MONEY-024`, `ITEM-PICK-009`, `ITEM-SACK-010`, `ITEM-SPAWN-026`, `TRIG-MONEY-028`, `HERO-KILL-027`, `SESS-START-034`, `MISSION-MONEY-022` | High for participant-owned purse, the fresh 100 value, sack pickup, instant 23, death-gold inputs, and the normal payment-completion path. The shipped mission-20 payment is 0; it does not establish the authored +500. |
| FR-6 | `REG-SCN-064`, `DLG-WIN-001`, `DLG-NPCTAG-018`, `DLG-NPCTAG-019`, `REG-NPC-088`, `REG-NPC-089`, `REG-NPC-091`, `SHOP-TRAY-024` | High for dialogue addressing, per-part speaker selection, portrait source, and the original five-place tray. |
| FR-7 | `HERO-FIGURE-059`, `HERO-APPEAR-050` | High for the mage figure-program ordering and layer availability. |
| FR-8 | `MAGIC-STAFF-022`, `MAGIC-AUTOCAST-020`, `MAGIC-AUTOCAST-021`, `MAGIC-SPELLHOP-023`, `MAGIC-DMG-005`, `MAGIC-ACTKEY-080` | The live weapon-spell path establishes the powered damage interval, raw release range and Stone Curse's pre-target duration that staff presentation reports. It does not establish a physical staff interval as live damage. Prismatic Spray's displayed ray count is owner-authored, not inferred from these claims. |

## Owner-authored behaviour

| Spec anchors | Decision |
|---|---|
| FR-1, FR-2 | Selection is presentation-only. It transfers no appearance, equipment, carried items, experience, skills, weight, or other simulation state. |
| FR-2 | Every selected unit has its own nonblank doll-box picture. Every selected party member, including a temporary ally or mercenary, has its own pack and worn-equipment state; only `StartingHero` presents campaign gold and documents. A non-party enemy receives no party-inventory authority. |
| FR-4 | Only `StartingHero` presents the shared purse and Quest Documents. A companion pickup routes those campaign values to that surface while retaining ordinary picked-up items on the companion. |
| FR-5 | The current fresh campaign begins with a participant purse of 100. On transition after mission 20, an authored configurable reward seam adds 500 to the participant purse. This diverges from the shipped scenario payment of 0 and is stated in reader units. The shop remains read-only; scrolling is an authored extension beyond the original five-place tray. |
| FR-6 | Dialogue is an overlay and absorbs underlying control input. The Mission Complete latch swallows only the same click's release. |
| FR-8 | A staff tooltip states `Magic` and useful spell-specific characteristics. Stone Curse states exact pre-target seconds; Prismatic Spray states powered damage, `Rays = min(power/20 + 2, 7)`, and range. Neither exposes level/power or raw leading-`#` physical/value lines. A kind-41 enchanted shop staff uses the same composition without a duplicate raw effect line. The headless snapshot carries the same worn-item tooltip lines. Runtime Prismatic target selection remains `DIV-081`; this presentation rule does not claim to settle it. |

## Open evidence boundaries

| Scope | Limit |
|---|---|
| Original save | The decoded boundary restores party identity, party loadout, purse, positions, and partial town entry. Completed missions, consumed offers, and available gates remain unrestored. |
| Lawful data and GUI | Synthetic tests are not evidence for the preserved GOG Humans data, save `666`, or a visible desktop interaction. Those observations require the acceptance checklist in `verification.md`. |

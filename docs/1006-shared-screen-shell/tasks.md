# 1006 — shared screen shell — implementation tasks

## Dependency gate

Story 1005 landed before finalization. The branch merged implementation master `7fc09de`, retained
all interactive-doll behaviours in `docs/1005-interactive-doll/contract.md`, and advanced the
research pin to the story-boundary master `fcdb02e`. The post-merge focused tests are recorded in
`closure.md`.

## Dependency-ordered tasks

### T1 — establish the shell contract in tests

- Add synthetic composition and hit-routing fixtures for the left region, distinct upper-right
  extension points, x=464 overlap, lower character region and modal layer.
- Witness region capture across press, drag and release, wheel-under-pointer routing, missing-art
  fallback and disabled-control behaviour.
- Keep shell state in Client. Do not add a Sim Core field or command.

### T2 — build the shared shell and character region

- Add the reusable shell surface and screen-owned extension interfaces.
- Move lower-column composition and party selection onto one shared character component.
- Add previous/next member and doll/statistics controls with visible enabled and selected states.
- Make previous/next available in every town room. Use the full character panel for statistics.
  Defaulting a newly entered room to doll mode is acceptable; cross-room mode persistence is optional.
- Reuse `PanelSubject`, `RenderCharacterPanel`, `composeUnitFigure` and `memberFigure`. Remove any
  screen-local stat or doll derivation that becomes redundant.
- Carry the 1005 slot mask, drag state and suppressed figure with the resolved subject.
- Prove that member selection atomically rebinds every screen-owned subject consumer.

### T3 — implement live tavern hiring

- Add the Town & Economy state and data adapters for mercenary type, pool, mission list, permanent
  unlocks, price and immediate hired state.
- Implement bottom-cell selection and double-click whole-squad hire or return. Apply purse, party and
  stock changes immediately. Add no screen-level count limit on available squads.
- Route Hire through the same selected-cell toggle. Show its current price. Add repeatable generic
  Talk and preserve repeatable dialogue NPC conversations in the same bottom area.
- Add committed members through the existing party and hero construction seams. Do not duplicate
  stat, equipment, figure or container derivation.
- Compose the tavern on the shell with the mercenary roster and existing NPC mission conversations.
- Show the current purse with EXIT. EXIT only leaves and performs no deferred commit.
- Add save/load coverage for committed purse and party changes, plus missing-data fallbacks.

### T4 — implement live skill purchase

- Add one Town & Economy purchase seam over the existing Party, Items & Heroes skill mutation.
- Implement click-to-select followed by Train. Show the selected price on Train. Disable Train when
  the selected purchase is invalid or unaffordable.
- Keep slots 0..5 in the underlying mutation seam, fixed price witnesses, purse guard, experience
  update, total update and derived-stat refresh. Do not expose General on the school screen.
- Compose fighter and mage panels from their own art, masks, icons and enable flags. Keep the shared
  geometry. Read the visible order from detailed character generation through one named mapping.
  Reconcile the later `TOWN-GENERAL-106` mapping against the owner's required order.
- Bind purchases to the shared selected hero. Preserve school mission dialogue and modal routing.
- Show the current purse with EXIT and prove it buys no selected skill.
- Add save/load coverage for skill, experience, derived values and purse.

### T5 — migrate the shop after the 1005 merge

- Move 0157 shop composition into the shell without changing trade state or arithmetic.
- Preserve the x=464 upper overlap, distinct shelf hit and draw rectangles, all grid controls,
  wheel routing, hover, dialogue layering, member-bound pack and missing-art fallback.
- Replace the shop-local lower-right painter with the shared character region.
- Add one Book toggle independent of doll/statistics. Replace the trade table with the selected
  member's learned spells while Book is enabled, and restore the unchanged table when disabled. Do
  not add an Inventory toggle.
- Rebind doll, statistics, pack, Book, usability and item targets on member change. Keep the trade
  table unchanged so it supports cross-member transfers.
- Keep the visible fourth-button EXIT action as clear-table-and-leave.
- Route every merged 1005 shop interaction through the shared slot mask, drag machine, suppression,
  wear rule and transaction paths. Do not fork any of them.
- Re-run all 0157 and 1005 focused tests without weakening expectations.

### T6 — migrate detailed character generation

- Leave pre-create outside the shell and unchanged. Preserve its four hero pictures, three difficulty
  pictures, encoded ten-byte name input, Back and Forward.
- Preserve detailed skill selection, four-stat point-buy, pool, preview and final participant-name
  validation. Put Accept, Reset and Back in the upper-right controls.
- Render only the preview doll in the right region through the existing subject/figure derivation.
  Put the full character panel in the symmetric left region.
- Do not add member arrows, doll/statistics, Book or Inventory controls to detailed generation.
- Keep the failed-validation destination Unknown. Do not choose one without new authority.
- Record the owner-authored shell composition and unresolved rejection destination for divergence
  reconciliation.

### T7 — close cross-screen state and shipped content

- Add a headless real-campaign integration witness covering tavern hire, school purchase, shop trade
  and unchanged pre-create followed by migrated detailed chargen through the production front-end
  seams.
- Add a production save/load witness for committed hire and training.
- Sweep the EN and RU shipped roots separately for every new art and text lookup and for the full
  route in `spec.md` section 7. Use `AGAINROM_ASSETS` in the owner's invocation form.
- Verify town square, gates, world map, town dialogues, mission offers and loaded-town entry remain
  unchanged.
- Confirm no Sim Core source, command, digest or serialized format version moved. Stop and reclassify
  the story if one did.

### T8 — canonicalize and close

- Update `spec.md` to the behaviour actually built.
- Write `closure.md` with all twelve aspect verdicts, the integration witness, EN/RU shipped-content
  evidence and research reconciliation.
- Reconcile `DIV-015`, `DIV-016`, `DIV-017`, `DIV-018` and `DIV-020`; add only the expected typed
  rows that the as-built Unknowns and chargen deviation require.
- Run `gofmt`, build, vet, `go test -trimpath -count=1 ./...`,
  `scripts/check-no-game-assets.sh` and `scripts/check-no-game-assets.sh --history` on the clean
  commit.
- Build `builds/1006-shared-screen-shell/` and rebuild `builds/current/`. Run every README command
  before recording it.
- Push the implementation commits for adversarial review. A known in-scope gap fails closure.

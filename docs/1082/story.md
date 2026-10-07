# Campaign difficulty

## Result and contract

Character creation chooses Easy, Normal or Hard through the three shipped
Levels pictures. A new generator defaults to Normal. Forward, Back and the
detailed stat Reset retain the selection; leaving the generator discards it.
Only a successfully opened new game commits its draft and resets the previous
campaign. The keyboard focus list appends the three difficulty controls after
Forward; Up/Down selects focus and Enter activates. Pointer press previews,
matching release selects, and dragging away cancels.

The campaign passes its difficulty to every mission start, including the next
mission and town return. Native map and town saves preserve it. The additive
gob zero from old saves defaults to Normal; other values outside 1..3 are
refused before replacing the current session. Native restore builds its
candidate with the candidate difficulty, then replaces the world with saved
bytes without scaling them again. Original import uses Head.Difficulty.
Original-town provenance retains the original writer when difficulty is
unchanged and chooses lossless .ags if that session field changes.

## Authority and limits

`UNIT-GATE-012/013/014/033` establish default 2, values 1..3, the three controls,
and the Units-only once-at-spawn adjustment. `UNIT-PLACE-034` keeps authored
current HP after scaling. The existing arithmetic is unchanged: Easy uses
H*66/100; Hard uses H*3/2 and +50 ToHit/Defence; every Humans placement and
party member is excluded. `AI-DIFF-016` supplies no AI behaviour multiplier.
Existing `DIV-037` ghost-template treatment remains unchanged.

`TOWN-223` supplies the keyed on/l/lon art states. `TOWN-236/244` leave the
rectangle table Unknown. `DIV-532` records the owner-directed installed-art
fit; these coordinates are not claimed as decoded original coordinates.

Touched surfaces: generator model/input/composition, asset loading, campaign
mission opening, native save/restore, original import/provenance, headless
create_character. No simulation byte-form or arithmetic change. Top-level
HeadlessScenario.Difficulty remains mission-only; character.difficulty drives
the actual production UI.

## Proof

`verification.md` records the final gates and production output. The headless
game drive selects Hard through character creation and reports difficulty 3
on the map. Installed tests cover all three levels through missions 10, 20
and 30, map and town persistence, and original-import wiring. Native resume
is byte-exact. The arithmetic test computes scaling independently of Adjust;
it verifies transport, not the table decoder's separate proof.

No simulation or save-envelope version changed. The current native-envelope
fixture hash changes only because its gob descriptor adds Difficulty; its
inner world hash remains 293920d10f45d9d8. No original map-state fidelity,
ghost behaviour, AI policy, or exact original geometry is newly claimed.

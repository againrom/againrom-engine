# Tasks — a name, a picture, a corner, a font

Legend: **files** what the entry may change — a permission, not a prediction · **fences** what it
must not do · **done when** the observable it leaves behind. They land in ascending order, each
depending only on those before it.

## T1 — the name reaches the window tier

**files** MODIFY `pkg/render/terrain/units.go`, `pkg/game/units.go`, `pkg/game/world.go`,
`pkg/ui/overlay.go`; ADD or MODIFY the tests beside each.

FR-1 — DD-1, DD-2.

**fences** each of the four files gains one field or one assignment and nothing else; no existing
test is edited. Nothing converts, trims, folds, validates or defaults the text — it is carried as
the registry handed it over. The snapshot's name is read from the class the entity's own id
resolves to, at a site the corpse substitution cannot reach; the substitution itself is untouched.
No new import in any of the four packages.

**done when** AC-1, AC-2 and AC-3 hold. A fixture registry whose classes carry distinct names, one
of them with bytes at and above `0x80` written as escapes, and a class pair whose death link names a
differently named class, are all built in test code.

## T2 — the layout value and the picture

**files** ADD `pkg/ui/panel.go`, `pkg/ui/panel_test.go`.

FR-3, FR-4, FR-5 — DD-3, DD-4, DD-6, DD-11.

**fences** no viewer state is read or written and no method is added to the viewer here; the file
imports nothing the package does not already import, and nothing from the engine. The pen and the
box come from the font's own two answers — no width, height or advance is summed a second way, and
no glyph is indexed here. One switch turns a subject into a field's text; a second site that knows
which fields exist is the defect this entry exists not to create.

**done when** AC-6 through AC-10 hold, plus P-1, P-3 and P-4 as properties over generated names —
empty, ASCII, and holding bytes at and above `0x80` — healths including zero, negative and a zero
maximum, and cells at and past the map's edges. Mutants: the minimum width applied as a ceiling; a
skipped row still consuming its gap; the value's advance taken from the box instead of the pen.

## T3 — the panel on screen

**files** MODIFY `pkg/ui/viewer.go`, `pkg/ui/panel.go`, `pkg/ui/panel_test.go`; ADD
`pkg/ui/panel_draw_test.go`.

FR-2, FR-6, FR-8 — DD-5, DD-7, DD-8, DD-10.

**fences** the subject is taken from the package's existing present-and-not-dead filter by calling
it; no second filter, no copy of its predicate and no stored subject. The pass slice, its order and
every existing draw path are untouched — the presentation is appended after them, and the one
engine call is the only line of it not reachable from a test. The refresh key holds stated values,
never an entity.

**done when** AC-11, AC-12, AC-4, AC-5 and AC-14 hold, plus P-5 over two viewers differing only in
the layout. Mutants: the key's health field dropped; the area size dropped from the key; the texture
kept while the picture is rebuilt.

## T4 — the font reaches the game

**files** MODIFY `pkg/game/frontend.go`, `pkg/game/frontend_test.go`, `cmd/againrom/main_test.go`.

FR-7 — DD-9.

**fences** the load goes through the container filesystem already open, opens no handle of its own
and adds no flag anywhere on the path. The viewer's own tolerance of a missing font is not
weakened — no constructor or setter gains a requirement — and the standalone developer viewer gains
no font and no startup condition. Nothing under `cmd/` changes but its install fixture.

**done when** AC-13 holds against a synthetic container in each of its three shapes, and a
front-end assembled without one still yields a viewer that draws and advances.

## Traceability

| Task | FR | DD | AC |
|---|---|---|---|
| T1 | FR-1 | DD-1, DD-2 | AC-1, AC-2, AC-3 |
| T2 | FR-3, FR-4, FR-5 | DD-3, DD-4, DD-6, DD-11 | AC-6, AC-7, AC-8, AC-9, AC-10 |
| T3 | FR-2, FR-6, FR-8 | DD-5, DD-7, DD-8, DD-10 | AC-4, AC-5, AC-11, AC-12, AC-14 |
| T4 | FR-7 | DD-9 | AC-13 |

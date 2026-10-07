# Tasks — 0110

`FR`/`AC`/`P`/`DD` → `spec.md`; `D-x`, `R-x`, `SC-x` → `plan.md`. Four implementation tasks, in
order: each leaves the tree building and green on its own.

| Task | FRs | ACs | Ps | DDs | D / R | SCs |
|---|---|---|---|---|---|---|
| T1 | FR-1, FR-2, FR-3 | AC-1, AC-2 | P-1, P-2 | — | D-1, D-2 | — |
| T2 | FR-4, FR-5 | AC-4, AC-5 | P-5 | DD-1, DD-7 | D-3, D-4, D-5, R-1, R-3 | — |
| T3 | FR-6, FR-7, FR-8, FR-11 | AC-6, AC-7, AC-8, AC-12 | P-2, P-3, P-4 | DD-3, DD-4, DD-5 | D-9, D-10, D-12, D-13, R-5 | — |
| T4 | FR-7, FR-8, FR-10 | AC-9 | P-4 | DD-2, DD-6, DD-7 | D-6, D-7, D-8, D-11, R-2, R-4, R-6 | SC-5 |

AC-3 and AC-10 are measurements against an installed root, and SC-1, SC-2, SC-3, SC-4 and
SC-6 are the evidence stage's own steps. No task asserts them and no task reads an install.

## T1 — the code and the three addresses

**Kind:** implementation. **Boundary:** the value and its names. Nothing reads a slot, a weapon or
an archive.

**Files:** `pkg/data/itemcode.go` `ADD`, `pkg/data/itemcode_test.go` `ADD`.

**Covers:** FR-1, FR-2, FR-3, AC-1, AC-2, P-1, P-2, D-1, D-2.

**Fences.** Do not touch `pkg/formats/alm` — its own class and index readers stay as they are, and
this type does not import it or wrap it. Do not add a `String` on the code that returns the
seven-digit name; the namer is a named function, so a reader can tell which of the two forms it got.
Do not decode, open or list anything.

**Done when:** every one of the 65 536 codes answers four fields and a seven-digit name, both name
forms are exercised at their boundary values, and the three addresses are built by three functions
over one namer.

## T2 — the slot carries a code, and so does a resolved weapon

**Kind:** implementation. **Boundary:** the accessor rename and the composed code. The party path's
behaviour does not change; only the two `pkg/game` tests that name the old accessor are re-spelled,
because the rename does not compile without them.

**Files:** `pkg/data/equip.go` `MODIFY`, `pkg/data/equip_test.go` `MODIFY`,
`pkg/data/weapon.go` `MODIFY`, `pkg/data/weapon_test.go` `MODIFY`,
`pkg/game/hero_test.go` `MODIFY`, `pkg/game/heroappear_test.go` `MODIFY`.

**Covers:** FR-4, FR-5, AC-4, AC-5, P-5, DD-1, DD-7, D-3, D-4, D-5, R-1, R-3.

**Fences.** `HeroBodyFor` keeps its signature, its totality and its refusals; only what it reads the
row *out of* changes. Do not add a second constructor for a code beside the one inside the
resolution. Do not widen `data.Weapon` with the shape or material index. In the two `pkg/game` tests
change the accessor spelling and nothing else — no assertion moves. Nothing in `pkg/sim` is opened,
and `pkg/game/hero.go` belongs to T4.

**Done when:** a slot round-trips a code, the body derived for each shipped weapon literal is the
name it was before, and `data.Weapon` states a code whose field D is its own row.

## T3 — the window

**Kind:** implementation. **Boundary:** the drawing tier. It receives pictures; it opens nothing and
decodes nothing.

**Files:** `pkg/ui/inventory.go` `ADD`, `pkg/ui/inventory_test.go` `ADD`, `pkg/ui/app.go` `MODIFY`.

**Covers:** FR-6, FR-7, FR-8, FR-11, AC-6, AC-7, AC-8, AC-12, P-2, P-3, P-4, DD-3, DD-4, DD-5, D-9,
D-10, D-12, D-13, R-5.

**Fences.** Do not import `pkg/data`, `pkg/formats/*` or any archive type here, and do not give the
subject type a field that is an archive address rather than a picture — a nil picture is how an
unread address arrives. Do not extend `MapEntity`. Do not let the binding reach the command path,
the selection or the camera.

**Done when:** the window is closed at mission open, the binding toggles it only with one assembled
character selected, cancel closes it, a subject with a nil picture in a cell still composes, and the
package's own sources name no archive type.

## T4 — the figure, built once, from the archive

**Kind:** implementation. **Boundary:** the wiring tier. It reads the archive, paints the figure and
hands the finished subject over.

**Files:** `pkg/game/inventory.go` `ADD`, `pkg/game/inventory_test.go` `ADD`,
`pkg/game/hero.go` `MODIFY`, `pkg/game/world.go` `MODIFY`, `pkg/game/frontend.go` `MODIFY`,
`pkg/game/hero_test.go` `MODIFY`.

**Covers:** FR-7, FR-8, FR-10, AC-9, P-4, DD-2, DD-6, DD-7, D-6, D-7, D-8, D-11, R-2, R-4,
R-6, SC-5.

**Fences.** The builder takes an entry source and returns the subject plus the addresses it could not
read; it writes to no stream and holds no viewer. Build the subject **once**, where the mission opens
— not per tick and not per frame. Do not add a field to any `pkg/sim` type, do not open
`pkg/sim/binary.go`, and do not spend a byte-form version: this story adds no encoded field. Tests
build their archive from synthetic bytes and read no install.

**Done when:** the figure is one picture carrying the base and each occupied slot's layer, an
address the source does not hold appears in the returned list and leaves the rest of the picture
whole, the front end prints that list once at mission open, and a world of fixed entities encodes
and hashes exactly as it did before.

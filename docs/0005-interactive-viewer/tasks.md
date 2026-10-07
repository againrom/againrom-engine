# Tasks — 0005 interactive terrain map viewer

| Task | Scope | Discharges |
|---|---|---|
| **T1** | `pkg/render/camera` — the pure camera model: state, `Clamp`, `Pan`, `ZoomAbout`, `VisibleTiles`, `ScreenToWorld`/`WorldToScreen`. Register the package in `internal/archtest`. Table tests + a randomized pan/zoom invariant test. No Ebitengine, no `formats` import. | FR-3, FR-5 · AC-1…AC-4 · P-1, P-2 |
| **T2** | `pkg/ui` — the Ebitengine viewer: `ebiten.Game` implementation, input→camera intent (arrows/WASD, edge-scroll, wheel zoom, Esc), visible-cell draw through `terrain.Resolve` + the sub-cell→`*ebiten.Image` cache, placeholder fill for absent slots. Adds the Ebitengine dependency (`go.mod`/`go.sum`), its row in `THIRD_PARTY_NOTICES.md` and license text under `LICENSES/`, widens `externalAllowed` in `internal/archtest` to permit the engine for `pkg/ui` + cmd tier only, and registers `pkg/ui`'s new edges. | FR-2, FR-4 |
| **T3** | `cmd/mapview` — the standalone viewer CLI: `-assets`/`AGAINROM_ASSETS`, `-map`, `-graphics`, `-check`; loads the archive + map + tileset, reports load errors before any window opens, runs the viewer otherwise. Register in `internal/archtest`. Tests for the headless `-check` path over synthetic inputs. | FR-1, FR-5 · AC-5 |

Then, untagged: `verification.md` recording the gate results, AC coverage, and AC-6 as pending manual
verification.

## Ordering note

T1 lands first and is entirely self-contained (stdlib only), so the camera invariants are proven before
any engine code exists. T2 introduces the only third-party runtime dependency in the project so far and
therefore carries the notices/licence bookkeeping in the same commit that makes `go.mod` require it —
keeping `go mod tidy` stable at every commit. T3 wires them together.

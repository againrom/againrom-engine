# Story1167 verification

Base: `96637ad41b8fdaecca50152b650039a27c593ac5`. Knowledge remains
`84f328362e6c946303da3d15dddfcefbcfec4946`.

The observable result is town SAVE with mapped quick spells producing SAV,
followed by current bindings and a working next F8 cast after fresh LOAD.
The prior writer refused those changed/generated bindings into AGS. The
standalone saveconvert command now also converts those current bindings.

Branch evidence is outside Git under `review/story1167/` in the seat:

- `focused-go.log`: all tests in `pkg/formats/sav`, `pkg/game`,
  `cmd/saveconvert` and `internal/gatedtests` passed with no install environment.
- `focused-en.log` and `focused-ru.log`: seven registered release families
  passed per locale, including both new city origins, quick-key cast, two
  custom-ID fallback controls and three city conversion families.
- `old-imported-red.log` and `old-generated-red.log`: reverting either
  production exporter to the base reproduces its inappropriate AGS fallback.
- `wrong-identity-red.log`: deliberately writing IDs instead of controller
  indices fails the independent raw-word oracle with `[23,18,-1,16]` against
  `[5,23,-1,7]`. These are expected runtime failures, not compile failures.
- `div.log`: all 418 live rows parsed; 117 advisory retraction matches remain.
  The two touched rows cite the active281 and promoted286 claims.
- `alloc-before.log` and `alloc-after.log`: 35 allocator answers, none missing.

The seat owns the sole fresh review and the final full gate chain after any
correction; neither is claimed as completed by this branch.

The installed missionrun probes (`census-{base,edit}-{en,ru}-m{10,20}.log`)
report zero UNSUPPORTED nodes for both missions on both locales. The prior
`builds/current/missionrun.exe` was verified as base96637ad, unmodified.
The branch executable gives the same 0/0 result. The seat's
`pipeline/milestone-baseline.txt` also carries no unsupported nodes; the
16/27/12 and 14/15/11 script populations are unchanged. This story changes
town persistence and leaves that measured script population unchanged.

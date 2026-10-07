# Story 1093 — read-only trigger inspection

## Intent

The owner can inspect authored map triggers in mapedit: condition comparisons,
ordered actions, typed parameters and confirmed map references. Inspection does
not compile or execute the script and never writes the map or a save.

## Scope

- Add Triggers in the existing eighth filter slot. Retain every decoded raw
  trigger, including empty condition/action slots, repeated references and
  records the runtime compiler would omit.
- Show the authored name, raw fields, all three comparison pairs and four
  action slots. Resolve descriptions and references only where pinned claims
  support their meaning. Unknown, missing and ambiguous targets remain visible.
- Show confirmed map targets and focus them without replacing the selected
  trigger or losing its detail-scroll position. Do not substitute roster IDs,
  array indices, external hero ordinals or runtime actor identities.
- Preserve ordinary object inspection, failed-open transactions, complete-map
  Fit, fractional scrolling and useful inspection at 640x480 on EN and RU.
- No editing, script execution preview, live enabled/disabled state or original
  editor fidelity claim. Keep the existing research pin unless new evidence is
  actually required.

## Design

Retain the native map and fixed 320px rail. A second graph canvas would obscure
the spatial result and leave too little room for clauses at 640px; the existing
eighth filter slot and scrollable inspector carry this slice instead.

The signature is a selected trigger whose condition/action targets are visible
on the map while its authored clauses remain in the inspector. Use the existing
shipped font1 for restrained headings, body and raw values; new fonts would
break the established install-specific text handling. Preserve byte-aware RU
wrapping and the separate host-text encoding seam.

Palette roles: ink #201B16, panel #30291F, parchment #D6C49B, action gold
#B8954F, condition blue #78B6C2, warning #B85E46. Color identifies the selected
target's role, not an invented runtime state. Keep labels short and raw IDs
available. Use generic display/reference DTOs in UI; ALM interpretation stays
in the game adapter, outside rendering and simulation.

## As built

`pkg/game/mapinspectiontriggers.go` reads decoded raw section 7, never the
runtime compiler. The summary shows three comparison pairs joined by AND, four
ordered action slots, raw Once and its meaning. First-left-zero triggers stay
visible with an explicit runtime-builder omission note. Each referenced node
identity expands once; repeated slots remain in the summary. Standalone action
and condition records remain in All/Other, including unreferenced records.

Descriptions are independent of authored labels. Arguments keep Par indices,
numeric type tags and all ten raw values/tags, including tag-zero values.
Unknown operations, tags, comparators, missing and duplicate identities are
explicit. No duplicate identity picks a first match and no reference narrows
its uint32 value to a placed-record ID width.

Group references resolve through Unit.GroupID, placed units through UnitID,
structures through Field12. Player references name one-based roster slots.
Hero ordinals, external static units and items receive no invented placement.
Adjacent typed X/Y slots supply authored cell markers; exact check-3 signatures
also expose their distance square on byte-coordinate maps. Wide coordinate
values or map axes above 256 retain the authored point but explicitly withhold
the semantic region; the inspector does not emulate byte-coordinate aliases.
Check-2 endpoint rules remain Unknown. Map targets use bounded clipped
primitives. Focus bounds every projected footprint and anchor-cell corner,
not just two diagonal samples of a cell-space union. UI owns generic DTOs and a
shared wrapped-line draw/hit layout; link focus keeps the trigger, filter and
both scroll positions. Failed opens remain transactional.

The read-only CLI accepts `-triggers` and `-focus-reference N`; selected
`-check` output lists structured details and reference indices. No new decoder,
map writer, simulation, dependency or font was added.

## Authority and limits

Pin ba21c9aa is unchanged. The raw grammar is `ALM-TRIG-044` through
`ALM-TRIG-047`; comparisons, Once and omissions are `TRIG-CMP-006`,
`TRIG-FIRE-007`, `TRIG-BIND-010`. Reference identity and placement use
`TRIG-REC-011`, `ALM-UNIT-048`, `ALM-OWN-039`, `ALM-PLACE-033` and
`TRIG-PARAM-030`. The bounded vocabulary uses `TRIG-COND-003`, `TRIG-ACT-004`,
`TRIG-SACK-022`, `TRIG-DIST-014`, `TRIG-ADDITEM-027`, `TRIG-GIVEALL-025`,
`TRIG-MONEY-028` and `TRIG-DROP-013`; no corrected fallback/randomness or
notification-tail clause is used. Authored labels are metadata (`TRIG-CAT-026`).

DIV-626 records the owner-directed UI, not original-editor fidelity. Parameter
name buffers, reserved node words and opaque trigger bytes remain preserved in
SourceBytes but are not decoded into labels. The operation-caption vocabulary
is deliberately bounded; an unlisted opcode is shown as Unknown with its raw
number, even when another gameplay consumer supports it. No live state, script
execution, original-editor equivalence or native GPU witness is claimed.

The sole review returned 93d9990d for projected group focus and wide-coordinate
region admission. One correction pass fixes both; the unchanged reviewer probes
and independent EN/RU seat acceptance pass. No second review was commissioned.

Base 734a3a7ea2f71df8f8e73fd4903862c5d401345d. Verification and the exact
observable result are in [verification.md](verification.md). The seat owns the
sole fresh review, landing, paired final release gate and `builds/current/`.

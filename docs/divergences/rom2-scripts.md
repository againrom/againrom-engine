# Divergences -- Second game roots -- scripts

1 row(s). Part of the split described in `docs/DIVERGENCES.md`: read that file first for what a row means, what each column holds, and how a row is found.

## Divergences

Implementation and researched ROM1 behaviour differ, or the owner ruled against a claim.

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|

## Authored where research is silent

No promoted claim establishes the readers or meaning covered below.

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-5097 | ROM2 scripts / action 1 session counter at +0xc710 | Owner: do not invent a meaning for session counter +0xc710 | `R2-ENGINE-1450` (High): action 1 adds 1 to a session counter at +0xc710 and to register 902. `R2-SESSION-019` (High): the serialized session block is 10,374 bytes (registers, latches, raw spans, diplomacy, counters), which excludes +0xc710. No claim in `knowledge/` names a reader or meaning for +0xc710 | Action 1 (`ScriptInstantCounter`) adds 1 to register 902; the World neither keeps nor saves the separate session counter at +0xc710 | UNKNOWN | No published claim names any reader, effect or persistence for session +0xc710 | A claim identifying a reader or gameplay effect of session +0xc710 | OPEN |

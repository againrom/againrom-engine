# Divergences -- Second game roots -- actor names

1 row(s). Part of the split described in `docs/DIVERGENCES.md`.

## Authored where research is silent

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-4550 | ROM2 actor names / own-name precedence for campaign hires and necromancers | Owner: show the installed type label when no authored own name is present; never expose an internal template key | `R2-ENGINE-581` (High / Medium / Unknown) establishes the unitname.txt type line and actor own strings, but leaves their writer Unknown. `R2-ASSET-145` (High) establishes the installed table convention. `R2-ENGINE-378` (Medium) identifies internal hire row keys and leaves displayed names Unknown | The shared word producer retains every loaded unitname.txt line. Ordinary actors use the type label through the existing name stages. Campaign hires resolve their template key through the existing definition-name stage, retaining mod names and explicit actor-name overrides; a missing type label shows no internal key. Authored map names retain final precedence. Actor state and SAV names retain their original keys | UNKNOWN | The native writer and precedence of own names for these actors are not established. DIV-4087 retains inn-heading debt; DIV-4249 retains construction, placement and dismissal debt | A claim establishing when the original assigns an own name to a campaign hire or necromancer and when it overrides the type label, for EN and RU | OPEN |

# F2 Save and F3 Load or Diplomacy

Bind the original function keys to existing destinations without adding a
second save implementation. AI-KEY-125's keyboard table rows 12-15 names F2
Save and F3 Load on phase-2 map/town surfaces, and F3 Diplomacy on other map
phases. MENU-ITEM-011 and MENU-ITEM-012 describe the existing menu surfaces;
SESS-INPUT-037 gives focused-child precedence.

The production GameMenuContext.Campaign projection selects phase 2 for maps;
town takes the campaign route. F3 must open the existing Load window even
when the store is empty: the menu row's directory-presence disable is not a
keyboard gate. Existing generated save names, acknowledgement/error messages,
Load origin, and Diplomacy contents remain under DIV-099.

Physical keys use the client's press edges. Modal notices, town dialogue,
other screens, and unfocused input cannot trigger a second destination.
A navigation key consumes same-frame side input. Long Load idle followed by
Escape and immediate menu Escape/Return must not repay simulation or ambient
time. F1 Help remains absent; F9/F10/F11 diagnostics stay unchanged.

The shared action seam now serves both validated menu rows and function keys.
F2 keeps the existing generated-name save and acknowledgement; F3 enters Load
through the same held game-menu origin. Standalone F3 enters the existing
Diplomacy page. Escape from the returned menu consumes its held interval even
when no neutral menu frame intervenes. No save format or simulation state changes.

Independent tests state the key/action/context table without querying menuRows,
parse physical bindings, and exercise App.step with save spies and an
elapsed-time-aware halt fixture. EN/RU witnesses pass through production
SaveSeams with temporary stores. The executable scenario saves, cancels Load,
returns, and restores the saved character on each install. See verification.md.

DIV-294 remains open for absent F1 Help and points to DIV-526 for the retained
press-edge/repeat and simultaneous-key ordering differences. DIV-099 is unchanged.
The reserved DIV-527 is unused.

The sole review returned an earlier-held mouse gesture whose release activated
the new destination. The correction consumes that release using the continuing
physical button level, for every admitted F2/F3 route. The next fresh click
remains usable. The independent regression fails all five earlier-down routes
before the correction and passes them afterward.

Story 1074's paused cutscene work is excluded. When that branch lands later,
its early cutscene return must remain before these function-key actions.

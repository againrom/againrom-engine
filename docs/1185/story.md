# Shared original dialog frames

Load Game, game menus, settings, save prompts and conversations use one installed
green and gold frame. The minimap uses the installed green crystal. The owner
provided original screenshots and asked for a reusable component across these
surfaces. The separate structure-shroud hotfix is inherited from main.

## Behaviour

- `ui.DialogFrame` draws `lm.256` window pieces and the `t_border.256` portrait
  border. One FrontEnd cache feeds map notices, town conversations, menu pages,
  objectives, cutscenes and SAVE. Existing face crops and click geometry remain.
- LOAD displays ten rows with the original installed title and labels, scrollbar,
  OK, Delete and Cancel. Clicking a row selects it; OK or Enter loads it. Wheel,
  arrows, Home and End move the selection. Disk tokens remain exact UTF-8.
- Delete first presents confirmation. Only saves in the writable current save
  directory can be deleted. Source-tagged local SAVs and original-install SAVs
  remain distinct. A changed file invalidates confirmation. Cancel preserves it.
- The right column retains its 158-pixel height. The native 160x158 crystal
  surrounds a centered map inside a 128x128 area. Unseen terrain reveals the
  crystal texture. Camera outline, marks and hit testing share the content area.
- Installed-font game menus now expose the same CPU panel uploaded by Draw, so
  the headless town-menu image includes the actual dialog and dimming.

## Proof

Focused UI tests cover scrolling, selection, exact LOAD tokens, confirmation,
cancel, minimap content hits and bevel rejection. File tests cover original
protection, local SAV identity, changed-file refusal and confirmed deletion.
The complete UI and game packages pass. `TestReleaseSharedDialogFrames1185`
passes independently on EN and RU resources. Local rendered EN/RU LOAD and
town-menu images were inspected. Final commit gates are recorded by the seat.

The sole adversarial pass found overlapping integer identities for controls and
save indexes 100..105. The correction separates target kind from row index. A
120-save pointer regression reproduces all six failures before the fix, then
proves selection alone has no load, delete or navigation side effect and OK
loads the exact selected token. The review report is held by the seat.

## Boundaries

DIV-1281 and DIV-1282 retain exact original geometry and control-policy limits.
The saved-file population is unchanged; this framing change does not manufacture
a Restart Last Mission checkpoint. All assets remain external. Native mouse
operation of the original executable is not claimed as evidence. The owner
requested local integration and deferred push.

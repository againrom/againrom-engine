# F5–F8 quick spells

## Result and contract

Ctrl+F5–F8 binds the current spell, or a hovered book cell when no spell is
current. A duplicate moves to the new slot. Plain F5–F8 selects a populated
binding: an open book selects only; a closed book also requests Cast when the
accepted selection knows that spell and has Cast capability. Empty slots do
nothing. An unavailable populated slot changes current without requesting a
mode. Targeting continues through the existing book-cast path, with item-cast
context retaining precedence.

The four bindings belong to FrontEnd/session, not an actor or Viewer. Native
SAVE and fresh LOAD preserve them atomically. Authored session policy carries
them across mission/town transitions and resets them only when a new campaign commits.
Current spell, armed mode and book visibility are transient, not new persisted
fields. Original new-campaign reset and no-load mission carry remain Unknown.

Base: master94245f177edf1f800f42424e3654419cab67dd16, reconciled by a normal
merge. Research moved forward to e30c92d13d75201169fc94fe6eb2d89f8cad537f.
Claims consumed through the reader: AI-QUICKASSIGN-278, AI-QUICKINVOKE-279,
AI-QUICKOWNER-280, AI-QUICKSAVE-281, AI-SPELLIDENT-286, AI-SPELLPOP-287,
AI-SPELLCAP-288, AI-SPELLGUARD-289, AI-SPELLITEM-290, amended AI-PANEL-061,
and HERO-APPEAR-041/042/044.

## As built

Physical F5–F8 edges and headless aliases enter the map arm after focus,
modal and cutscene guards. Current spell is separate from armed Cast; changing
the primary book owner clears transient targeting, not session slots. The fixed 24-cell
catalog uses the promoted cell-to-real-ID table (cell5 maps to real23, not6).
Unavailable cells remain visible, hoverable and bindable. Availability is the
union of accepted selected actors; producers are those knowing the chosen
spell, capped at253. The primary actor still supplies ownership permission.
Cast capability is any accepted actor with nonzero book mask and resolved
client class23/24, not server TypeID, Mage alone, mana or successful art loading.
Ordinary equipment/body-name resolution supplies that class. Hired actors keep
the existing fixed-class policy; unresolved/custom actors retain their explicit
class. Custom/incomplete spell tables keep their own IDs and row order, and the
unprojected standalone Viewer retains its legacy capability seam.

Native snapshots add four real uint16-domain IDs to the existing envelope,
without changing sim bytes or the save version. Absent old fields default
empty; malformed lengths, duplicate nonzero IDs and out-of-domain values refuse
before adoption. Unknown-to-this-install IDs survive unchanged. Original
Shortcuts imports exactly four signed cell indices: -1 empty, 0..23 translated
by the promoted table. Invalid, duplicate or malformed records refuse the whole
load rather than normalize. Unchanged populated source shortcuts survive the
existing bounded SAV path; changed live slots explicitly take unsupported→AGS.
Fresh native LOAD derives the comparison baseline from retained source
provenance, so another SAVE cannot silently revert to opaque source shortcuts.
No original SAV writer expansion or original-runtime compatibility is claimed.

The frontend-design skill was applied within the owner's existing two-row
strip: installed icons and bitmap font, fill#101218, frame#8a7446, text#f2e6c4,
selected#ffd73c and existing muted ink. A small F5–F8 cell mark is the only new
signature. Current uses the book-cell border; armed Cast uses the command
panel/cursor state. Its existing tooltip appends the installed current spell
name, never an invented name for an unknown ID. No new art, font, animation or
persistent panel. RU retains the installed English spell name beside the
localized Cast verb; the existing C hint is separate input debt.

## Proof

Focused controls cover physical/source key mapping and gates; current-before-
hover, empty and duplicate assignment, unavailable/open/closed invocation,
ownership and later-selected capability, producer filtering, item precedence,
253-member bound, all24 scaled hit targets, custom catalogs and no-art class
projection. Same server TypeID with differing live client classes, weapon
removal, NPC and hired actors distinguish the capability rule. Independent
native gob/header readers cover old/new shapes and malformed atomic refusal;
the exact published pre-quick envelope SHA93b8ddb8 is frozen as decode-only
bytes, while current encoding is pinned at279f4c59 with unchanged World bytes;
source controls include populated [5,7,22,-1]→[23,16,19,0], duplicate/range/
wire-shape refusal and malformed retained provenance. New-game failure keeps
the old slots; committed adoption clears them.

Both installed release witnesses pass on EN and RU with the owner-save corpus:
real App hover/Ctrl assignment of [16,1,6,19], open selection, unavailable
current-only state, closed targeting that spends mana, menu SAVE, fresh
FrontEnd LOAD and the next targeted cast; mission finish→town→mission20 keeps
the slots. The mage's book is controlled input, not campaign-acquisition proof.
Original-derived town SAVE is SAV-capable unchanged, becomes AGS when only
bindings change, and remains lossless after fresh LOAD and a second SAVE.

Headless: ten EN scenarios pass—1119,1080,1090,1087,1089,1035,both1005,0163 and
1013—and RU1119/0163/1013 pass:13 runs. New1119 has41 steps at640x480; v8 adds
quick-state assertions and hover/real-ID book points, rejected by v1–7. The
existing endpoint helper uses source SHA60267c82 and isolated output; its fresh
controlled mission is not an original continuation or campaign playthrough.

The original1087 fixture failed after paused book refresh realized its implicit
open book. Exact clean master94245 passed that old fixture. Camera origin stayed
X256/Y1510: the all-unavailable warrior book reserved97px and covered the hero.
Per seat direction preserving the existing owner policy,1087 explicitly closes
Book through the ordinary key before D+D; every Defend/selection/state assertion
and real click is retained.
Both master and candidate pass that explicit fixture. A release subtest proves
the paused all-unavailable book appears, closing restores the hero hit, and
camera origin, tick and hash do not change. No auto-pan or zero-union hiding was
added. Old failure/comparison traces remain under ignored review/story1119/.

Actual App Draw produced16 final frames under ignored
review/story1119/{en,ru}/fixed24-{1024x768,640x480}-{open-current,closed-armed,open-unavailable,closed-unavailable}.png.
Seat visual inspection covered16/16: no new F-key/grid/tooltip clipping or
overlap in that finite set, and current/armed roles differ. Existing bitmap
text is heavily downscaled at640; this is not an all-text readability claim.
This is visual QA, not independent state proof. Capture was offscreen, with no
OS input or original executable. Earlier unprefixed captures are pre-final
controlled fixtures, not the final catalog proof.

Final branch gates: gofmt and go test -trimpath -count=1 ./... pass (49 test
packages,25 with no tests); no-game-assets passes. The first full run found
three fixture/guard mismatches—empty-shortcut byte expectations, the new
weapon-latch reader registration and the current envelope hash—corrected before
that full passing rerun. The old envelope remains a separate decode witness.
The divergence guard scans295 live rows/447 cited IDs, reporting77 inherited
partial-retraction rows, none in826–830. Allocation sweeps before/after the
ledger edits pass32 answers/missing0, floor834 unchanged. Preserved-install
checks pass181 files in both roots after install-capable witnesses.
The release population adds two named tests (149→151); the root owns the final
paired release gate and sole fresh-context story review. No milestone census
was rerun: UI/session projection changes no simulation timing, paths or script
population.

## Retained debt

DIV826–828 disclose authored lifecycle/native/presentation choices; DIV829
retains the upstream accepted-selection lifetime/mixed-population and hired/
custom-class boundaries. DIV830 retains the decoded per-cell target-predicate
matrix: reusing today's cast path is not full original target/input fidelity.
DIV287/328 are closed only for their superseded compact-book/primary-mana
claims. DIV831–833 were unused. No original populated-slot runtime round trip,
full SAV compatibility, global health/mana non-dependence, C-key support or
broader camera/UI redesign is asserted.

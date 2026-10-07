# Exact SAV document index

Original SAV LOAD must keep sparse living actors at their saved positions and
retain session state when both terrain lists are empty. An ordinary AGS save
and a fresh LOAD must preserve that imported state and its subsequent ticks.

`Open` follows the exact shared-archive document programme, including null and
repeated Player references, world selector, counted terrain, session, Sacks,
the common 400-byte trailer and optional word alignment. Raw members cannot
introduce Players, world data or actors. Body offsets remain the editing
authority, including extended terrain counts.

Position joins use unique living Unit-derived records reached through Player
actor lists. Ambiguous saved IDs or eligible map targets fail before publishing
a mission. Missing NPCs are not created. The first non-null Player remains the
party provenance; this does not introduce a multiplayer owner-selection rule.
The index retains source fine bytes and the ALM handoff writes them, but the
canonical simulation preserves cell positions, not fine/subcell movement.

Authority: SAV-DOC-053 (amended), SAV-FULLREAD-252, SAV-ARCHREL-253,
SAV-TERRKEY-056, SAV-TOKENPOS-074, SAV-ID-015, SAV-OWNER-048 and SAV-DEATH-051
(amended), from the pinned research. Structural corpus closure is not gameplay
completeness. No unpublished experiment supplies authority.

Proof: complete literal SAV fixtures, sparse/empty and raw-decoy cases,
duplicate-join rejection, App SAV LOAD → AGS → fresh LOAD continuation, existing
city conversion, EN/RU release witnesses and relevant session scenarios.

Exclusions: original process or install writes, generated world SAV export,
absent NPC creation, new multi-human semantics, and extended/Unicode CString
support in the general reader. Extended CString prefixes fail explicitly.
`saveconvert` remains city-only. An all-body source actor index is not a
from-scratch world exporter. Legacy `ClassRecords`/`Chain` diagnostic scans
remain outside `Open` and the gameplay position join.

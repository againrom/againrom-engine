# Earned hall record

Completing the terminal campaign mission captures one score and records it in
an isolated, persistent Hall of Fame. The ending shows the score and whether
recording succeeded; a failed record has a retry control. AGS preserves both
the immutable result and its recorded marker.

## Authority and behavior

FAME-INSERT-011, FAME-WRITE-007 and FAME-STRING-010 supply signed insertion,
new-before-old ties, ten-row trimming and the name/score/tail framing. Loading
retains stored order and duplicate names. The installed table is a read-only
seed; the mutable table stays beside the initial save directory even if SAVE
later browses elsewhere. Missing seed starts empty; corrupt local data refuses
instead of overwriting it. Publication writes/syncs/closes a temporary file and
atomically publishes the complete table. Installed paths and links into them
are fenced. No drawing operation writes.

FAME-021..025 establish bounded original input paths. Time adds signed
sub-ticks/16 on each first accepted completion, with low32 wrapping. The
corpse observer counts hostile transitions from stages0/1 into2/3/4 after a
simulation step; death stage1 alone and removal do not count. Bind/LOAD
baselines existing bodies; later resurrection can admit another transition.
The main Human hero's six current experience slots supply the score scalar.
Separate binary64 operations and signed64 truncation make the result stable.
DIV-1267 records the chosen hero, slot sum and arithmetic; native cached actor
class, latest packet and full floating-point control remain Unknown.

Original campaign LOAD supplies raw A/B. Supported city SAV writes current
counters, including imported provenance; original town adoption waits until
hired-roster validation succeeds. New AGS stores explicit score history.
Old AGS without both counters remains unknown, preserves later observations
and earns no invented score. SAV refuses when it would turn that absence into
known history; AGS still works. Terminal original SAV remains refused.

## Proof

Focused tests cover signed extremes, overflow, ties, duplicate names, framing,
old-save absence, primary identity, actual simulation transitions, hash
independence, save copy ownership, atomic adoption, write failure and retry.
Frozen historical AGS bytes and simulation digests stay fixed; only current
Go descriptor controls change for the additive DTOs.

The two EN/RU release witnesses cover installed mission150 script Victory,
local row publication, repeated navigation, cold AGS, and source-free/imported
city SAV counter round trips. Earlier prerequisite victories and score totals
are controlled; this is not an ordinary campaign playthrough. Screenshots and
raw generated SAV/AGS files remain outside the repository.

Sole review and final merge gates are recorded by the seat. Final evidence must
include full Go, asset/divergence guards, the paired registered release gate,
original-resume milestone2 gate, and exact packaged executable continuation.
Physical window input is blocked by the app-specific Computer Use denial.

## Remaining debt

DIV-1266..1268 preserve authored storage/timing, source/number and observation
policies. Native final packet/class/FPU and original ending lifetime remain
unproved. A low-ranked score may be trimmed immediately. Replaying an older
pending checkpoint can earn another same-name row; no historical identity
ledger or global cross-process writer lock is invented.

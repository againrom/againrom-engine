# Windows launch and player instructions

## Result and authority

Owner direction: a launcher beside the binary selects a lawful asset folder,
retains current stdout/stderr diagnostics and exit status, and explains the
current player save workflow. No new ROM1 behaviour or divergence is introduced.

`Play-Againrom.cmd` runs `launch-againrom.ps1` without a PowerShell profile and
keeps the console open after success or failure. `-NoPause` first disables the
final key wait. Assets are explicit or prompted; a blank answer cancels.
The launcher validates the five required archive names, chooses package-local
saves and logs, and never changes an install or shell profile. An unavailable
log directory falls back to a named local temporary log. If neither works, the
game does not start. Local output paths reject installation ancestors, network
drives and directory reparse points.

One fresh UTF-8 log contains launch paths, executable SHA256, the game's raw
combined stdout/stderr and its exit status. Child command paths travel through
environment values with delayed expansion disabled. Logs have unique names;
normal exit and launch failure retain earlier runs. The command returns the
child's status, or a launcher failure/cancellation status before it starts.

`PLAYER-GUIDE.md` is the package guide. Its controls and save descriptions come
from current `pkg/ui` and `pkg/game` production routes. README corrects normal
menu startup and distinguishes ordinary mission AGS/recovered-city SAV from
the separate converter's bounded source-backed mission export.

## Proof

`scripts/test-windows-launcher.ps1 -EvidenceDirectory <new-local-directory>`
passes 26 controlled launcher invocations on Windows PowerShell 5.1. The proof
builds console and GUI-subsystem Go fixtures and checks both normal and nonzero
exit statuses, a simulated final key press, 2,048 lines from each native stream,
UTF-8 text, exact argument arrays, working directory and retained earlier logs.
Paths include spaces, Unicode, percent signs, exclamation marks, ampersands,
parentheses and brackets. Explicit assets override an unrelated environment
root; prompted selection and blank cancellation are exercised separately.

The sole review returned terminal directory separators corrupting native argv
at both the CMD-to-PowerShell and PowerShell-to-game boundaries. The correction
passes CMD path parameters in its child environment and quotes terminal native
backslashes in pairs. Eight launcher probes inspect the actual executable's
argv for prompted and explicit `\`/`/` endings, ordinary and Unicode paths, and
explicit log-directory endings. Both drive-root entry routes reach the output
fence unchanged. Two direct fixture calls through the production start builder
retain drive-root arguments; a third deliberately unescaped argument control
produces the expected corrupt argv. No drive alias, junction or root-level
archive is created. Redirected console input and output both use UTF-8.

Missing assets, missing archives, absent/invalid executables, missing launcher
script, blocked preferred/fallback logs and both selected/other installation
output fences retain a log or console diagnostic. The five synthetic archive
files remain byte-identical, with no added entries. Evidence stays under the
explicit output directory and is excluded from Git. `.gitignore` also excludes
package-local `logs/`.

No original process or real interactive game was launched by this proof.

## Open debt

The seat owns the sole adversarial pass, final combined gates, packaging and
the actual packaged game witness. Fixture launches do not prove graphics,
audio, original-runtime SAV interoperability or the final RC package.

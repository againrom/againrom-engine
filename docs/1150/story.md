# Clickable fountains and levers

One selected player character can use a visible fountain or lever through the
ordinary town cursor. The command approaches, faces and applies one effect.
Lever state feeds existing script word checks; fountains spend one charge on
the installed big healing/mana potion amount, including at a full stat.
Charges, state and unfinished use survive native SAVE/LOAD.

Authority: AI-CURSOR-226/231, AI-CLICK-050, AI-STRUCTUSE-306 and
UNIT-STRUCTUSE-090/091 in reviewed public knowledge k5. UNIT-STRUCTZERO-080
bounds the separate recharge callback. The owner requests these interactions.

The implementation targets the clicked identity, uses bounded approach routing
and existing turn progression, and schedules recharge on the represented
session counter's multiple60. Exact original neighbor selection and callback
timing remain Unknown. Only installed kinds15/16/28/29 gain use; town buildings,
save points and unidentified arches are separate surfaces. No arch effect is
inferred from appearance. Form89 carries interaction metadata and pending use;
older saves restore immutable metadata from their map or source roster.

Focused simulation tests cover charge bounds, source-derived pools and atomic
refusal, lever-to-script check21, approach/facing, cancellation by attack/move/
manual teleport, unreachable retirement and cold continuation under both native
and saved Group AI. Historical87/88 byte and hash fixtures remain pinned; form89
has independent malformed-footer and predecessor controls.

The sole review returned two defects, corrected together. An admissible use
suppresses the predecessor's final book/scroll release tick; failed refund
admission preserves the old action. Completing use synchronizes the saved
Group destination to the stopping cell. Independent regressions include cold
replacement and 500 idle ticks after completion under both Group forms.

TestReleaseStructureUse1150 drives the real App pointer against both fountains
on mission90 and both lever classes on missions91/101. Installed EN results:
HP10 to110 or mana20 to120 with3 charges becoming2; levers1 to0 change art.
SaveStore during approach and a fresh front end continue to identical hashes.
The release gate repeats the same witness on RU. Actor position/population is
controlled; this is an installed-engine witness, not original gameplay evidence.

DIV-285/289 close the absent cursor/command seams. DIV-284 keeps other classes
out of the implemented capabilities. DIV-1030 records interaction integration
and DIV-1031 recharge placement;1032..1035 retire unused. Final gate, review,
landing and executable witness records belong to the seat journal.

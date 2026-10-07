# Hired actor identity

## Intent

SAV retains the identity of a hired Human through city saves, cancellation and
rehire, and entry into the next mission. The ordinary actor fields carry the
hire type and exact definition independently of the Againrom supplement.

## Authority

The owner chain in `review/owner-hired-duplicates` contains an engine city save
with seven hired Humans whose U148 is zero and whose Name is a technical
template. The original cancellation and rehire keeps those seven actors and
adds seven actors with U148 12 or 13 and empty Name. This observation establishes
the anonymous original representation for these two hired squads. It does not
establish a rule that every named mercenary must lose its name.

The pinned claim review covers SAV-ACTORDISPLAY-545, MERC-TYPE-001,
MERC-HIRE-003, SAV-678 and SAV-797. Token18 remains a publication mask.

## Scope

The city and mission producers write the effective hire byte. The import path
recognizes anonymous hired Humans from their ordinary identity. The exact
Humans definition remains independent of the ordinary display name. Existing
health, inventory, modifiers and member order remain current state.

Old files with the exact technical template name may use an explicitly bounded
compatibility path. An anonymous NPC definition with U148 zero is not a hire.
Mixed files already containing duplicate actors retain every actor; the engine
does not choose which actor's progress the owner intended to discard.

## Proof

Focused RED covers native and source-backed city U148, Name, definition row,
TypeID, publication mask and the three/four actor counts. Further proof covers
two city SAV cycles, ordinary-field import without the supplement, cancellation
and rehire, next-mission saves, and the supplied mixed original resave.

Required release witnesses run on both lawful EN and RU installs. Original
executable continuation of the corrected files remains Unknown until witnessed.

## As built

The hired member stores its definition row independently of Name. Generated
city and mission records write the hire type to U148 and leave the exact
technical template name empty. Explicit custom names remain. Source-backed
actors project the current backing value on every SAVE. A legacy player hire
with zero backing and the exact matching template name is normalized on LOAD;
unrelated NPC actors do not take that compatibility path. City residue retains
unmodelled high bits. Siege type policy remains separate from Human identity.

The supplied hired, resavedhired and resavedhired150 inputs pass two SAV cycles
on EN and RU. The already duplicated resaves retain all fourteen hired actors.
Ordinary city records load without the private supplement. Cancellation and
rehire do not add actors. New mission continuations match World hashes for four
ticks after each cold LOAD. Custom names, exact definition rows, mixed siege
order and upper backing bits have separate controls. The final focused batch
and source guards pass in 90.633 seconds with zero remaining processes.

The sole review returned one source-backed legacy mission route: current
manifest restore could replace the repaired name with the retained technical
name. The correction normalizes the restored manifest and projects current
hire names after manifest capture. A regression retains the current supplement
while restoring the old zero-byte/technical-name shape. A separate control
covers absent manifests, explicit custom names and a non-hired NPC.

DIV-1445 records the bounded legacy repair and original acceptance gap. Final
release and milestone gates remain required after the correction.

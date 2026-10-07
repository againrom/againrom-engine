# Tasks — 0055 selection count

No `plan.md`: the contract is small enough that a design stage between it and these two entries
could only restate it.

## T1 — the count becomes a field of the panel

FR-1, FR-3, FR-4, FR-5.

The panel's field set is closed and its members are named in one place; the count joins it there, and
the one function that turns a subject into a field's text gains the arm that formats it and reports
it absent below two. Nothing else may branch on this field.

The number rides on the value that carries the described unit's stated values, because that value is
what the redraw key compares. Say in the type's own comment why a statement about the selection lives
on a value named for one unit — a reader who does not know that reason will move it.

The authored layout gains one row for it, ahead of the three that exist, with a label of our own
choosing in the register the existing labels use. Update the comments that say the name is the only
field that can be absent: that is now false.

Tests: the field's text and its absence at 0 and 1; the byte-identity of AC-2, built by composing
against the authored layout with the count row filtered out of its rows rather than against a
hand-copied one; the four-unit picture; and a placed-row layout that puts the count at its own
offset. Reuse the file's existing font and ink-layout fixtures.

## T2 — the count reaches the screen

FR-2, FR-6, FR-7.

The one place that builds the panel's subject from the viewer already computes the filtered set it
takes the described unit from; the count is that set's size, and it must be read from that same call
rather than from a second walk of the selection — a second walk is a second chance to disagree about
which units are present.

Nothing else should need to change: if the redraw key is a comparison of the value T1 extended, the
redraw requirement is already met, and the task is to prove that rather than to add to it.

Tests, against the viewer: a mixed selection of dead, absent and alive ids; a death mid-run driving a
rebuild and a new number; an unchanged frame driving neither; empty selection and fontless viewer
drawing nothing. Reuse the file's existing viewer and entity fixtures.

## Traceability

| Task | Requirements |
|---|---|
| T1 | FR-1, FR-3, FR-4, FR-5 |
| T2 | FR-2, FR-6, FR-7 |

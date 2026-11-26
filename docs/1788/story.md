# The Save page is one more list page built from data

## Result

The Save page looks and plays as before in both games: the same layout, keys, name entry, overwrite and delete
questions, and the same save file written. Inside, it is no longer a page of its own. It is the same list page as
Load, Characters and Maps, and what makes it a save page is a few plain values: the name field, the question
before an overwrite, the button row and the action on accept.

## Authority

- Owner: the architecture audit item 2, one builder per kind; the list page is a kind and Save goes through it.

## Rules

1. The Save screen is stepped, composed and drawn by the same functions as the Load screen.
2. No list page body asks whether the page is the save page; the save role is data on the page's role.
3. Typing a name, Enter on an existing target, the overwrite and delete questions and Cancel act as before.
4. The request the page sends to the save writer is the one the name, folder and format name; the writer is unchanged.
5. A save written from the page is the same file as before the change.

## Code

| kind | its one builder | change |
|---|---|---|
| list page open | `pkg/ui/save.go` openListPage | Save opens through it with the save role |
| list page step | `pkg/ui/load_window.go` stepPage | one step for Load and Save |
| list page compose | `pkg/ui/load_draw.go` composePage | one paint; the save paint is gone |
| list page draw | `pkg/ui/app.go` drawList | takes no screen argument |
| screen handlers | `pkg/ui/screenhandler.go` listPageHandler | builds both handlers |
| save role | `pkg/ui/listrole.go` listRole | six hooks became data |
| save writer | `pkg/game` save producer | untouched |

## Proof

| test | rules |
|---|---|
| `TestSaveScreenSharesTheListPageHandlers` | 1 |
| `TestSaveScreenHasNoBodyOfItsOwn` | 1 |
| `TestListPageRolesAreRows` | 2 |
| `TestSavePageEntryOverwriteAndCancelUseTheSharedPage` | 3, 4 |
| `TestSaveDialogConfirmsExactTargetAndDefaultsToBack` | 3 |
| `TestSaveDialogLoadStyleHasLiteralRowsAndBar` | 3 |
| `TestSaveDialogPrunesOnlyWhenTheFileIsWritten` | 5 |

Rule 5 holds because no file of `pkg/game` or `pkg/formats` changed; the `pkg/game` save tests pass.
Release tests at the release: none.

## Open debt

- The unreadable-save count is dropped by the save scan and shown on neither Load nor Save; Characters shows it.
- The save page state keeps an unused edit field beside the shared edit state.
- A Delete key with no deletable row now falls through, and a press and release in one frame are handled in turn;
  neither is visible to the player.

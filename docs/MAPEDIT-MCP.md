# Map editor MCP

The map editor can accept MCP tool calls while its window remains visible.
MCP is off unless `-mcp` is supplied. Each tool runs through the editor's input
doors on its window loop. The result follows the frame that applied it. Edits
use the same undo history as local input. Besides the window's keys, pointer
and text, typed edit and read tools place records, paint, write triggers and
read what an edit needs (see Typed edit tools).

## Connect to an open window

Start the editor with HTTP enabled. Use the installed executable and an asset
root from a lawful installation. These commands are for the owner to run:

```powershell
& 'C:\Games\Againrom\mapedit.exe' -assets 'C:\Games\ROM' -mcp http -mcp-address 127.0.0.1:8765 -mcp-pace 300ms
```

The status strip shows the URL. A connection report in the window shows that
URL, the bearer token and the path of the MCP record, with a note row when the
start made a new token or could not reuse the port. The selected player
profile contains one record, `mapedit-mcp.json`, with `url`, `token` and `port`
fields. The profile is resolved by the game's existing profile rules, including
`AGAINROM_PROFILE`. An executable inside an installation uses its selected
private `Againrom` profile, with the existing user configuration fallback.
Explicit profile paths keep the install write fence.

Replace `TOKEN` with that window's token:

```powershell
claude mcp add --transport http againrom-editor http://127.0.0.1:8765/mcp --header "Authorization: Bearer TOKEN"
```

The token and the port persist. Every start reads `mapedit-mcp.json` and reuses
its token, so a client configured once keeps working after the editor restarts.
The default address is `127.0.0.1:0`: the editor first tries the port the record
holds and takes a free port only when that port is busy, for example because a
second editor is running. The note row and the diagnostic line then say so, and
the record keeps the new port. An address with a port, such as
`127.0.0.1:8765`, is used as given and fails when the port is busy.

To rotate the token, start the editor once with `-mcp-new-token`. It makes a new
token, records it and reports it; later starts reuse the new one. A record that
is missing, unreadable or malformed is replaced by a new one in the same way,
with a diagnostic line, and never stops the editor from starting.

The record remains after shutdown; it does not imply that its process is still
running. It is written to a temporary file in the profile and renamed over the
old record, so an interrupted start leaves the old record or the new one. The
file mode is `0600` on systems supporting those permission bits. Windows uses
the profile directory's access control.

HTTP accepts authenticated POST requests at `/mcp`. GET and DELETE return 405.
The listener binds only to `127.0.0.1`. Requests with an Origin must name the
exact `http://127.0.0.1:port` origin. Native clients can omit Origin. Host must
match that same address and port. There is no OAuth or browser login flow.

## Let the client start the window

Stdio uses no network listener. All stdout is MCP JSON; diagnostics use stderr.

```powershell
claude mcp add --transport stdio againrom-editor -- 'C:\Games\Againrom\mapedit.exe' -assets 'C:\Games\ROM' -mcp stdio -mcp-pace 300ms
```

The client starts the executable and its window when it connects. Keep the
window open while using its tools. Closing the window closes its request queue.

With stdio the client owns the editor process. When the client session ends,
because the user leaves the client, the client restarts or reconnects the
server, or the client crashes, the client closes the window's standard input
and may then terminate the process. A terminated editor closes without the
unsaved-changes prompt, so edits made since the last save, the person's and the
agent's alike, are lost. Save before ending the client session. For a window
the owner keeps, use the HTTP route above: the window is started by the person,
and ending a client leaves it open. When the editor sees its standard input end
before it is terminated, the status strip shows "MCP client disconnected (stdio
input ended)"; the editor cannot stop a termination that follows.

## Desktop client configuration

A desktop MCP client with local stdio support can use this JSON configuration:

```json
{
  "mcpServers": {
    "againrom-editor": {
      "command": "C:\\Games\\Againrom\\mapedit.exe",
      "args": ["-assets", "C:\\Games\\ROM", "-mcp", "stdio", "-mcp-pace", "300ms"]
    }
  }
}
```

A desktop client with Streamable HTTP and static header support can instead
attach to an already open window:

```json
{
  "mcpServers": {
    "againrom-editor": {
      "type": "http",
      "url": "http://127.0.0.1:8765/mcp",
      "headers": {"Authorization": "Bearer TOKEN"}
    }
  }
}
```

HTTP configuration keys depend on the client. The stdio configuration is the
portable choice for clients without native HTTP support.

## Tool table

Arguments are JSON objects. Unknown fields, null fields and wrong types are
refused. Tool results are text content containing JSON, except `frame`, which
returns PNG image content. Refusals use `isError: true`; ordinary window prompts
remain in the state and require the same confirmation as local input.

| Tool | Arguments | Window route and result |
| --- | --- | --- |
| `actions` | `{}` | Every named `editorKeyRows` row: name, label, localized help, and `heard`. The flag describes the window input gate; individual actions retain their own context checks. |
| `action_<name>` | `{}`; optional `verbose` | One generated tool for every named key row; calls `Key(name)`. |
| `key` | `{"name":"unit"}`; optional `verbose` | Calls `Key` with a name from `actions`. Optional `ctrl` and `shift` use the window's native text-navigation policy, for example `{"name":"left","ctrl":true}`. On a map with unsaved changes and nothing open to close, `escape` opens the unsaved-changes question of leaving the editor, as the window's Esc does. |
| `text` | `{"text":"value"}`; optional `verbose` | Calls `TextInput` on the focused field. |
| `pointer` | `{"action":"click","x":400,"y":300}`; optional `verbose` | Calls `Pointer` at window pixels. |
| `cell` | `{"x":10,"y":12}` | Calls `ShowCell`, then `CellPoint`; returns window `x,y`. |
| `state` | `{}` | Status, message, panel, popup, prompt, Help/report, dropdown, groups, selection, object fields, dirty flag, path and window size. |
| `frame` | `{}` | Reads the editor's drawn window after Draw, including the agent pointer. It does not call the diagnostic snapshot composer. |
| `map` | `{"x":10,"y":12}`; optional `verbose` | Without `verbose`: `width`, `height`, `playable` (the cells inside the border ring, `x0,y0..x1,y1`), `cell` (the cell's terrain, height, objects, overlay, passability and border flag; a missing plane is null or marked unknown) and `record_counts` (the count of each record group). It lists no players, units, structures or loot. With `"verbose":true` it lists them, with the other records (`records`) and the script nodes and triggers (`script`), instead of `record_counts`; the `records` tool lists the records meeting a cell range. |
| `open` | `{"path":"C:\\Maps\\map.alm"}`; optional `discard`, `verbose` | Uses the window's typed Open prompt. Unsaved changes still ask before replacing the document; with `"discard":true` the question is answered as its Open button does and the changes are dropped. |
| `save` | `{}`; optional `verbose` | Uses the window's Save key route, including in-place and mod destinations, validation and prompts. A map with no file of its own yet is refused, naming `save_as`; the Save prompt the route opened is closed again. |
| `save_as` | `{"path":"C:\\Maps\\copy.alm"}`; optional `verbose` | Uses the window's typed Save As prompt. Install fences, overwrite checks and validation remain active. The map's own file is refused: `save` writes it. A folder that does not exist is named as missing; a save creates no folder. |

### Short results

Each tool answers its own result. A window tool (`key`, `text`, `pointer`,
`open`, `action_<name>`) answers `last_call`, `status`, `message`, `mode`,
`dirty`, `path`, `saved`, `prompt`, `prompt_open`, `prompt_text`, `popup`,
`dropdown`, `report_open`, `report_note` and `selected`. `save` and
`save_as` answer `last_call`, `saved` (the file the call wrote, empty when it
wrote none), `path`, `dirty`, `status`, `message`, `prompt`, `prompt_open`,
`report_open`, `report_note` and `report_asks`. With `"verbose":true` each
answers the whole window state; `state` always does.

A save of a map the check finds errors in is not refused: the window asks
whether to save with errors, and the result has `report_open` and
`report_asks` true and no `isError`. `key {"name":"enter"}` answers yes and
writes the file, `key {"name":"escape"}` writes nothing. A save that wrote
its file answers no error.

The place that chooses this is `everyToolOwnResult` in
`pkg/ui/mapeditormcp.go`.

`pointer.action` accepts `move`, `press`, `release`, `click`, `double_click`,
`drag` and `wheel`. `button` is `left` (default), `right` or `middle`.
`shift`, `ctrl` and `alt` are optional booleans. `wheel` is the wheel amount,
from -1000 to 1000.
`drag` requires `to_x` and `to_y`. Coordinates must be inside the window.
Press/release levels persist between calls. A held agent button owns the
pointer only until the person presses a button, moves the mouse, scrolls the
wheel or pans with the arrow keys in the window, or until the stdio client's
input ends or the window closes its queue. The editor then releases the button
through its own pointer release at the agent's last point, so a drag ends as a
person's release would end it, and the person's input goes through. An HTTP
client that leaves is not seen by the window, so a button it left held ends by
the person's input. Clicks and drags release their button automatically. Drags include eight intermediate points, each shown in a frame.

The agent cursor is a light cross at the last supplied pointer point. The
status strip shows the last call, such as `MCP: key unit`. `-mcp-pace` leaves a
pause between completed tool calls; multi-frame gestures still progress with
the window's frames. Local input is available between calls, except for the
pointer while an agent button is held (see above). While a gesture is being
applied, its frames own the input route.

## Typed edit tools

Each typed edit tool is one row of the editor command table
(`pkg/ui/mapeditorcommands.go`). The window's route for the same edit calls
the same row's apply method, so a typed call writes the bytes the window
writes, takes one step of the same Undo history, and shows a host refusal on
the window's status line. `tools/list` gives each tool's input schema; extra,
null, missing or mistyped arguments are refused with `isError` and change
nothing. A typed call is refused while a window prompt, path question or
popup field is open; finish or cancel it with `key` first. The refusal names
what is open: the system file dialog, the prompt by its title (`key enter`
answers it, `key escape` closes it), or the popup field being typed. A
`save_as` or `open` call that is refused closes the path prompt it opened, so
the next call is not refused by it.

An edit answers a JSON report: `changed`, `record` (the record the call
names, as `{"Kind","Index"}`), `script_id` (when that record is a unit or
structure: the number a script parameter names it by), `dirty`, `can_undo`,
`can_redo`, `message` (the status line), `prompt_open`, `cells` (up to 1024
changed cells as `[x,y]`) with `cells_total`, `cells_box` (the changed
cells' bounding cells `[x0,y0,x1,y1]`, corners included), `planes` (the
count of changed cells per plane: `terrain`, `passability`, `height`,
`objects`), `records` (up to 256 changed records) with `records_total`, and
`warnings`: the map check's findings (see `check`) on the records the edit
changed or names, each with its `rule`, `error`, `text`, `record` and
`cell`. `paint` and `randomize` list `cells` only with `"verbose":true`;
without it they answer the count, the box and the planes. A height `paint`
also answers `height_change`: `cells` (the cells the brush covered),
`changed` (those whose height changed), `box` (the covered cells' bounding
cells), and `min` and `max` (the smallest and largest change, after minus
before); with `"verbose":true`, `heights` lists up to 1024 covered cells as
`[x,y,before,after]`. A drop zone edit answers `drop_zone`: `zone` (its
number from 0), `cell`, `zones` (the count after the edit) and `removed`.
An edit of several records at once
(`move_records`, `delete_records`, `copy_records`, `move_area`) also
answers `selected` (the records the selection holds after it), `note` (the
selection's status text) and `refusals` (a reason per record the host
refused). A message edit answers `page`, the message page it wrote. The
report compares what the map
stores before and after the call: each cell's tile word, its passability
bit, height and object byte, and each record's stored bytes, so a field
change names its record. A call that changes nothing reports `changed`
false and no cells or records. `new_map` names `width`, `height` and
`playable` instead. A number field that stores 0 as "inherit" refuses a typed 0 with
the field prompt's words; a field entry without `value` is the prompt's
empty text and inherits.

| Tool | Arguments | Window route |
| --- | --- | --- |
| `new_map` | `width`, `height`; optional `altitude` (0..126, default 64), `multiplayer`, `discard` | The catalogue's New question. Sizes are multiples of 8 from 32 to 256. Unsaved changes still ask; `"discard":true` drops them as the question's Open button does. |
| `paint` | `plane` (`terrain`, `height`, `objects`, `passability`), `cells` (`[[x,y],...]`, the stroke in order); optional `value`, `radius` (default 0), `shape` (`circle`, `square`), `fill` (`pattern`, `random`, `variant`) and `variant` for terrain; `set`, `flat`, `mode` (`step`, `level`, `smooth`) for height; `forest` (`{"kinds":[...],"share":25,"thin":false}`) for objects; `verbose` | One brush stroke. `value` is a terrain class or object code from `brushes`, a height change (-31..31, not 0), the height with `set`, or 1 to close and 0 to open a cell. With `forest` the stroke is the object brush's Forest stroke and `value` is left out: it plants the object classes `kinds` over `share` per cent (1..100, default 25) of the open cells it covers, or with `thin` removes that share of those classes. A forest stroke draws from the window's stroke seed, so two strokes over one map plant different cells. A height stroke works as Height brush below states. |
| `add_record` | `kind`; optional `x`, `y` for a record on a cell; optional `values` by step key; `parts` for a `Message` | The Add window, step by step with its defaults and checks. `add_steps` lists the steps and choices. A unit's steps are class, tier, owner, group and cell; a unit, structure or drop zone is refused on a cell of the border ring. Kinds: `Enchantment`, `Unit`, `Structure`, `Sack`, `Player`, `Trigger`, `Action`, `Condition`, `Message`. A `Message` takes `parts` alone (see `write_message`) and adds a new message page and an action that shows it, as the Add window's New message and the message editor's Apply do. |
| `set_field` | `kind`, `index`, `key`; `value` (number) or `text` | The field prompt (F). `record` lists the keys. |
| `set_fields` | `kind`, `index`, `fields` (a list of `{"key","value"}` for a number or `{"key","text"}` for a text) | A script record popup's Apply: one edit. |
| `set_object` | `kind`, `index`; optional `fields`, `cell` (`[x,y]`) | The object popup's Apply: fields and a move as one edit. |
| `move_record` | `kind`, `index`, `x`, `y` | A drag on the map. A unit or structure is refused on a cell of the border ring. |
| `clone_record` | `kind`, `index`, `x`, `y` | Clone (C) and a click. Units, structures and sacks; a unit or structure is refused on a cell of the border ring. |
| `delete_record` | `kind`, `index` | Del. |
| `set_drop_cell` | `x`, `y` | The drop tool (D) on a map without drop zones. A cell of the border ring is refused. On a map that keeps drop zones it is refused, naming `add_drop_zone`, `move_drop_zone` and `delete_drop_zone`. |
| `add_drop_zone`, `move_drop_zone`, `delete_drop_zone` | `x`, `y`; `zone` (from 0) | The drop tool's click, drag and Del. A drop zone is a drop action of the map's script, so the report's `records` names an `Action`; its `drop_zone` names the zone. A cell of the border ring is refused. |
| `write_trigger` | `name`, `actions`; optional `conditions`, `once` (default true), `index` (default -1 adds a trigger) | The trigger wizard's Apply. Each row is `{"item", "params"}`, a condition also `comparison` when its comparison is chosen; `trigger_items` lists the items. A parameter is an integer, `[x,y]` for a cell or `[x1,y1,x2,y2]` for an area. A unit or group parameter is one of the choices `trigger_items` lists, or a unit of the hero band 10001..11000; any other value is refused. One action row of a call whose item shows a message page (`a.dialog`) may carry `parts`, as `write_message` takes them, and leave the message parameter out: the trigger and the new page its action shows are one edit and one Undo step, as the wizard's message editor writes them. The place that allows this is `writeTriggerDialogue` in `pkg/ui/mapeditorcommands.go`. |
| `move_records` | `records` (`[{"kind","index"},...]`), `dx`, `dy` | A drag of a selection: units, structures and sacks move together as one edit. A record that would leave the map, enter the border ring (a unit or structure) or enter a closed cell is not moved: the report's `refusals` names it, as the window's drag does, and the other records move in the same step. |
| `delete_records` | `records` | Del on a selection. |
| `copy_records` | `records` or `area` (`[x0,y0,x1,y1]`, corners included); `x`, `y` | Ctrl+C and a Ctrl+V click on the same map. The copy's top-left lands on `x,y`. An area copy also carries the area's tiles, heights and objects. |
| `move_area` | `area`, `dx`, `dy`; optional `records` (default every unit, structure and sack in the area) | A drag inside a selection box: the records and the objects move, the tiles and heights stay. |
| `set_units_group` | `units` (unit indices of one player), `group` | The group keys (Ctrl+0..9) and the To group question. `group` 0 takes the units out of their groups; -1 is a new group, the lowest free number. |
| `add_stock` | `unit` | K on a selected unit: the unit's shop stock. |
| `add_item_recipe` | `kind` (`Stock`, `Sack`), `index`, `item` (from 1) | I and its item prompt. |
| `write_message` | `parts`; `page`, `action` or both | The message editor's Apply. `page` alone rewrites that page; `action` alone gives the action a new page; both point the action at the page and rewrite it. A part is `{"body"}` for a new card, with `source` for a card the page holds (`messages` lists them) and optional `speaker`; a card of the page left out is struck. |
| `add_box_area` | `area`; optional `trigger` (default -1, a new trigger) | The box area tool (L) and its trigger choice: one more box condition pair on the trigger. A trigger with three pairs is refused. |
| `set_goal_target` | `goal`, `kind` (`Unit`, `Structure`), `index` | A pick from the Goals (G) report. `goals` lists the goals and the kind each takes. |
| `set_map_field` | `scope` (`metadata`, `pages`, `package`), `key`, `value` | The Metadata (M) and Pages (X) prompts and the mod package window (W). `map_fields` lists the keys. The package fields `folder`, `folder-open`, `output` and `source` open a folder or switch the mission and stay in the window. |
| `randomize` | `plane` (`terrain`, `heights`), `seed`; optional `verbose` | Ctrl+E and Ctrl+Shift+R. One seed on one map gives one result. |
| `undo`, `redo` | `{}` | The window's Undo and Redo, the same function: it leaves a drag or pick and keeps an open message page on a page the map still holds. |

### Height brush

A height `paint` is the window's height stroke (`pkg/game` `paintStroke`
over `pkg/mapedit` `Brush`); no other route writes heights. `cells` are the
stroke's centres in order. The brush covers `radius` cells around each
centre (a circle or a square of `shape`). The line between two centres is not
filled: the window's drag fills it from its pointer samples before the stroke,
and a typed stroke takes the centres as given, so a line takes every centre on
it. Three centres 9 cells apart with radius 4 cover 3 × 49 = 147 cells.

| Mode | `value` | Each covered cell |
| --- | --- | --- |
| `step` (default) | a change -31..31, not 0 | gains `value` once, however many centres cover it |
| `level` | a change -31..31, not 0 | takes the target: the first centre's height before the stroke plus `value`, clamped to 0..127; no blend at the rim |
| `smooth` | a change -31..31, not 0 | gains `value` × w / 64 around each centre in turn, w from 64 at the centre through 63, 59, 53, 45, 36, 24, 12 to 0 one cell past the radius (a cosine of the distance); a cell two centres cover gains twice |
| `set` without `flat` | a height 0..255, written clamped to 0..127 | only the first centre counts: the cells of a circle of radius at least 1 around it move toward `value` by the cosine of their distance, and the other centres are unused |
| `set` with `flat` | a height 0..255, written clamped to 0..127 | every covered cell of every centre takes `value` |

`level` with `value` 30 on a new map (every cell 64) gives 94: the target is
64 + 30, written to every covered cell. `smooth` with `value` 12 and radius 4
gives the centre +12 and a cell next to it +11 (12 × 63 / 64, rounded toward
0).
The brush bounds only some row steps (`TERR-245`, `TERR-251`), and a step above 31 can
remain. `step`, `level`, `smooth` and `set` with `flat` lower a covered cell
that stands more than 31 above the cell in the row above it, whether or not
that cell is covered; in the rows above the first centre's row they first
raise a covered cell that stands more than 31 below the cell under it.
`set` without `flat` bounds only the covered cells whose column distance from
the first centre is less than their row distance, each against its neighbour
towards the first centre's row (the upper cell is raised above the centre, the
lower cell lowered below it); the first centre's row and the other covered
cells are not bounded, so covered cells can stand 33 apart (`set` 127 at radius 2 leaves 89 over 56). A cell
the stroke does not cover never changes, so a covered cell can stand 32 or
more from an uncovered neighbour: `set` with `flat` 0 over a map at 64 leaves
steps of 64 below the circle's lower half. Horizontal steps are never bounded. `heights`
reads the stored result and `height_change` in the report counts it.

The View menu's height numbers show 64 as 0. `heights` and the report's
`heights` give stored heights; `heights` names that offset as `label_zero`.

Read tools answer JSON and change nothing:

| Tool | Arguments | Answer |
| --- | --- | --- |
| `record` | `kind`, `index`; optional `verbose` | The record, with `ScriptID` for a unit or structure, and every field: key, value, text, choices, read-only. A unit or structure also has the read-only field `script_id`; no tool writes it. A unit that sets a definition names it in `Definition`: the Humans row its npc flag or definition id reaches, or the id no row answers. A field with more than 32 choices lists none; `choices_omitted` names each such field and its count, and `"verbose":true` lists them all. A message action's page list holds every page of the map. |
| `records` | `x0`, `y0`, `x1`, `y1` | The records whose footprint meets the cell range, corners included. |
| `heights` | `x0`, `y0`, `x1`, `y1` | The stored height of every cell of the range, corners included, at most 4096 cells: `rows` top to bottom, each left to right, `range`, `min`, `max` and `label_zero` (64, the height the View menu shows as 0). A reversed range or a cell off the map is refused. |
| `add_steps` | optional `kind`, `x`, `y`, `values` | Without `kind`, the kinds `add_record` adds; with it, the steps in order with choices, defaults, whether each is asked, and a refusal of a given value. |
| `brushes` | `{}` | Planes, terrain classes with names and variant counts, object codes with names, largest radius and height change. |
| `check` | `{}` | The map report: every error and warning with its rule, record and cell. |
| `trigger_items` | optional `verbose` | The wizard's conditions and actions: item key, label, parameters with kind and default, comparison. A unit or group parameter's `choices` is `units` or `groups`: the list of that name at the top of the answer, which holds the placed units by script ID or the groups the map's units carry. A unit parameter also names `hero_band`, `10001..11000`, which any unit parameter also takes. A unit choice whose unit sets a definition names it, as `record`'s `Definition` does. With `"verbose":true` each parameter lists its own `choices`, the hero band's units included, and the two lists at the top are left out. |
| `map_fields` | `scope` | The fields of the Metadata or Pages prompt or of the mod package window. |
| `goals` | `{}` | The Goals (G) report: goal index, arm, text, `pick` (the kind a target takes, empty when the goal takes none), the script record and the target's cell. |
| `messages` | optional `page` | Without `page`, the message pages the map's script shows and their text; with it, the page's parts in order (`source`, `number`, `body`, `speaker`) and the speakers a part takes. |
| `unit_settings` | `index`; optional `fields` (number fields) | The unit settings window's preview: every field of the unit as it reads with the given values set. Nothing is written. |

### Script IDs, groups, the playable area and warnings

- **Script IDs.** A script parameter names a unit or structure by its script
  ID, not by its index. `add_record` answers the new unit's or structure's
  `script_id`; `record` gives it as `ScriptID` and the read-only field
  `script_id`; `trigger_items` lists the placed units' IDs, with the hero
  band 10001..11000, as a unit parameter's choices, and the groups the map's
  units carry as a group parameter's. `write_trigger` refuses
  a unit or group value outside those choices. The window lists heroes 1 to 8
  and keeps a loaded hero value. `check` names, under `script-reference`, a unit
  value above 11000, which no placement holds.
- **Groups.** A unit's Add steps include `group`. Without a `group` value a
  new unit takes the lowest group number no unit of the map carries, so each
  unit added that way is a script group of its own; a `group` value always
  wins. `set_units_group` puts units in one group afterwards.
- **Playable area.** The border ring is the 8 cells next to each edge of the
  map (`mapload.Playable`); the game removes a unit placed there at load
  (`TERR-PLACE-208`). `new_map` and `map` answer `playable` as `x0,y0..x1,y1`
  (corners included), or `none` on a map of 16 cells or fewer on a side.
  `add_record`, `move_record`, `clone_record`, `move_records`, `copy_records`
  and the drop zone tools refuse a unit, a structure or a drop zone on a ring
  cell, naming the playable area in the refusal. A structure is judged by
  its cell, the footprint's top-left. Terrain, height, object (scenery) and
  passability strokes take ring cells, and so does a sack or a cell spell
  added on one. `check` warns of a structure whose footprint meets the ring
  (`structure-border`): one from a map written elsewhere, or one whose cell
  is playable and whose footprint reaches into the ring, which the edit's
  `warnings` then name.
- **Warnings.** An edit's `warnings` are the `check` findings on the records
  it touched; they come from the same rule table, so `check` gives the same
  rows for the whole map.

### Example: a small map from nothing

```text
new_map       {"width":32,"height":32}
brushes       {}
paint         {"plane":"terrain","value":<sand class>,"radius":4,"shape":"square","cells":[[16,16]]}
paint         {"plane":"height","value":6,"radius":2,"cells":[[8,24]]}
add_record    {"kind":"Player","values":{"name":"Rivals"}}
add_steps     {"kind":"Unit","x":12,"y":12,"values":{"class":40}}
add_record    {"kind":"Unit","x":12,"y":12,"values":{"class":40,"owner":<first owner>}}
add_record    {"kind":"Unit","x":20,"y":20,"values":{"class":40,"owner":<last owner>}}
add_record    {"kind":"Structure","x":10,"y":17,"values":{"kind":5,"owner":<first owner>}}
add_drop_zone {"x":13,"y":14}
trigger_items {}
write_trigger {"name":"Victory","conditions":[{"item":"c.heroarea","params":[<hero>,[18,18,22,22]]}],"actions":[{"item":"a.win"}]}
check         {}
save_as       {"path":"C:\\Maps\\small.alm"}
```

The values in angle brackets come from the reads before them: `brushes`
names the classes, `add_steps` the owner choices, `trigger_items` the
condition's default hero. `TestMapEditorMCPBuildsAPlayableMapFromEmpty`
runs this session on a synthetic table.

### Example: a village whose chief's death wins

```text
new_map       {"width":32,"height":32}
add_record    {"kind":"Structure","x":16,"y":9,"values":{"kind":5}}
add_record    {"kind":"Unit","x":10,"y":10,"values":{"class":40}}
add_record    {"kind":"Unit","x":14,"y":14,"values":{"class":40}}
record        {"kind":"Unit","index":1}
add_record    {"kind":"Unit","x":18,"y":18,"values":{"class":40,"group":1}}
trigger_items {}
write_trigger {"name":"Chief","conditions":[{"item":"c.unitdead","params":[<chief>]}],"actions":[{"item":"a.win"}]}
check         {}
```

`new_map` answers `"playable":"8,8..23,23"`, so every placement stands in
8..23 on both axes. The second unit is the chief: `<chief>` is the
`script_id` its `add_record` answers, which `record` gives as `ScriptID` and
`trigger_items` lists among the `c.unitdead` unit choices. The first two
units take groups 1 and 2, one each; the third joins group 1 because the
call names it. A `c.unitdead` row naming a value the map does not hold,
such as 20001, is refused. `TestMapEditorMCPVictoryTriggerNamesTheUnitAddRecordReports`
runs this session on a synthetic table and finds the condition naming the
chief's cell.

### Example: a selection, a group, a stock, a message and a goal

```text
map             {"x":18,"y":14}
move_records    {"records":[{"kind":"Unit","index":2},{"kind":"Unit","index":3}],"dx":0,"dy":4}
set_units_group {"units":[2,3],"group":-1}
add_stock       {"unit":0}
add_record      {"kind":"Message","parts":[{"body":"Hold the bridge"},{"body":"Then go north"}]}
messages        {"page":<page>}
goals           {}
set_goal_target {"goal":<goal>,"kind":"Unit","index":3}
save_as         {"path":"C:\\Maps\\session.alm"}
```

`<page>` is the `page` the `add_record` report names; `<goal>` is a goal
whose `pick` is `Unit`. Each edit is one Undo step.
`TestMapEditorMCPExampleSessionSavesAndReopens` runs this session on a
synthetic map with four units and a script with goals, then reopens the
saved map and its message page.

### Not covered by typed tools

These edits keep their window route only; drive them with `key`, `pointer`
and `text`:

- a paste into another open map;
- the mod package window's folder, output and mission choice, and its
  confirmations.

A message page written from inside the trigger wizard, with the wizard's
unapplied trigger, as one edit is `write_trigger` with `parts` on the
dialogue action row. No tool runs a trigger without the game: a trigger's
effect is seen in a mission only.

The unit settings window's Apply is `set_object`; its preview is
`unit_settings`.

No route resizes an open map. Save, Save As and Open remain the tools
above.

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `-mcp` | empty | `stdio` or `http`; empty disables MCP. |
| `-mcp-address` | `127.0.0.1:0` | HTTP bind address; any other host is refused. Port 0 tries the recorded port first. |
| `-mcp-new-token` | off | Make a new bearer token and record it in `mapedit-mcp.json`. |
| `-mcp-pace` | `0` | Pause between tool calls, from zero to one minute. |

MCP requires the window route. It cannot be combined with `-list`, `-check`,
`-snapshot` or `-out`. The existing `-assets`, `-map`, `-new` and `-mod` flags
retain their meanings.

## First visible check

After connecting, ask the client to make these three tool calls in order:

1. `state {}`: identify the open document and current prompts.
2. `key {"name":"help"}`: the Help report opens in the visible window.
3. `frame {}`: inspect the PNG of that window and its Help report.

Use `key {"name":"escape"}` afterwards to close Help. To point at a map
record, obtain `cell {"x":10,"y":12}`, then pass its returned coordinates
to `pointer` with `action: "click"`. Read `state` after confirmations and
edits; `key {"name":"undo"}` uses the owner's normal history.

## Protocol and limits

The current revision is `2026-07-28`: `server/discover`, per-request version and
capability metadata, and `resultType: "complete"` are supported. Modern
`server/discover` and `tools/list` results include `ttlMs: 300000` and
`cacheScope: "public"` over HTTP and stdio. Other results and legacy replies
carry no cache hints. HTTP requests supply matching `MCP-Protocol-Version`,
`Mcp-Method` and, for tools/call,
`Mcp-Name` headers. Legacy `2025-11-25` clients can use `initialize`,
`notifications/initialized`, `ping`, `tools/list` and `tools/call` on the same
protocol layer. HTTP uses JSON responses without sessions or SSE streams.
Requests are limited to 1 MiB, and the queue holds 64 waiting calls. Calls are
applied in queue admission order, one at a time: a call starts after the one
before it has answered, so calls sent together run in the order they arrive
and none is refused because another is running. A call can still be refused
by what an earlier call left open, such as a prompt; the refusal names it
(see Typed edit tools). Concurrent clients share the same document and undo
history.

Typed edit tools come from the editor command table the window's own routes
call (see Typed edit tools); there is no second edit path beside it.

Resources, subscriptions, progress notifications, tasks, prompts, OAuth and
SSE are not implemented. Stdio processes one request at a time; cancellation
notifications do not interrupt a running call. HTTP cancellation can remove a
waiting request, but cannot roll back an edit already applied. Use Undo for
that. A minimized or stalled drawing loop can delay completion. MCP cannot
attach retroactively to a window launched without the flag, and the client
cannot automate an operating-system dialog. Agent Open/Save actions use the
editor's existing typed fallback prompts.

Ordinary tests exercise synthetic maps and a synthetic capture callback. They
verify protocol, queue/frame acknowledgement, pointer input, undo and temporary
Save As output, and that each typed edit writes the bytes and undo steps of
its window route. They do not establish installed EN/RU rendering or a live GPU
readback. A release must run the existing EN/RU editor frame, interaction and
move/undo/save/reopen witnesses, plus the visible first check above.

## References

- [MCP versioning](https://modelcontextprotocol.io/specification/2026-07-28/basic/versioning)
- [MCP caching](https://modelcontextprotocol.io/specification/2026-07-28/server/utilities/caching)
- [MCP Streamable HTTP](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http)
- [Claude Code MCP setup](https://code.claude.com/docs/en/mcp)

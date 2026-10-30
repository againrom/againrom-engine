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
URL, a random bearer token and its receipt path. The selected player
profile contains `mapedit-mcp-<process-id>-<suffix>.json`, with `url` and `token` fields.
The profile is resolved by the game's existing profile rules, including
`AGAINROM_PROFILE`. An executable inside an installation uses its selected
private `Againrom` profile, with the existing user configuration fallback.
Explicit profile paths keep the install write fence.

Replace `TOKEN` with that window's token:

```powershell
claude mcp add --transport http againrom-editor http://127.0.0.1:8765/mcp --header "Authorization: Bearer TOKEN"
```

The default address is `127.0.0.1:0`, which selects a free port. Use the URL
printed in the window or stored in the profile when using that default.
The token changes on every launch. The receipt remains after shutdown; a
receipt does not imply that its process is still running. It is created
exclusively with file mode `0600` on systems supporting those permission bits.
Windows uses the profile directory's access control.

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
| `action_<name>` | `{}` | One generated tool for every named key row; calls `Key(name)`. |
| `key` | `{"name":"unit"}` | Calls `Key` with a name from `actions`. Optional `ctrl` and `shift` use the window's native text-navigation policy, for example `{"name":"left","ctrl":true}`. |
| `text` | `{"text":"value"}` | Calls `TextInput` on the focused field. |
| `pointer` | `{"action":"click","x":400,"y":300}` | Calls `Pointer` at window pixels; returns window state. |
| `cell` | `{"x":10,"y":12}` | Calls `ShowCell`, then `CellPoint`; returns window `x,y`. |
| `state` | `{}` | Status, message, panel, popup, prompt, Help/report, dropdown, groups, selection, object fields, dirty flag, path and window size. |
| `frame` | `{}` | Reads the editor's drawn window after Draw, including the agent pointer. It does not call the diagnostic snapshot composer. |
| `map` | `{"x":10,"y":12}` | Open document size, players, units, structures, loot, script nodes/triggers, other records, and the cell's terrain, height, objects, overlay and passability. A missing plane is null or marked unknown. |
| `open` | `{"path":"C:\\Maps\\map.alm"}` | Uses the window's typed Open prompt. Unsaved changes still ask before replacing the document. |
| `save` | `{}` | Uses the window's Save key route, including in-place and mod destinations, validation and prompts. |
| `save_as` | `{"path":"C:\\Maps\\copy.alm"}` | Uses the window's typed Save As prompt. Install fences, overwrite checks and validation remain active. |

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
popup field is open; finish or cancel it with `key` first.

An edit answers a JSON report: `changed`, `record` (the record the call
names, as `{"Kind","Index"}`), `dirty`, `can_undo`, `can_redo`, `message`
(the status line), `prompt_open`, `cells` (up to 1024 changed cells as
`[x,y]`) with `cells_total`, `planes` (the count of changed cells per plane:
`terrain`, `passability`, `height`, `objects`), and `records` (up to 256
changed records) with `records_total`. The report compares what the map
stores before and after the call: each cell's tile word, its passability
bit, height and object byte, and each record's stored bytes, so a field
change names its record. A call that changes nothing reports `changed`
false and no cells or records. `new_map` names `width` and `height`
instead. A number field that stores 0 as "inherit" refuses a typed 0 with
the field prompt's words; a field entry without `value` is the prompt's
empty text and inherits.

| Tool | Arguments | Window route |
| --- | --- | --- |
| `new_map` | `width`, `height`; optional `altitude` (0..126, default 64), `multiplayer` | The catalogue's New question. Sizes are multiples of 8 from 32 to 256. Unsaved changes still ask. |
| `paint` | `plane` (`terrain`, `height`, `objects`, `passability`), `cells` (`[[x,y],...]`, the stroke in order); optional `value`, `radius` (default 0), `shape` (`circle`, `square`), `fill` (`pattern`, `random`, `variant`) and `variant` for terrain; `set`, `flat`, `mode` (`step`, `level`, `smooth`) for height | One brush stroke. `value` is a terrain class or object code from `brushes`, a height change (-31..31, not 0), the height with `set`, or 1 to close and 0 to open a cell. |
| `add_record` | `kind`; optional `x`, `y` for a record on a cell; optional `values` by step key | The Add window, step by step with its defaults and checks. `add_steps` lists the steps and choices. Kinds: `Enchantment`, `Unit`, `Structure`, `Sack`, `Player`, `Trigger`, `Action`, `Condition`. |
| `set_field` | `kind`, `index`, `key`; `value` (number) or `text` | The field prompt (F). `record` lists the keys. |
| `set_fields` | `kind`, `index`, `fields` (a list of `{"key","value"}` for a number or `{"key","text"}` for a text) | A script record popup's Apply: one edit. |
| `set_object` | `kind`, `index`; optional `fields`, `cell` (`[x,y]`) | The object popup's Apply: fields and a move as one edit. |
| `move_record` | `kind`, `index`, `x`, `y` | A drag on the map. |
| `clone_record` | `kind`, `index`, `x`, `y` | Clone (C) and a click. Units, structures and sacks. |
| `delete_record` | `kind`, `index` | Del. |
| `set_drop_cell` | `x`, `y` | The drop tool (D) on a map without drop zones. |
| `add_drop_zone`, `move_drop_zone`, `delete_drop_zone` | `x`, `y`; `zone` (from 0) | The drop tool's click, drag and Del. |
| `write_trigger` | `name`, `actions`; optional `conditions`, `once` (default true), `index` (default -1 adds a trigger) | The trigger wizard's Apply. Each row is `{"item", "params"}`, a condition also `comparison` when its comparison is chosen; `trigger_items` lists the items. A parameter is an integer, `[x,y]` for a cell or `[x1,y1,x2,y2]` for an area. |
| `set_map_field` | `scope` (`metadata`, `pages`), `key`, `value` | The Metadata (M) and Pages (X) prompts. `map_fields` lists the keys. |
| `randomize` | `plane` (`terrain`, `heights`), `seed` | Ctrl+E and Ctrl+Shift+R. One seed on one map gives one result. |
| `undo`, `redo` | `{}` | The window's Undo and Redo, the same function: it leaves a drag or pick and keeps an open message page on a page the map still holds. |

Read tools answer JSON and change nothing:

| Tool | Arguments | Answer |
| --- | --- | --- |
| `record` | `kind`, `index` | The record and every field: key, value, text, choices, read-only. |
| `records` | `x0`, `y0`, `x1`, `y1` | The records whose footprint meets the cell range, corners included. |
| `add_steps` | optional `kind`, `x`, `y`, `values` | Without `kind`, the kinds `add_record` adds; with it, the steps in order with choices, defaults, whether each is asked, and a refusal of a given value. |
| `brushes` | `{}` | Planes, terrain classes with names and variant counts, object codes with names, largest radius and height change. |
| `check` | `{}` | The map report: every error and warning with its rule, record and cell. |
| `trigger_items` | `{}` | The wizard's conditions and actions: item key, label, parameters with kind and default, comparison. |
| `map_fields` | `scope` | The fields of the Metadata or Pages prompt. |

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

### Not covered by typed tools

These edits keep their window route only; drive them with `key`, `pointer`
and `text`: moving, deleting, copying and pasting a selection or an area,
units into a group, shop stock and item recipes, message text and the Add
window's Message kind, a trigger's box area, goal targets, forest density
and thinning strokes, and the mod package window's fields. No route resizes
an open map. Save, Save As and Open remain the tools above.

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `-mcp` | empty | `stdio` or `http`; empty disables MCP. |
| `-mcp-address` | `127.0.0.1:0` | HTTP bind address; any other host is refused. |
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
applied in queue admission order. Concurrent clients share the same document
and undo history.

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

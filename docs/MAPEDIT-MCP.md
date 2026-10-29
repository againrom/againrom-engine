# Map editor MCP

The map editor can accept MCP tool calls while its window remains visible.
MCP is off unless `-mcp` is supplied. Each tool runs through the editor's input
doors on its window loop. The result follows the frame that applied it. Edits
use the same undo history as local input.

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

There are no typed semantic tools such as "place unit X at cell 10,12". Those
need a shared editor command table for unit/structure placement, field edits,
brush strokes and script operations, with both the window and MCP dispatching
that table. Adding direct model writes would create a second edit path.

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
Save As output. They do not establish installed EN/RU rendering or a live GPU
readback. A release must run the existing EN/RU editor frame, interaction and
move/undo/save/reopen witnesses, plus the visible first check above.

## References

- [MCP versioning](https://modelcontextprotocol.io/specification/2026-07-28/basic/versioning)
- [MCP caching](https://modelcontextprotocol.io/specification/2026-07-28/server/utilities/caching)
- [MCP Streamable HTTP](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http)
- [Claude Code MCP setup](https://code.claude.com/docs/en/mcp)

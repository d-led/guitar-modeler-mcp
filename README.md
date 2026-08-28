# headrush-gigboard-mcp

An MCP server and CLI for designing and writing **HeadRush Gigboard** sound
presets (`.rig` patch files), written in Go.

Give it a song and a tone description, and it will:

1. translate real-world hardware (amps, cabs, mics) into the HeadRush models
   that emulate them,
2. order the effects into a musically sensible signal chain,
3. write a `.rig` patch file you can load onto the device,
4. produce a human-readable HTML page of the settings used,
5. decode any existing `.rig` file so an agent can analyze or fix it.

## How it works

The device model data is embedded in the binary:

- `internal/catalog` — every amp, cab, mic and effect module, plus the
  **translation layer** that maps real hardware ("Marshall JCM800", "blackface
  Deluxe Reverb", …) onto device models. This is the encoded version of the
  hardware↔HeadRush translation table.
- `internal/assets/data/blocks` — the factory block definitions captured from
  the device backup, used as defaults for every effect module.
- `internal/rig` — builds the exact on-disk `.rig` format (outer JSON envelope
  whose `content` field is a second JSON document describing the signal chain).

## Build

```sh
go build -o headrush-gigboard-mcp .
go test ./...
```

Requires Go 1.27+.

## CLI

```sh
# What models exist?
headrush-gigboard-mcp catalog amps
headrush-gigboard-mcp catalog cabs
headrush-gigboard-mcp catalog mics
headrush-gigboard-mcp catalog fx
headrush-gigboard-mcp catalog presets "Tape Echo"

# Translate real hardware into device models
headrush-gigboard-mcp translate amp "Marshall JCM800"
headrush-gigboard-mcp translate cab "greenback 4x12"
headrush-gigboard-mcp translate mic "SM57"

# Dial in a tone and write the patch + HTML report
headrush-gigboard-mcp design \
  --name "Brown Sound" --song "Van Halen - Panama" \
  --amp "Marshall JCM800" \
  --fx '[{"type":"Green JRC-OD","enabled":true},{"type":"Tape Echo","enabled":true}]' \
  --out ./rigs

# Decode an existing rig for analysis
headrush-gigboard-mcp decode "001 HOW DOES IT FEEL.rig"

# Render an HTML report for an existing rig
headrush-gigboard-mcp report --rig "001 HOW DOES IT FEEL.rig"

# Install the MCP server in a client (default: VS Code user profile = global)
headrush-gigboard-mcp mcp install
headrush-gigboard-mcp mcp install --target workspace   # .vscode/mcp.json here
headrush-gigboard-mcp mcp install --target claude      # Claude Desktop
headrush-gigboard-mcp mcp install --print              # show the config only
headrush-gigboard-mcp mcp uninstall --target vscode
```

## MCP server

Run over stdio:

```sh
headrush-gigboard-mcp serve
```

`mcp install` writes the registration for you. The equivalent manual
`.vscode/mcp.json` entry is:

```json
{
  "servers": {
    "headrush-gigboard-mcp": {
      "type": "stdio",
      "command": "/absolute/path/to/headrush-gigboard-mcp",
      "args": ["serve"]
    }
  }
}
```

### Tools

| Tool | Purpose |
| --- | --- |
| `catalog_list_amps` | List amps with the real hardware each emulates |
| `catalog_list_cabs` | List cabinet models |
| `catalog_list_mics` | List microphone models |
| `catalog_list_fx` | List effect modules by category |
| `catalog_list_block_presets` | List factory presets for one effect |
| `translate_amp` / `translate_cab` / `translate_mic` | Hardware → device model |
| `design_rig` | Translate, order, write `.rig` + HTML report |
| `render_report` | HTML report for an existing `.rig` |
| `rig_decode` | Decode a `.rig` into chain + parameter values |

Example agent workflow: list amps → translate the song's amp → `design_rig` with
effects → read the report → tweak by re-running `design_rig` with parameter
overrides or by decoding and fixing an existing file.

## Data provenance

Amp/cab/mic model lists come from the device backup and the community-maintained
[Gigboard Hints](https://boguz.github.io/gigboardhints/) translation table.
Where the emulated amplifier is not publicly documented, the brand is left empty
rather than guessed.

## Testing

```sh
go test ./...             # unit + approval + integration tests
go test -race ./...       # with the race detector
UPDATE_GOLDEN=1 go test ./...   # regenerate approval snapshots after a change
```

- **Unit tests** cover the translation layer, rig builder (round-trip,
  validation, parameter overrides), design ordering, and config install logic.
- **Approval tests** (`internal/golden`) snapshot the full catalog, the exact
  `.rig` JSON the builder emits, and the HTML report, so any change to the
  device format or model data is caught by a diff.
- **Integration tests** (`internal/tools/integration_test.go`) drive the real
  MCP server over the JSON-RPC stdio transport: initialize, tools/list, and a
  full `design_rig` → `rig_decode` → `render_report` round trip.

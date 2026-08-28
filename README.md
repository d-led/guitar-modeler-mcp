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
  Deluxe Reverb", …) onto device models. Each entry carries a `modeled_after`
  string and a `confirmed` flag from the [Gigboard Hints](https://boguz.github.io/gigboardhints/)
  data, and each module's listing includes **capability keywords** (e.g.
  `pitch shift`, `reverb`, `delay`) pre-computed from its parameters.
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
headrush-gigboard-mcp catalog params "Tape Echo"   # ranges, units, options

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
| `catalog_list_amps` | List amps with the real hardware each emulates (`modeled_after`) and capabilities |
| `catalog_list_cabs` | List cabinet models |
| `catalog_list_mics` | List microphone models |
| `catalog_list_fx` | List effect modules with capabilities (e.g. pitch shift, reverb, delay) |
| `catalog_list_block_presets` | List factory presets for one effect |
| `catalog_list_module_params` | Describe a module's parameters (kind, range, unit, options) |
| `translate_amp` / `translate_cab` / `translate_mic` | Hardware → device model |
| `design_rig` | Translate, order, write `.rig` + HTML report (serial or parallel chain) |
| `render_report` | HTML report for an existing `.rig` |
| `rig_decode` | Decode a `.rig` into chain + parameter values |

Example agent workflow: list amps → translate the song's amp → `design_rig` with
effects → read the report → tweak by re-running `design_rig` with parameter
overrides or by decoding and fixing an existing file.

## Signal chain & parallel routing

The Gigboard's 11 chain slots are not always one straight line — the chain can
split into **two parallel paths**. The topology is recorded in the `Routing`
field, which (across the 293 device backup rigs) takes exactly three values:

| `Routing` | Topology | Slot layout (1–11) |
| --- | --- | --- |
| `S` | Serial | 11 slots, one path |
| `SPS-1` | Serial → parallel → serial | 3 shared slots → path A (3) + path B (3) → 2 shared slots |
| `PS-1` | Parallel from the input → serial | path A (3) → path B (4–5) → remaining shared slots |

Constraints, measured from the backups:

- **`SPS-1` (dual amp).** The split and merge points are fixed: slots 1–3 are
  shared (pre-amp FX and/or a shared amp+cab), slots 4–6 are **path A**, slots
  7–9 are **path B**, and slots 10–11 are shared (post FX). Each path holds at
  most **3** blocks, the prefix **3**, the suffix **2**.
- **Same amp on both channels vs two amps.** There is no separate switch: a
  second amp block is just another `Amp` module. `Amp` (path A) and `Amp 2`
  (path B) may carry the **same** model (same amp on both channels) or
  **different** models. A single `Amp` placed in the shared prefix (slots 1–3)
  feeds *both* parallel paths — the wet/dry/wet pattern (`+WDW-*` presets).
- **`PS-1`** splits at the input: path A is the first **3** slots, path B the
  next **4–5**, then the merge and the remaining serial slots.
- **Path mixer.** `Para1Level`/`Para2Level` (dB, default −6),
  `Para1Pan`/`Para2Pan` (−100…100, default 0) and `ParaDelay` (ms, default 0)
  balance the two paths; pan −100/+100 hard-pans paths A/B for wet/dry/wet.
- **Slot budget.** 11 slots total; the builder rejects any section that exceeds
  its budget rather than silently overflowing.

In `design_rig`, pass `routing: "SPS-1"` with `amp2` (and optional `cab2`,
`mic2`) for a dual-amp rig, or `routing: "SPS-1"` without `amp2` and
`path_a_fx`/`path_b_fx` for a shared-amp split. The same is available on the
CLI via `--routing`, `--amp2`, `--cab2`, `--mic2`, `--path-a-fx`,
`--path-b-fx`.

The builder **validates every parameter** against the device's specifications
(extracted from `headrush-desktop/renderer/config/modules/*.ts` plus the
backup-derived catalog): unknown parameter names, out-of-range numbers and
invalid enum options are rejected with a clear message, so an invalid `.rig` is
never written. Regenerate the extracted spec with
`node scripts/extract-module-config.cjs <headrush-desktop-root> internal/modspec/data/params.json`,
and the hints data with
`node scripts/extract-hints.cjs <gigboardhints-root> internal/catalog/data/hints.json`.

## Data provenance

Amp/cab/mic model lists come from the device backup and the community-maintained
[Gigboard Hints](https://boguz.github.io/gigboardhints/) translation table.
Where the emulated amplifier is not publicly documented, the brand is left empty
rather than guessed.

## Trademarks & disclaimer

All trademarks, logos and brand names are the property of their respective
owners. All company, product and service names used in this project are for
identification purposes only; use of these names, trademarks and brands does not
imply endorsement. This project is not affiliated with, endorsed by, or
sponsored by HeadRush or any of the referenced brands.

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

# guitar-modeler-mcp

An MCP server and CLI for designing and writing guitar-modeler presets for
multiple hardware devices, written in Go. The first supported device is the
**HeadRush Gigboard** (`.rig` patch files); the architecture is a
device-agnostic design core with per-device backends, so Quad Cortex, Mooer
GE100 Pro and others can be added as new backends.

## Roadmap

- **Gigboard** — implemented (the only backend today).
- **Quad Cortex** — planned; see [OpenCortex](https://github.com/VanIseghemThomas/OpenCortex)
  (open-source QC preset format work) as a starting point for the preset format.
- **Mooer GE100 Pro and others** — planned, pending a sample preset dump to
  reverse-engineer each format.

Give it a song and a tone description, and it will:

1. translate real-world hardware (amps, cabs, mics) into the models the device
   emulates,
2. order the effects into a musically sensible signal chain,
3. write the device's preset file you can load onto the hardware,
4. produce a human-readable HTML page of the settings used,
5. decode an existing preset so an agent can analyze or fix it.

## How it works

The design core (translation, effect ordering, hardware-control assignment,
level estimation, report rendering, the agent workflow) is device-agnostic. A
per-device backend supplies the model catalog and preset file format:

- **Gigboard backend** — `internal/catalog` (amps/cabs/mics/FX + the translation
  layer from [Gigboard Hints](https://boguz.github.io/gigboardhints/)), and
  `internal/rig` (the exact on-disk `.rig` format: outer JSON envelope whose
  `content` field is a second JSON document describing the signal chain).
- `internal/assets/data/blocks` — factory block definitions captured from the
  device backup, used as defaults for every effect module.
- `internal/docs/agent-guide.md` — the agent-facing guide (signal-chain topology,
  parallel routing constraints, effect categories, workflow). It is embedded in
  the binary and exposed to agents through the `get_guide` MCP tool.

## Build

```sh
go build -o guitar-modeler-mcp .
go test ./...
```

Requires Go 1.27+.

## Releases

Binaries are built with [GoReleaser](https://goreleaser.com) and published as
GitHub release artifacts when a `v*.*.*` tag is pushed:

- Linux `amd64` / `arm64`
- Windows `amd64`
- macOS universal (`amd64` + `arm64`)

Each push to `main` also uploads fresh "latest" per-platform binaries as CI
build artifacts (see `.github/workflows/ci.yml`).

```sh
git tag v1.0.0
git push origin v1.0.0   # goreleaser publishes the release
```

## CLI

```sh
# What models exist?
guitar-modeler-mcp catalog amps
guitar-modeler-mcp catalog cabs
guitar-modeler-mcp catalog mics
guitar-modeler-mcp catalog fx
guitar-modeler-mcp catalog fx --category delay
guitar-modeler-mcp catalog fx-categories
guitar-modeler-mcp catalog presets "Tape Echo"
guitar-modeler-mcp catalog params "Tape Echo"   # ranges, units, options

# Translate real hardware into device models
guitar-modeler-mcp translate amp "Marshall JCM800"
guitar-modeler-mcp translate cab "greenback 4x12"
guitar-modeler-mcp translate mic "SM57"

# Fuzzy-search amps, cabs, mics and effects (by name or the real hardware)
guitar-modeler-mcp search "JCM800"
guitar-modeler-mcp search "tube screamer" --kind fx

# Where each effect category goes in each chain layout
guitar-modeler-mcp fx-placement

# Dial in a tone and write the patch + HTML report
guitar-modeler-mcp design \
  --name "Brown Sound" --song "Van Halen - Panama" \
  --amp "Marshall JCM800" \
  --fx '[{"type":"Green JRC-OD","enabled":true},{"type":"Tape Echo","enabled":true}]' \
  --output-level 6 \
  --out ./rigs

# Decode an existing rig for analysis
guitar-modeler-mcp decode "001 HOW DOES IT FEEL.rig"

# Estimate a rig's output level and the RigVolume to reach 0 dB
guitar-modeler-mcp level "001 HOW DOES IT FEEL.rig"

# Render an HTML report for an existing rig
guitar-modeler-mcp report --rig "001 HOW DOES IT FEEL.rig"

# Install the MCP server in a client (default: VS Code user profile = global)
guitar-modeler-mcp mcp install
guitar-modeler-mcp mcp install --target workspace   # .vscode/mcp.json here
guitar-modeler-mcp mcp install --target claude      # Claude Desktop
guitar-modeler-mcp mcp install --print              # show the config only
guitar-modeler-mcp mcp uninstall --target vscode
```

## MCP server

Run over stdio:

```sh
guitar-modeler-mcp serve
```

`mcp install` writes the registration for you. The equivalent manual
`.vscode/mcp.json` entry is:

```json
{
  "servers": {
    "guitar-modeler-mcp": {
      "type": "stdio",
      "command": "/absolute/path/to/guitar-modeler-mcp",
      "args": ["serve"]
    }
  }
}
```

### Tools

| Tool | Purpose |
| --- | --- |
| `get_guide` | Return the embedded agent guide (chain topology, routing constraints, categories, workflow) |
| `get_fx_placement` | Where each effect category goes (pre/post amp) and the slot budget per chain layout |
| `search_catalog` | Fuzzy-search amps/cabs/mics/effects by name or the real hardware they emulate (both directions) |
| `catalog_list_amps` | List amps with the real hardware each emulates (`modeled_after`), a `gain` character (clean/edge of breakup/crunch/high gain/bass) and capabilities |
| `catalog_list_cabs` | List cabinet models |
| `catalog_list_mics` | List microphone models |
| `catalog_list_fx` | List all effect modules with capabilities |
| `catalog_list_fx_categories` | List effect categories (distortion, dynamics, eq, expression, modulation, delay, reverb, utility) |
| `catalog_list_fx_by_category` | List effect modules in one category (e.g. `delay`, `reverb`) |
| `catalog_list_block_presets` | List factory presets for one effect |
| `catalog_list_module_params` | Describe a module's parameters (kind, range, unit, options) |
| `translate_amp` / `translate_cab` / `translate_mic` | Hardware → device model |
| `design_rig` | Translate, order, write `.rig` + HTML report (serial or parallel chain); assign the 4 stomp switches with `footswitches` (toggle or scene) |
| `create_setlist` | Bind several `.rig` files into a device `.setlist` for songs that need multiple chains |
| `render_report` | HTML report for an existing `.rig` |
| `rig_decode` | Decode a `.rig` into chain + mixer (levels/pans/delay) + parameter values |
| `estimate_rig_level` | Estimate a rig's output level and recommend a RigVolume for a target level |

Example agent workflow: list amps → translate the song's amp → `design_rig` with
effects → read the report → tweak by re-running `design_rig` with parameter
overrides or by decoding and fixing an existing file.

## Signal chain & parallel routing

The full agent-facing guide lives in `internal/docs/agent-guide.md` and is
embedded in the binary — agents read it via the `get_guide` tool. In short, the
Gigboard's 11 chain slots can split into **two parallel paths**; the `Routing`
field takes exactly three values across the 293 device backups:

| `Routing` | Topology | Slot layout (1–11) |
| --- | --- | --- |
| `S` | Serial | 11 slots, one path |
| `SPS-1` | Serial → parallel → serial | 3 shared → path A (3) + path B (3) → 2 shared |
| `PS-1` | Parallel from the input → serial | path A (3) → path B (4–5) → shared remainder |

Key constraints: **`SPS-1`** has fixed split/merge points (3 shared slots,
3+3 parallel, 2 shared); a second amp is just another `Amp` module (`Amp` vs
`Amp 2`, same model = same amp on both channels); a single `Amp` in the shared
prefix feeds both paths (wet/dry/wet). Each section has a hard slot budget the
builder enforces. In `design_rig` pass `routing`, `amp2`, `cab2`, `mic2`,
`path_a_fx`, `path_b_fx` (CLI: `--routing`, `--amp2`, …).

### Footswitches

The four stomp switches (FS5–FS8) can each turn a module on/off. Assign them
with `design_rig`'s `footswitches` argument (CLI: `--footswitches`), an ordered
list of up to four `{"module": "...", "operation": "On"}` entries mapped to
FS5, FS6, FS7 and FS8. `module` is the module **instance name** exactly as it
appears in the chain (`Wham`, `Green JRC-OD`, `Amp 2`); `operation` defaults to
`"On"` (toggle on/off). A module not in the chain — or a fifth switch — is
rejected.

A **Scene** switch (`"mode": "scene"`) recalls a multi-block on/off snapshot in
one press: give it a `label` and list the blocks the scene turns `on` and `off`
(blocks not listed keep their current state):

```sh
guitar-modeler-mcp design --name "Always" --amp "67 Black Duo" \
  --fx '[{"type":"S1 Drive"},{"type":"Green JRC-OD"},{"type":"BBD Delay"}]' \
  --footswitches '[
    {"module":"Green JRC-OD","mode":"scene","label":"DRIVE",
     "scene":{"on":["S1 Drive","Green JRC-OD"],"off":["BBD Delay"]}},
    {"module":"BBD Delay","mode":"scene","label":"CLEAN",
     "scene":{"on":["BBD Delay"],"off":["S1 Drive","Green JRC-OD"]}}]'
```

The scene writes each slot's state directly into the `.rig` (0 = no change,
1 = on, 2 = off), matching what the device's own scene editor produces.

### Songs with multiple sounds

One song with incompatible chains (e.g. clean, drive, solo) is best handled as
**several rigs bound into a setlist** — design each rig, then bind them:

```sh
guitar-modeler-mcp design --name "Song Clean" --amp "65 Black SR" --out <card>/Rigs
guitar-modeler-mcp design --name "Song Drive" --amp "68 Plexiglas 50W" --out <card>/Rigs
guitar-modeler-mcp setlist --name "Song" --out <card>/Setlists <card>/Rigs/*.rig
```

Copy `Rigs/` and `Setlists/` onto the Gigboard and the whole song travels as one
bank. Scenes (one rig, blocks toggled) suit variations of the *same* chain;
setlists suit chains that must be rebuilt.

The builder **validates every parameter** against the device's specifications
(extracted from `headrush-desktop/renderer/config/modules/*.ts` plus the
backup-derived catalog): unknown parameter names, out-of-range numbers and
invalid enum options are rejected with a clear message, so an invalid `.rig` is
never written. On top of that, a **plausibility check** refuses to write a rig
whose estimated output level would be very loud (> +20 dB) or effectively muted
(amp master 0%), pointing out the offending level and how to fix it; `output
level`, `input gain` and `cab out gain` are all range-capped. Regenerate the
extracted spec with
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

# guitar-modeler-mcp — agent guide

You are designing guitar presets for hardware modelers. Four device families
are supported:

- **HeadRush Gigboard** — preset is a `.rig` file (JSON). Design with
  `design_rig`.
- **Mooer** (GE150 Pro Li, GE200, GE150, GE100 Pro) — a fixed nine-module chain
  (FX → DS/OD → AMP → CAB → NS → EQ → MOD → DELAY → REVERB). List devices with
  `device_list`, browse models with `mooer_catalog_list_*`, and design with
  `mooer_design`. Only the file-capable models — GE150 Pro Li, GE200, GE100 Pro
  — write a `.mo` file; the classic **GE150 is card-only** (no `.mo`, just the
  printable HTML **setup card**). Mapped parameter values are **neutral
  defaults** (raw 0–255, 128 = noon): the two devices scale knobs differently,
  so source positions are not copied. Setup cards and reports are named
  `<preset>.<device>.html` (e.g. `Brown Sound.ge200.html`), while preset files
  keep the terse `<preset>.mo` / `.rig` / `.tsl` scheme.
- **BOSS Waza Air** — a wireless headphone amp. Preset is a BOSS TONE STUDIO
  backup (`.tsl`): a named set of one or more patches, each a 2335-byte binary
  record stored as hex under `data[0][].paramSet["User%Patch"]`. The record is
  the Katana dense patch layout, so amp (type/gain/volume/bass/middle/treble/
  presence), booster (type/drive/tone/level), mod/fx (type), delay (type/time/
  feedback/level), reverb (type/level) and the spatial settings — POSITION
  (gyro SURROUND/STATIC/STAGE), AMBIENCE (STUDIO/STAGE) and MODE
  (DELAY/DLY+REV/REVERB) — are read and written at their Katana offsets.
  Amp gain uses the Katana scaling (stored = 20 + 0.8×gain); a requested delay
  switches the second delay block off. Browse models with
  `waza_catalog_list_*`, design with `waza_write_tsl` (writes a backup from the
  built-in template patch with the chosen tone applied) and `waza_setup_card`
  (a printable card), and read a backup's decoded patches with
  `waza_read_tsl`. An **XSONIC AIRSTEP BW** foot controller turns the six
  channel memories (CH 1–6, the Waza Air's "scenes") and the effect blocks into
  hands-free footswitches; list its four layouts with `waza_catalog_list_modes`
  and print one on the card with `waza_setup_card`'s `airstep_mode` (1–4).
- **Yamaha THR** (THR-II, THR10, THR10C, THR10X) — a desktop practice amp with
  no preset file format, so the only output is a printable setup card. The
  THR-II amp selector is a grid of eight types (CLEAN, CRUNCH, LEAD, HI GAIN,
  SPECIAL, BASS, ACOUSTIC, FLAT) × three modes (CLASSIC, BOUTIQUE, MODERN) —
  24 positions, each with Yamaha's official description plus a
  community-sourced "inspired by" real amp. THR-II also models 16 cabinets
  (Brown 4x12, American 1x12, California 1x12, …), the EFFECT knob
  (CHORUS, FLANGER, PHASER, TREMOLO), two ECHO delay types (Tape, Digital
  Delay) and four REVERB types (Plate, Hall, Spring, Room), plus app-only
  COMPRESSOR and NOISE GATE. Browse with `thr_catalog_list_*` and design with
  `thr_setup_card`. The legacy
  THR10/THR10C/THR10X amp lists are partial (community reference) and have no
  cabinet list.

Every parameter you pass is validated before a file is written, so an invalid
preset is never produced. `design_rig` is Gigboard-only; Mooer presets go
through `mooer_design`, Waza Air presets through `waza_write_tsl`, THR cards
through `thr_setup_card`, and cross-device conversion through `map_preset`.

## Tool contract

- **Writing tools.** `design_rig` (Gigboard `.rig` + `.html` report),
  `mooer_design` (Mooer `.mo` + setup card), `waza_write_tsl` (Waza Air
  `.tsl`), `waza_setup_card` (Waza Air `.html` card), `thr_setup_card` (THR
  `.html` card), `render_setup_card` (card from a `.mo`) and `map_preset`
  (cross-device conversion) write files. Every catalog/translate tool
  (`search_catalog`, `catalog_list_*`, `translate_amp/cab/mic`, `get_guide`,
  `get_fx_placement`, `catalog_list_module_params`, `mooer_catalog_list_*`,
  `waza_catalog_list_*`, `thr_catalog_list_*`, `device_list`, `waza_read_tsl`)
  returns its answer inline as JSON text — there are no files to open
  afterwards.
- **Never read source code** (this project's, the MCP's, or the desktop app's).
  The catalog tools are the complete interface to the device's models and
  their parameters; digging into `.go`/`.ts` files is a dead end.
- **One concept per query.** `translate_*` and `search_catalog` match one
  description at a time; "Marshall Plexi clean edge of breakup tweed" mixes
  four amp characters and returns noise. Run separate queries (or filter
  `catalog_list_amps` by one keyword) and pick from the results.
- **If a tool argument seems missing, trust `get_guide` and just try it.** The
  server serves `get_guide` fresh on every call — it always documents the
  current schema — while the client's tool list can be a stale snapshot from
  session start. Likewise `rig_decode` re-reads the `.rig` file and reports the
  actual `footswitches` on every call, so never guess hardware assignments from
  memory: decode the file and read them.

## Workflow discipline

This is a small, well-bounded task: pick models, call `design_rig`, verify.
Keep it tight and don't spiral:

- **Decide, don't deliberate.** One `search_catalog`/`translate_*` round, pick
  the first sensible model, move on. Don't re-derive the same amp/pedal choice
  over and over in prose — the user will correct a wrong pick faster than you
  can pre-empt it.
- **One `design_rig` call, then `rig_decode` to verify.** Don't re-plan the
  chain between calls; the tool's reply already reports the result.
- **When unsure about a capability, try it once.** A parameter or footswitch
  the schema doesn't advertise will either work or be rejected with a clear
  message — testing it costs one turn, reasoning about it costs ten.
- **Don't re-read `get_guide` or re-list catalogs mid-task.** They are stable
  for the session; re-calling them is dead time.

## Tools and workflow

1. `search_catalog` — fuzzy-search every amp, cab, mic and effect by device name
   or the real hardware it emulates (`modeled_after`), in both directions:
   "JCM800" finds "82 Lead 800 100W" and "Tube Screamer" finds "Green JRC-OD".
2. `catalog_list_amps` / `translate_amp` — find the amp model for the tone. Each
   amp lists the real hardware it emulates (`modeled_after`) and a `gain`
   character (`clean`, `edge of breakup`, `crunch`, `high gain`, `bass`). Match
   the character to the song: a clean song needs a `clean` amp (search
   `query: "clean"`), a lead song a `crunch`/`high gain` one — don't use a lead
   channel for a clean part.
3. `catalog_list_cabs`, `catalog_list_mics` — pick a cabinet and microphone.
4. `catalog_list_fx_categories` then `catalog_list_fx_by_category` — browse
   effects by category (see below) and their `capabilities`. To find an effect
   by what it does (e.g. `query: "pitch shift"` or `query: "reverb"`), use
   `catalog_list_fx` with a query instead of listing everything.
5. `catalog_list_module_params` — read a module's editable parameters, ranges
   and enum options before setting them. Pass a `types` list to describe several
   modules in one call.
6. `get_fx_placement` — where each effect category goes (before vs after the
   amp) and how many slots each chain layout offers, so you know what fits.
7. `design_rig` — resolve everything and write the `.rig` + an HTML report.
   **Assign the hardware controls in the same call** (see Footswitches below):
   a wah, whammy or solo-boost that the player toggles must be put on a stomp
   switch, otherwise the rig is unplayable as a stompbox. The tool's reply
   lists the assigned switches (or warns that none are assigned) so you can
   tell at a glance whether the controls are wired up.
8. `rig_decode` / `render_report` — inspect or re-report an existing preset.
9. `estimate_rig_level` — check a rig's net output level and the RigVolume that
   reaches a target. Default gain staging: input 0 dB → amp master 50% (−6 dB)
   → cab 0 dB → output **+6 dB** (the designer's default) ≈ 0 dB net. For more
   **drive** raise the amp `Gain` (or the drive pedal's `Drive`) — raising
   `Master`/`output_level` only makes it louder, not more overdriven.

## Effect categories

Effects are grouped into eight categories, mirroring the standard HeadRush
effect grouping: `distortion`, `dynamics`, `eq`, `expression`, `modulation`,
`delay`, `reverb`, `utility`. List them with `catalog_list_fx_categories` and
the modules of one with `catalog_list_fx_by_category`.

## Signal chain & parallel routing

The Gigboard's 11 chain slots are not always one straight line — the chain can
split into **two parallel paths**. The topology is the `Routing` field, which
takes exactly three values:

| `Routing` | Topology | Slot layout (1–11) |
| --- | --- | --- |
| `S` | Serial | 11 slots, one path |
| `SPS-1` | Serial → parallel → serial | 3 shared → path A (3) + path B (3) → 2 shared |
| `PS-1` | Parallel from the input → serial | path A (3) → path B (4–5) → shared remainder |

Constraints:

- **`SPS-1` (dual amp).** The split and merge points are fixed: slots 1–3 are
  shared (pre-amp FX and/or a shared amp+cab), slots 4–6 are **path A**, slots
  7–9 are **path B**, slots 10–11 are shared (post FX). Each path holds at most
  **3** blocks, the prefix **3**, the suffix **2**.
- **Same amp on both channels vs two amps.** There is no switch: a second amp is
  just another `Amp` module. `Amp` (path A) and `Amp 2` (path B) may carry the
  **same** model (same amp on both channels) or **different** models. A single
  `Amp` in the shared prefix (slots 1–3) feeds *both* paths — the wet/dry/wet
  pattern.
- **`PS-1`** splits at the input: path A is the first **3** slots, path B the
  next **4–5**, then the merge and the remaining serial slots.
- **Path mixer.** `Para1Level`/`Para2Level` (dB, default −6),
  `Para1Pan`/`Para2Pan` (−100…100, default 0) and `ParaDelay` (ms, default 0)
  balance the two paths; pan −100/+100 hard-pans paths A/B for wet/dry/wet.
  Set them directly on `design_rig` (`para1_level`, `para2_level`, `para1_pan`,
  `para2_pan`, `para_delay`) — there is no need to edit the file afterwards.
- **Slot budget.** 11 slots total; a section that exceeds its budget is
  rejected rather than silently overflowing.

In `design_rig`, pass `routing: "SPS-1"` with `amp2` (and optional `cab2`,
`mic2`) for a dual-amp rig, or `routing: "SPS-1"` without `amp2` and
`path_a_fx`/`path_b_fx` for a shared-amp split. Balance and pan the two paths
with `para1_level`/`para2_level`, `para1_pan`/`para2_pan` and `para_delay`.

### A/B switching

To alternate between two tones on the Gigboard:

- **Toggle the modules that differ.** In a split (`SPS-1`/`PS-1`) or serial
  rig, assign a footswitch to each module you want on/off — e.g.
  `footswitches: [{"module":"Amp"},{"module":"Amp 2"}]` toggles each amp.
  This is the method `design_rig` emits today.
- **Amp/cab Doubling** (device): one Amp block can carry two channels —
  `Doubling`/`DoubleStates`, `Type`/`Type2`, `Master`/`Master2`, … — the
  tightest two-amps-in-one-block switch. Not yet emitted by `design_rig`.
- **Scene footswitches**: a footswitch can be set to Scene mode to recall a
  multi-block on/off snapshot in one press. Specify which blocks each scene
  turns on and off with the `scene` object (see below).

The Gigboard's mixer has **no "Solo A/B"** parameter — that belongs to newer
HeadRush hardware (Flex Prime / Prime). Do not try to assign a footswitch to a
mixer "Solo"; use module toggles, Doubling or Scenes instead.

## Footswitches

The Gigboard has four stomp switches (FS5–FS8) that can each turn a module on
and off (or control a module-specific parameter). When a rig needs a switch —
for example a **whammy toe switch** or a **boost on/off** — assign it with
`design_rig`'s `footswitches` argument, an ordered list of up to four entries
mapping to FS5, FS6, FS7 and FS8:

```json
"footswitches": [{"module": "Wham"}, {"module": "Green JRC-OD"}]
```

Each entry's `module` is the module **instance name** exactly as it appears in
the chain (`Wham`, `Green JRC-OD`, `Amp`, `Amp 2` — repeated modules get a
`" N"` suffix). The default `operation` is `"On"` (toggle the module on/off);
pass a different `operation` to control a module-specific parameter instead.
A module that is not in the chain is rejected, and you can never assign more
than four switches.

A **Scene** switch recalls a saved snapshot of which blocks are on and off in
one press. Set `"mode": "scene"`, give it a `label` for the screen, and list
the blocks the scene turns `on` and `off` (any block not listed keeps its
current state):

```json
"footswitches": [
  {"module": "Green JRC-OD", "mode": "scene", "label": "DRIVE",
   "scene": {"on": ["S1 Drive", "Green JRC-OD"], "off": ["BBD Delay"]}},
  {"module": "BBD Delay", "mode": "scene", "label": "CLEAN",
   "scene": {"on": ["BBD Delay"], "off": ["S1 Drive", "Green JRC-OD"]}}
]
```

Every scene turns on, turns off, or leaves alone each of the 11 chain slots
(0 = no change, 1 = on, 2 = off) — exactly what the device's scene editor
writes.

**Order matters: put the most important switches first.** The first two entries
land on buttons 1 and 2 (FS5/FS6), which stay dedicated to the patch in every
button mode. Buttons 3 and 4 (FS7/FS8) are repurposed for bank switching in the
hybrid button mode, so a switch the player hits mid-song (a whammy toe switch,
a solo boost) must be in the first two slots.

## Songs with multiple sounds (scenes vs setlists)

When one song needs several distinct tones (clean, drive, solo), you have two
tools. **Ask the user which they want when it is not obvious** — the wrong
guess wastes a round trip:

- **Scenes** — one rig, one chain. A Scene footswitch turns several blocks on
  and off at once. Use this when the sounds are variations of the *same chain*
  (same amp/cab, different pedals or a boost). Design the rig with scene
  footswitches (see above); the on/off snapshot is written into the `.rig`, so
  nothing needs editing on the device afterwards.
- **A setlist of rigs** — several full `.rig` files stepped through as a bank.
  Use this when the sounds need *incompatible chains* (different amps/cabs that
  won't all fit in 11 slots, or a chain that must be rebuilt). Design each rig
  with `design_rig`, then bind them with `create_setlist`.

Folder layout for a copy-ready card — write rigs into `Rigs/` and the setlist
into `Setlists/`, then copy both onto the Gigboard:

```
design_rig ... --out <card>/Rigs     # one call per sound
create_setlist --name "Song" --out <card>/Setlists <card>/Rigs/*.rig
```

`create_setlist` reads each `.rig` file's id and name, so the setlist always
points at the exact rigs you just wrote.

## Reading a rig (reverse-engineering)

To analyze an existing `.rig` file, pass its path to `rig_decode`. It returns a
structured summary instead of raw JSON:

- `routing` + `slots` — the topology and the full 11-slot layout (including
  `Empty Slot` placeholders), so you can see exactly where the split/merge
  points are.
- `mixer` — the parallel-path balance (`para1_level`, `para2_level`,
  `para1_pan`, `para2_pan`, `para_delay`).
- `modules` — every module in chain order with its `name`, `on` state and
  effective `params` (amp/cab models, pedal settings).
- `footswitches` — the FS5..FS8 assignments, if any.

Workflow to reproduce or tweak a preset:

1. `rig_decode` the file and note the amp/cab/mic models and each effect with
   the params that matter.
2. Rebuild it with `design_rig`, passing those models/params (and the same
   `routing`, `footswitches` and mixer values) — or adjust them deliberately.
3. `render_report` turns any `.rig` into a readable HTML report of the chain,
   params and switches, if you want a human-facing view.
4. `estimate_rig_level` on the existing file reports its net output level —
   useful before changing gain staging.

A `.rig` is plain JSON whose `content` field is an escaped second JSON document
(`{FootSwitch, Pedal1, Pedal2, data:{Patch}, info:{version}}`). Prefer
`rig_decode` over hand-parsing the raw JSON — it already resolves the
instance names, mixer and footswitches.

## Optimizing a rig

Once you have decoded a rig, improve it in this order:

1. **Level first.** `estimate_rig_level` with a target (default 0 dB) tells you
   how far off the rig is. Fix it with `output_level` (RigVolume), amp
   `Master`, or cab `OutGain` — but keep the summed level within −60…+20 dB or
   the build is refused.
2. **Match the amp to the part.** Compare the amp's `gain` character
   (`clean` → `high gain`) with the song. A lead-channel amp on a clean song is
   the most common mismatch — swap it and keep the cab/mic if they fit.
3. **Check placement.** Verify each effect sits on the right side of the amp
   (`get_fx_placement`): wah/boost/compressor before, time-based effects after.
   A delay in front of the amp muddies the tone.
4. **Wire the controls.** If the rig has a wah, whammy or boost but
   `footswitches` is empty, assign it (see Footswitches) — an unswitched
   whammy is unplayable.
5. **Trim or fill the chain.** Drop modules that fight the tone (two stacked
   drives where one suffices), add what's missing (a reverb for a spacey part,
   a boost for a solo).
6. **Rebuild through `design_rig`** with the corrected values. It re-validates
   every parameter, so a bad edit is rejected rather than written.

## Validation

Every parameter you set is validated against the device's specifications:
unknown parameter names, out-of-range numbers, invalid enum options and unknown
amp/cab/mic models are rejected with a clear message.

A plausibility check also runs before a rig is written: a rig whose estimated
output level would be very loud (above +20 dB) or effectively muted (amp master
at 0%) is refused, with a hint on what to lower or raise. `design_rig`'s
`output_level` (and the `input_gain`, amp `Master` and cab `OutGain`) are all
capped, so an accidental `output_level: 100` is rejected rather than written.

## Mooer devices & cross-device mapping

Mooer devices use a **fixed nine-module chain** (FX, DS/OD, AMP, CAB, NS, EQ,
MOD, DELAY, REVERB) — there is no free slot layout, no parallel paths and no
dual-amp config. Each module holds one effect, selected by its `effect_type`
index into that module's own list.

1. `device_list` — which devices are supported and whether each supports
   preset **file exchange** (`file_ext`) or only a **printable setup card**
   (`file_exchange: false`).
2. `mooer_catalog_list_amps` / `mooer_catalog_list_cabs` /
   `mooer_catalog_list_fx` — browse a device's models
   (`model: ge150pro|ge200|ge150|ge100pro`); each row carries the real
   hardware it emulates (`inspired_by`).
3. `mooer_design` — resolve amp/cab/effects and write the preset. Arguments:
   `model`, `name`, `amp` (model name or a real-hardware description, e.g.
   `"Marshall JCM800"`), optional `cab`, and `fx` — an ordered list of
   `{"module": "od|fx|mod|delay|reverb|ns|eq", "type": "...", "enabled": bool}`.
   It always writes an `.html` setup card; file-capable models also write a
   `.mo` file.
4. `render_setup_card` — turn an existing `.mo` file into the printable card.
5. `map_preset` — convert a preset between devices. A `.rig` maps to a Mooer
   preset (GE150 Pro Li) plus a setup card; a `.mo` maps back to a Gigboard
   `.rig`.

The setup card is the deliverable for models without file exchange (the
non-pro GE150): it lists every module's effect, on/off state, the real hardware
it emulates, and the raw parameter values, so the player can dial it in by hand.

## Disclaimer

All trademarks, logos and brand names are the property of their respective
owners. Use of company, product and service names is for identification only and
implies no endorsement; this project is not affiliated with HeadRush or any
referenced brand.

This software, its MCP tools, and any output produced by AI agents or the MCP
server are provided "as is", without warranty of any kind. Presets, setup cards
and other generated files are for reference only: verify them before use. The
authors and contributors accept no liability for any damage to hardware, loss
of data, or other consequences arising from the use or misuse of this software
or of output produced by AI agents or the MCP server.

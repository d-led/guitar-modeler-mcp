# HeadRush Gigboard — agent guide

You are designing sound presets for the **HeadRush Gigboard**. A preset is a
`.rig` file: one line of JSON whose `content` field is a second JSON document
describing the signal chain (the `Patch`). Use these tools to discover models
and to write `.rig` files; every parameter you pass is validated before a file
is written, so an invalid preset is never produced.

## Tool contract

- **Nothing writes files except `design_rig` and `render_report`.** Every
  catalog/translate tool (`search_catalog`, `catalog_list_*`,
  `translate_amp/cab/mic`, `get_guide`, `get_fx_placement`,
  `catalog_list_module_params`) returns its answer inline as JSON text — there
  are no files to open afterwards. `design_rig`'s reply tells you the `.rig`
  and `.html` paths it wrote.
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
9. `estimate_rig_level` — check a rig's output level (the default amp master of
   50% is −6 dB, so a fresh rig is usually ~6 dB quiet). Set `output_level` on
   `design_rig` (or the recommended RigVolume) to compensate.

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
- **Scene footswitches** (device): a footswitch can be set to Scene mode
  (`ModeNew="Scene"`) to recall a module on/off snapshot. Not yet emitted by
  `design_rig`.

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

**Order matters: put the most important switches first.** The first two entries
land on buttons 1 and 2 (FS5/FS6), which stay dedicated to the patch in every
button mode. Buttons 3 and 4 (FS7/FS8) are repurposed for bank switching in the
hybrid button mode, so a switch the player hits mid-song (a whammy toe switch,
a solo boost) must be in the first two slots.

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

## Disclaimer

All trademarks, logos and brand names are the property of their respective
owners. Use of company, product and service names is for identification only and
implies no endorsement; this project is not affiliated with HeadRush or any
referenced brand.

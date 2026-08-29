# HeadRush Gigboard — agent guide

You are designing sound presets for the **HeadRush Gigboard**. A preset is a
`.rig` file: one line of JSON whose `content` field is a second JSON document
describing the signal chain (the `Patch`). Use these tools to discover models
and to write `.rig` files; every parameter you pass is validated before a file
is written, so an invalid preset is never produced.

## Tools and workflow

1. `catalog_list_amps` / `translate_amp` — find the amp model for the tone (each
   amp lists the real hardware it emulates, `modeled_after`, and its
   `capabilities`).
2. `catalog_list_cabs`, `catalog_list_mics` — pick a cabinet and microphone.
3. `catalog_list_fx_categories` then `catalog_list_fx_by_category` — browse
   effects by category (see below) and their `capabilities`. To find an effect
   by what it does (e.g. `query: "pitch shift"` or `query: "reverb"`), use
   `catalog_list_fx` with a query instead of listing everything.
4. `catalog_list_module_params` — read a module's editable parameters, ranges
   and enum options before setting them. Pass a `types` list to describe several
   modules in one call.
5. `design_rig` — resolve everything and write the `.rig` + an HTML report.
6. `rig_decode` / `render_report` — inspect or re-report an existing preset.

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

## Validation

Every parameter you set is validated against the device's specifications:
unknown parameter names, out-of-range numbers, invalid enum options and unknown
amp/cab/mic models are rejected with a clear message.

## Disclaimer

All trademarks, logos and brand names are the property of their respective
owners. Use of company, product and service names is for identification only and
implies no endorsement; this project is not affiliated with HeadRush or any
referenced brand.

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
  printable HTML **setup card**). The GE200 and GE150 Pro Li `.mo` byte layouts
  differ; the tools detect and write each device's own format automatically, so
  a preset designed with `device: "ge200"` loads on the GE200.

  **Dial in the knobs, don't leave them at noon.** `mooer_design` accepts raw
  parameter values (0–100, 50 = noon) via `amp_params`, `cab_params`, and each
  `fx` item's `params` object. Canonical keys per module: amp `gain`/`bass`/
  `mid`/`treble`/`presence`/`master`; cab `mic`/`center`/`distance`/`tube`;
  drive `gain`/`tone`/`volume`; mod `rate`/`depth`/`level`; delay `level`/
  `feedback`/`time_ms`/`subdivision`; reverb `pre_delay`/`level`/`decay`/`tone`;
  noise gate `threshold`/`attack`/`release`; EQ `band1`..`band12`. Keys are
  case- and separator-insensitive (`"Time (ms)"` = `time_ms`). Convert a
  0–10 knob to raw with `value/10 × 100`; an EQ dB lift of +N is `50 + N/20×100`.
  Cross-device `map_preset` still writes neutral 50s (the two devices scale
  knobs differently). Setup cards and reports are named
  `<preset>.<device>.html` (e.g. `Brown Sound.ge200.html`), while preset files
  keep the terse `<preset>.mo` / `.rig` / `.tsl` scheme.

  **Expression pedal is a device setting, not in the `.mo`.** A Wah or Volume
  in the FX module still needs the device's EXP pedal assigned to it by hand
  (the wah's `POSITION` follows the pedal); `mooer_design` dials the block's
  knobs but cannot write the pedal assignment into the preset. Say so when a
  wah/volume is part of the tone.
- **BOSS Waza Air** — a wireless headphone amp. Preset is a BOSS TONE STUDIO
  backup (`.tsl`): a named set of one or more patches, each a 2335-byte binary
  record stored as hex under `data[0][].paramSet["User%Patch"]`. The record is
  the Katana dense patch layout, so amp (type/gain/volume/bass/middle/treble/
  presence), booster (type/drive/bottom/tone/solo/level/direct mix), mod/fx
  (type **plus per-effect knobs**), delay (type/time/feedback/high cut/level/
  direct mix), reverb (type/time/pre-delay/level/direct mix), the noise
  suppressor (on/threshold/release) and the spatial settings — POSITION
  (gyro SURROUND/STATIC/STAGE + guitar position), AMBIENCE (STUDIO/STAGE +
  level) and MODE (DELAY/DLY+REV/REVERB) — are read and written at their
  Katana offsets. Amp gain uses the Katana scaling (stored = 20 + 0.8×gain);
  a requested delay switches the second delay block off.

  **Effect knobs.** Every MOD/FX effect exposes its editable parameters through
  `mod_params` / `fx_params` (a flat object of knob → number), keyed by
  canonical names such as `rate`, `depth`, `effect_level`, `direct_mix`,
  `manual`, `resonance`, `sustain`, `attack`, `threshold`, `release`,
  `feedback`, and the chorus `low_rate`/`high_rate` bands (the short `rate`/
  `depth`/`effect_level` names target the chorus's low band). **`*_direct_mix`
  is the DRY (unprocessed) signal level — 100 = unity, so leave it unset unless
  you explicitly want to attenuate the dry guitar; setting it to a low
  "effect level" value makes the whole patch very quiet.** Browse models
  with `waza_catalog_list_*`, design with `waza_write_tsl` (writes a backup
  from the built-in template patch with the chosen tone applied) and
  `waza_setup_card` (a printable card), and read a backup's decoded patches
  with `waza_read_tsl`. An **XSONIC AIRSTEP BW** foot controller turns the six
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
- **Neural DSP Quad Cortex** — full catalog, with the setup card as the real
  output. The model list and every knob's scale come from the device's own
  `ModelRepo.xml`, so each model carries a wire id and each parameter its real
  min/max and taper (`skew`). Browse with `qc_catalog_list_amps`/`_cabs`/`_fx`
  (names carry their ids and "based on" hardware), translate with
  `qc_translate_amp`/`_cab`, and inspect one model's knobs with
  `qc_list_model_params` (use the **screen units** it reports — GAIN 5 on a
  0..10 knob, dB and % values as shown).

  `qc_design` places a **serial chain** — amp, then cab, then the effects in
  the order you give — and writes a **self-contained HTML setup card** (every
  block and every knob, with values, so a person can dial the tone in by hand)
  plus a `.pb` reference archive. There are exactly three roles, and you must
  state them to the user:

  - **HTML card = the setup instructions.** This is what reproduces the tone.
  - **.pb = a reference archive** for saving and reloading the tone in this
    tool (`qc_decode_preset`, `qc_render_setup_card`). It is **not** a file the
    Quad Cortex imports — never claim it can be loaded onto the unit.
  - **`qc_usb` = live unit control, not a .pb upload.** It shells out to the
    user's `qcctl` (pyquadcortex), whose only subcommands are `version`,
    `recall --setlist --slot`, `scene --index` and `dump-preset --setlist
    --slot`: it reads the firmware version, recalls a preset **already in a
    slot on the unit**, switches scenes, and dumps the preset in a slot. It
    cannot upload a `.pb` — to put a tone on the unit, dial it in from the
    HTML card or place it in a slot with Cortex Control, then recall it. Ask
    the user first: it needs a USB-connected unit, `qcctl` installed, Cortex
    Control quit, and it can change the device — so only run it with
    `confirm: true` after they agree.

    Install `qcctl` once: `pip install pyquadcortex` (macOS also
    `brew install hidapi`).

    **Crucial prerequisite:** before any `qcctl` command the user must quit
    the official **Cortex Control** desktop application — it holds an
    exclusive lock on the USB interface and blocks `qcctl` while it runs.
    Wi-Fi on the device may stay on.

  The wire is free-form (4 lanes that split and merge), but `qc_design` covers
  the common single-lane case; for parallel/dual-amp rigs, describe the
  routing to the user. There is no private key anywhere: the `.pb` archive
  uses the device's public `KEY_MATERIAL` + serial, and `qc_usb` uses the
  unit's own protocol over USB-HID.

  **Expression pedals are not authored by `qc_design`.** A wah, pitch or filter
  block that is rocked by a foot pedal must be wired to EXP1/EXP2 on the unit
  itself — the card lists the block and its knobs, but the controller routing
  is a manual step. Say so when a tone depends on a swept block.

Every parameter you pass is validated before a file is written, so an invalid
preset is never produced. `design_rig` is Gigboard-only; Mooer presets go
through `mooer_design`, Waza Air presets through `waza_write_tsl`, THR cards
through `thr_setup_card`, Quad Cortex presets through `qc_design`, and
cross-device conversion through `map_preset`.

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
- **Hand off with a visual check.** Once the HTML report/setup card is written,
  pass the clickable report link to the user — the tool returns it as a
  `[filename](file://…)` markdown link, so present it as a link (don't flatten
  it to a raw path) — and tell them to eyeball it before deploying to the
  device: AI model, parameter and routing choices can be off, and the card is
  the human verification step against the actual hardware.

## HeadRush Gigboard — capabilities at a glance

- **11 chain slots**, three routings: `S` (serial), `SPS-1` (serial→parallel→
  serial), `PS-1` (parallel from the input). Serial is the default.
- **4 stomp switches (FS5–FS8)** + **2 expression pedals** (Pedal1, Pedal2).
  A wah/whammy needs a pedal (sweep) **and** a footswitch (on/off).
- **Scenes flip blocks on/off only — they never change a parameter value.** To
  switch a sound *per scene*, toggle whole blocks (two IRs, two boosts, a
  corrective EQ). `LastScene` marks the scene active at load (the first one).
- **A custom IR (`IR`/`IR (1024)`) replaces the cabinet** — pass it in `fx` and
  the designer drops the cab. Selector: `[directory](<folder>)[name](<file>)`,
  root = `[IR ROOT]`; `IR (1024)` is half the DSP.
- **Preset names are stored and displayed ALL CAPS** (the device's convention);
  put the human title in `note`.
- **Amp `Master` = loudness, `Gain` = drive.** Gain-stage with the amp, not the
  output: RigVolume is a final trim (designer default +6 dB compensates a 50%
  master).
- Bypassed blocks render **grey** in the report (chain, card badge, and the
  stomp button), so what starts off is visible at a glance.

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
   tell at a glance whether the controls are wired up. **Keep `name` short** —
   the Gigboard stores and displays preset names in **all caps** and ellipsizes
   them beyond 21 characters, so `design_rig` uppercases and truncates over-long
   names (and says so in its reply); put the full song/artist/character title in
   `note`, which lands on the HTML report instead (the `name` itself comes out
   as `UPPERCASE` — that's the device, not a bug).
   Dial the amp and cab knobs with `amp_params`/`cab_params` (flat objects
   keyed by the exact parameter names from `catalog_list_module_params` — e.g.
   `{"GainA": 62, "Master": 55}`; the amp uses `GainA`/`GainB`, not `Gain`,
   and toggles like `OnAxis` take booleans). FX knobs go in each `fx` item's
   `params` the same way.
8. `rig_decode` / `render_report` — inspect or re-report an existing preset.
9. `estimate_rig_level` — check a rig's net output level and the RigVolume that
   reaches a target. Default gain staging: input 0 dB → amp master 50% (−6 dB)
   → cab 0 dB → output **+6 dB** (the designer's default) ≈ 0 dB net. For more
   **drive** raise the amp `Gain` (or the drive pedal's `Drive`) — raising
   `Master`/`output_level` only makes it louder, not more overdriven.

   **The estimate is a relative hint, not a measurement.** It sums only the
   *known* stages — input gain, amp **Master** (power-amp volume), cab out
   gain, the parallel mixer and **RigVolume** — and deliberately leaves out
   the amp's preamp gain (`GainA`/`GainB`) and drive-pedal `Level`, so a clean
   amp and a high-gain amp at the same Master read identically even though the
   high-gain amp plays much louder. To make a rig louder **without overdrive**,
   raise the amp **Master** (loudness), not `Gain` (drive), and leave RigVolume
   near unity as a final trim — see the "Level first" step under *Optimizing a
   rig* below.

## Effect categories

Effects are grouped into eight categories, mirroring the standard HeadRush
effect grouping: `distortion`, `dynamics`, `eq`, `expression`, `modulation`,
`delay`, `reverb`, `utility`. List them with `catalog_list_fx_categories` and
the modules of one with `catalog_list_fx_by_category`.

## Impulse responses (custom IR)

A custom impulse response loads a third-party cab capture (OwnHammer, York
Audio, ML Sound Lab, …) instead of a stock cabinet. There are two blocks:
`IR` (2048-sample) and `IR (1024)` (truncated, half the DSP). List them with
`catalog_list_fx` (`query: "IR"`) and read their knobs with
`catalog_list_module_params`.

- **An `IR` effect replaces the cabinet** — pass it in the `fx` list and
  `design_rig` drops the Cab block (the chain becomes amp → IR). Don't also
  pass a `cab`: the signal would be filtered twice.
- The IR selection is the `IR` parameter, a string of the form
  `[directory](<folder>)[name](<file>)` — `<folder>` is the folder under
  `Impulse Responses/` on the device and `<file>` is the `.wav` name without
  the extension; `[directory]([IR ROOT])` is the root folder. Example:
  `{"type": "IR", "params": {"IR": "[directory](YorkMixes)[name](YA MES 212 V30 Mix 01)"}}`.
- The other knobs are `Gain` (dB trim), `HiCut` (Hz), `LoCut` (Hz) and `Mix`
  (%); the block also carries a `Doubling`/stereo second set (`IR2`, `Gain2`,
  …) like Amp and Cab, which the builder writes with safe defaults.
- `estimate_rig_level` and the build-time loudness guard count the IR's `Gain`
  (scaled by `Mix`) — a +6 dB IR reads 6 dB louder, and a large positive `Gain`
  can trip the "very loud" refusal just like a hot cab `OutGain`.
- **Scenes cannot change the IR file** — a scene only flips blocks on/off. To
  switch between two IRs per scene, place **two IR blocks** (each loaded with
  one file) and let the scenes toggle which one is on.

## Choosing effects (browse the class, fit the chain)

Don't grab the first `search_catalog`/`translate_*` hit and call it done. For a
category that carries the tone — especially **distortion** — list the whole
class and pick for the *chain*, not just for the real pedal it emulates:

1. `catalog_list_fx_by_category` (e.g. `category: "distortion"`) and read each
   module's `modeled_after`, `confirmed` flag and `capabilities`. Two pedals can
   both be "inspired by" a Tube Screamer yet sit very differently in a rig.
2. **One overdrive is usually not enough for a singing lead.** A lead tone is
   gain-staged, not a single pedal: stack a low-gain boost/OD (e.g. `Green JRC-OD`
   at low `Drive`, high `Level`) into a higher-gain drive, or into the amp's own
   `Gain` — the amp's `gain` character decides how much help it needs. A `clean`
   amp needs more staging than a `crunch`/`high gain` amp.
3. **Match the effect to the chain, not to the song's gear list.** Decide what
   the block must *do* (clean boost, edge-of-breakup push, mid-hump, fuzz wall,
   solo level bump) and choose the model that does that job where it sits
   (before the amp it pushes the amp; after the amp it shapes the tone).
4. **Occasional effects start off.** A modulation effect (phaser, chorus,
   flanger, tremolo) or filter that only colours certain sections of the song
   should be *configured but bypassed*: add it with its dialled params and
   `"enabled": false`, then wire it to a footswitch (or a scene) so the player
   can bring it in. Turn it on by default only when the user wants it always-on
   or explicitly proposes switching it — if it's unclear, ask.

The amp's `gain` character (`clean` → `high gain`, from `catalog_list_amps`)
tells you how much drive the chain already has: a `high gain` amp needs at most
a clean boost, while a `clean` amp may need two stacked gain stages to reach a
lead sound.

Distortion effects also carry a `gain` character — `boost` → `overdrive` →
`distortion` → `fuzz` (plus `bass` and `bitcrusher`) — shown by
`catalog_list_fx`, `catalog_list_fx_by_category` and `search_catalog`. Don't
default to `Green JRC-OD`: it is a low-gain TS-808 **overdrive**, and a
singing/high-gain lead usually wants a **distortion** (`Black OP` Pro Co Rat,
`DC Distort`, `D1 Dist`, `MX Dist`) or a drive→distortion stack. Match the
pedal's `gain` to the part exactly as you match the amp's.

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

A **Scene** switch is how you **turn several blocks on and off at once** with a
single stomp: one press recalls a saved snapshot of which of the 11 chain slots
are on and off. Set `"mode": "scene"`, give it a `label` for the screen, and
list the blocks the scene turns `on` and `off` (any block not listed keeps its
current state). The `module` field still names a module in the chain — it
anchors the switch and its on-screen colour, but the *behaviour* comes from the
`scene.on`/`scene.off` lists, which can reference any blocks in the chain, not
just that one module:

```json
"footswitches": [
  {"module": "Green JRC-OD", "mode": "scene", "label": "LEAD",
   "scene": {"on": ["Green JRC-OD", "Tape Echo"], "off": ["Chorus"]}},
  {"module": "Green JRC-OD", "mode": "scene", "label": "CLEAN",
   "scene": {"on": ["Chorus"], "off": ["Green JRC-OD", "Tape Echo"]}}
]
```

Here `LEAD` switches the drive *and* the delay on and the chorus off in one
press; `CLEAN` does the inverse. Every scene turns on (1), turns off (2), or
leaves alone (0) each of the 11 chain slots — exactly what the device's scene
editor writes — so a scene can flip any combination of blocks in the chain.

**Order matters: put the most important switches first.** The first two entries
land on buttons 1 and 2 (FS5/FS6), which stay dedicated to the patch in every
button mode. Buttons 3 and 4 (FS7/FS8) are repurposed for bank switching in the
hybrid button mode, so a switch the player hits mid-song (a whammy toe switch,
a solo boost) must be in the first two slots.

## Expression pedals

A wah, whammy or volume pedal is **unplayable without an expression pedal** — a
footswitch only toggles it on/off, it does not sweep it. Wire the sweep with
`design_rig`'s `pedals` argument (Pedal1, then Pedal2) and **always set the
sweep range** (`min`/`max`):

```json
"pedals": [{"module": "Black Wah", "param": "Pedal", "min": 0, "max": 100}]
```

- `param` is the controller target: `"Pedal"` (wah sweep), `"Pitch"` (whammy),
  `"Volume"` (volume pedal).
- `min`/`max` is the controller range — 0–100 is the full sweep; a narrower
  range keeps a whammy in a usable window. Omit `max` to default to 100.
- `design_rig` auto-assigns the first wah/whammy/volume in the chain to Pedal1
  (0–100) when `pedals` is omitted — but don't rely on it: state the pedal and
  its range explicitly, and pair it with a `footswitches` entry that toggles
  the module on/off (a wah is usually **off by default**, brought in by its
  stomp switch).

## Songs with multiple sounds (scenes vs setlists)

When one song needs several distinct tones (clean, drive, solo), you have two
tools. **Ask the user which they want when it is not obvious** — the wrong
guess wastes a round trip:

- **Scenes** — one rig, one chain. A Scene footswitch turns several blocks on
  and off at once in a single press. Use this when the sounds are variations
  of the *same chain* (same amp/cab, different pedals or a boost) — the
  classic clean↔lead: the clean scene switches the drive off and the chorus
  on, the lead scene switches the drive (and maybe a delay) on and the chorus
  off. Design the rig with scene footswitches (see Footswitches above); the
  on/off snapshot is written into the `.rig`, so nothing needs editing on the
  device afterwards. Scenes flip *on/off state only* — they do not change
  parameter values, so a scene cannot switch the amp's gain or the delay's
  time; for that you need two rigs (see setlists).
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

The HTML report visualises on/off state: a block that starts bypassed is
greyed in the chain picture, carries an `off` badge on its card, and its
stomp-switch button is grey instead of green (only for on/off toggles — scene
buttons stay green). Reading a rig this way shows at a glance which pedals
start off.

For a programmatic edit in the repo (not via MCP): decode a `.rig` with
`RigFile.Decode()` into the typed `Content` model, change any node or
parameter, then `RigFile.SetContent(content)` re-encodes and `Write` saves —
untouched sections (the FootSwitch/Pedal blobs) survive byte-for-byte, and
round-trip tests in `internal/rig/roundtrip_test.go` cover the no-op,
all-parameters-changed and block-toggle cases.

A `.rig` is plain JSON whose `content` field is an escaped second JSON document
(`{FootSwitch, Pedal1, Pedal2, data:{Patch}, info:{version}}`). Prefer
`rig_decode` over hand-parsing the raw JSON — it already resolves the
instance names, mixer and footswitches.

## Optimizing a rig

Once you have decoded a rig, improve it in this order:

1. **Level first.** `estimate_rig_level` with a target (default 0 dB) tells you
   how far off the rig is — but treat it as a *relative* hint, because it
   ignores amp preamp gain (see above). Balance in this order:
   - **Amp `Master` is the loudness control, `Gain` is the drive control.**
     Raise `Master` to get louder *without* overdriving; raise `Gain` only to
     add drive. This is the most common fix — an under-driven clean amp
     (Master left at 50%) is why a rig comes out quiet and tempts a big
     RigVolume boost.
   - **A clean amp needs a higher Master than a high-gain amp.** To reach the
     same loudness without overdrive, put a clean/bass amp's Master around
     70–80% and a high-gain amp's around 50–60% — the high-gain amp's preamp
     gain makes it naturally much hotter.
   - **Leave RigVolume near unity as the final trim.** Crank `output_level`
     only for the last dB or two; a rig that needs +15 dB of RigVolume has a
     gain-staging problem (usually a quiet clean amp), not a trim problem.
   - **Aim for a healthy meter, not a specific number.** The goal is green
     meters that swing close to — but never into — the red on the loudest
     strum.
   Keep the summed estimate within −60…+20 dB or the build is refused; fix it
   with amp `Master` or cab `OutGain` before reaching for `output_level`.
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

## Delay & reverb mix (avoiding mud)

The factory block defaults are wet-heavy — `Tape Echo` defaults to `Mix` 40,
`AIR Reverb` to `Mix` 40 — so two or three time-based effects stacked at their
defaults collapse into a cloudy tail. **Always set `Mix` explicitly; don't
leave the factory default.** Start low and push up only if the part calls for
it:

| Effect | `Mix` range | Notes |
| --- | --- | --- |
| Delay (`Tape Echo`, `BBD Delay`, `Air Delay`, …) | **15–25** | The repeats should be *felt*, not heard as a separate echo; above ~30 the dry attack smears. Keep `Feedback` ≤ ~30 unless you want obvious repeats. |
| Reverb (`AIR Reverb`, `Eleven Reverb`, …) | clean **35–45**, dense/high-gain **15–30** | A clean or ambient part can sit wetter; a dense high-gain tone wants less room, or the reverb buries the gain. |
| Modulation (`Chorus`, `Flanger`, `Phaser`) | **15–25** | Low depth+mix reads as width; a high mix wobbles the pitch of the dry note. |

Rules of thumb:

- **Lower the mix for high-gain amps.** Distortion already fills the spectrum,
  so a driven tone wants *less* delay and reverb than a clean one. A lead patch
  at delay 24 + reverb 21 stays articulate; the same values on a clean patch
  can go delay 25 + reverb 42.
- **`Tails` stays on.** It only controls whether the trail rings out when the
  block is bypassed — it does not change the wet level.
- **If the dry attack is masked, lower `Mix`.** That is the fix, not an EQ
  or a tone knob.

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

## Cross-modeler ingredient matching

`map_ingredients` ports a preset's *blocks* from one modeler to another by
their **ingredients** — the kind (amp/bassamp/cab/fx) plus a static set of
feature tags (`drive`, `fuzz`, `delay`, `reverb`, `pitch`, `tape`, `analog`,
…). It is deterministic: the tag rules are encoded in source, and matching is
a scored overlap, not an LLM guess.

- Arguments: `source_device`, `target_device` (names from `device_list`) and
  `blocks` — the source preset's block names in signal order.
- It returns a mapping table (source → target + score + reason), an overall
  **coverage** fraction and per-kind coverage. Unmatched blocks are listed with
  a reason; they are never silently dropped.
- For each matched block it also maps the **knobs** by canonical name
  (`GAIN`↔`DRIVE`, `LEVEL`↔`OUTPUT`, `MIX`↔`DIRECT MIX`, …): a `params` list of
  `{source, target, canonical}` links. These are *name* links — carry the
  source value across yourself with the target's own scale (each device
  numbers its knobs differently), and treat them as the starting point, not a
  finished conversion.
- The matching intentionally refuses spurious substitutions: a plain delay will
  not "cover" a harmonizer, but a delay that also pitch-shifts (carrying both
  `delay` and `pitch`) will — that is the cookbook case of a sub-feature
  standing in for a whole block.

Use it as a first pass when porting a tone: run `map_ingredients`, present the
table and coverage to the user, then refine the actual target blocks with the
per-device design tools (`design_rig`, `mooer_design`, `waza_write_tsl`,
`thr_setup_card`, `qc_design`) for any blocks the user wants to adjust. The
signal-chain *shape* (shorter chains, missing parallel paths) is a separate
concern and is not yet modelled — say so when the target device has fewer
slots than the source chain.

### Rethink the chain, routing and switching

`map_ingredients` (and `map_preset`) match *blocks and knobs* — they cannot
carry over the source device's chain shape, parallel routing or footswitches,
because those are device-specific and often simply don't exist on the target.
Re-decide them for the target device instead of copying the source:

- **Chain shape differs.** Gigboard has 11 free slots with parallel routing
  (`S`/`SPS-1`/`PS-1`) and dual-amp configs; Mooer has a fixed nine-module
  chain (FX→OD→AMP→CAB→NS→EQ→MOD→DELAY→REVERB) with no parallel paths and no
  dual amp; Waza Air shares slots (booster vs mod, delay vs fx); the Quad
  Cortex is a free 4-lane wire. A parallel wet/dry/wet Gigboard rig must be
  flattened to a serial Mooer chain — decide which blocks survive the squeeze.
- **Switching differs.** Gigboard: 4 stomp switches (FS5–FS8) + scenes + 2
  expression pedals. Mooer: no per-effect footswitches — it switches whole
  patches. Waza Air: no footswitches at all (an AIRSTEP BW adds them). Quad
  Cortex: 8+ switches + scenes. Re-map the *control intent* — a wah on a
  pedal, a drive on a stomp, a clean/lead scene — to what the target actually
  has, don't copy the source's assignments.
- **Convert in two passes.** First `map_ingredients` to line up the blocks and
  knobs, then rebuild with the target's own design tool re-deriving order,
  routing and footswitches for that device. Present both: the mapping is a
  starting point, not the finished preset.

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

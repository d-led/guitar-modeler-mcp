# Quad Cortex support

The Quad Cortex is supported for **cataloguing, translation, parameter listing
and preset design** — but the tool cannot upload a preset file to the unit.
This page explains exactly what is and isn't possible.

## What's implemented

- **Model catalog** — `qc_catalog_list_*` lists amps, cabs and effects with the
  real hardware each is based on. The catalog is parsed from the device's own
  `ModelRepo.xml`, so every model carries its wire id and every knob its real
  scale (min/max/skew).
- **Translation** — `qc_translate_amp` / `qc_translate_cab` map real-world
  hardware descriptions to the exact device model.
- **Parameter listing** — `qc_list_model_params` describes one model's knobs
  with their scale, so values can be set on the screen's own line.
- **Preset design** — `qc_design` builds a serial chain (amp → cab → effects)
  and writes a **self-contained HTML setup card** (the dial-in instructions)
  plus a `.pb` **reference archive** for saving and reloading the tone with
  `qc_decode_preset` / `qc_render_setup_card`.

## The `.pb` file is not importable

The `.pb` written by `qc_design` is this tool's own reference archive — **not a
file the unit imports**. To put the tone on the unit, dial it in from the HTML
card, or place a preset in a slot with Cortex Control and let `qcctl` recall it.
There is no private key involved: the archive uses the device's public
`KEY_MATERIAL` + serial (AES-128-CTR).

## Live USB (`qc_usb` → `qcctl`)

`qc_usb` shells out to the user's `qcctl` CLI (from
[pyquadcortex](https://github.com/stokes-audio/pyquadcortex)). `qcctl` has
exactly four subcommands — `version`, `recall --setlist --slot`,
`scene --index` and `dump-preset --setlist --slot` — and none of them takes a
file: it reads the firmware version, recalls a preset that is **already in a
slot on the unit**, switches scenes, and prints (dumps) the preset in a slot.
It does **not** upload the `.pb`. The whole wire format is the device's native
**protobuf**, not JSON: `qcctl dump-preset` prints a `BinaryPreset` protobuf,
there is no JSON import or export anywhere in pyquadcortex, and its
`write_preset` is a documented trap — a full preset written back wholesale is
silently ignored, so only keyed edits like `set_param` / `set_bypass` /
`set_chain_input` actually persist.

Install `qcctl` once:

```sh
pip install pyquadcortex        # macOS also: brew install hidapi
```

> **Crucial prerequisite:** before running any `qcctl` command, **quit the
> official Cortex Control desktop application**. It holds an exclusive lock on
> the hardware's USB interface and will block `qcctl` while it runs. (Wi-Fi on
> the device can stay active.)

## Attribution

The parameter scale law and firmware constants are attributed to
[pyquadcortex](https://github.com/stokes-audio/pyquadcortex) (MIT); the catalog
and preset schema come from
[OpenCortex](https://github.com/VanIseghemThomas/OpenCortex). See
`internal/qc/NOTICE.md`.

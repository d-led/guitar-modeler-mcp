# Roadmap

Planned work that is **not yet implemented**. Anything listed as implemented in
the [README](README.md) or the [Quad Cortex page](quad-cortex.md) is done.

- **Mooer GE150 (classic)** — write `.mo` preset files. Today the classic GE150
  is card-only; only the file-capable models (GE150 Pro Li, GE200, GE100 Pro)
  get `.mo` output.
- **Quad Cortex** — produce a preset file the unit imports directly. Today
  `qc_design` writes a setup card plus a `.pb` reference archive, and `qc_usb`
  (via `qcctl`) can recall a slot already on the unit but cannot upload a file.
- **Cross-device mapping** — extend `map_preset` beyond Gigboard ↔ Mooer to the
  remaining devices.

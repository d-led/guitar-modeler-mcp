# Test fixtures — Mooer `.mo` presets

These files are functional binary device data, not creative works, and each is a
single small preset.

- `ge200-clean.mo` — "ANTO CLEAN" (community preset, Anto Addabbo) and
  `ge200-lead.mo` — "LEAD LIVEPLAYRO" (free lead-solo preset download): real
  GE200 exports, kept so the GE200 reader can be checked against what the device
  produces.
- `ge150-empty.mo` — the classic GE150's `Empty.mo` template, bundled with the
  official GE150 Edit app (Resources/Empty.mo). It is the "GE150 Preset" JSON
  schema the app writes, kept so the GE150 JSON reader can be checked against
  the editor's own output. Note it carries the editor's "CEBTER" typo for the
  cab CENTER knob.
- `ge100pro-reference.mo.golden` — **this project's own** reference tone ("MV
  LEAD": gate, Mark V lead, 4x12, tape delay, hall reverb), written by
  `MarshalMOFor` and committed as a golden file. It pins the GE100 Pro layout we
  emit, so an accidental format change shows up as a diff. Regenerate with
  `UPDATE_GOLDEN=1 go test ./internal/mooer/`.

No GE100 Pro preset exported from anyone's device is redistributed here. To check
that reader against real hardware output, point the optional test at a file you
exported yourself:

```sh
MOOER_GE100PRO_EXPORT=~/path/to/export.mo go test ./internal/mooer/
```

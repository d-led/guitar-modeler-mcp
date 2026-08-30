# GE200 effect parameter order (manual — byte mapping unverified)

Source: the GE200 owner's manual effect-parameter list, transcribed via an AI
summary. **Treat as reference only**: it is the per-effect *screen* knob order,
and the screen→byte mapping for the `.mo` record has **not** been verified
against a device dump.

The `.mo` loader (`preset.go`) stores fixed byte positions with generic names;
`setupcard.go` prints them generically. Once the byte→knob mapping is confirmed
per effect, use this table to name `Param 4/5/6` (and MOD's `Level`) correctly.

## Our current byte layout (reverse-engineered, per module, after `enabled`+`type`)

| Module | Bytes in order |
| ------ | -------------- |
| FX     | Q, Position, Peak, Level |
| DS/OD  | Volume, Tone, Gain |
| AMP    | Gain, Bass, Mid, Treble, Presence, Master |
| CAB    | Mic, Center, Distance, Tube |
| NS     | Attack, Release, Threshold |
| EQ     | Band 1..6 (+ Band 7..12 extra) |
| MOD    | Rate, Level, Depth, Param 4, Param 5 |
| DELAY  | Level, Feedback, Time (16-bit LE), Subdivision, Param 5, Param 6 |
| REVERB | Pre-Delay, Level, Decay, Tone |

## Manual screen order (per effect)

### 1. FX/COMP (compressors, filters & wahs)

Compressors:
- JST COMP / SWEET COMP — `VOLUME → COMP → TONE`
- STUDIO COMP — `VOLUME → THRESH → RATIO → ATTACK`
- ORANGE COMP — `VOLUME → COMP`

Auto-/touch-wahs:
- TOUCH WAH — `VOLUME → SENS → RESO → DECAY`
- AUTO WAH — `VOLUME → SPEED → RANGE → RESO`

Manual/expression wahs:
- ANALOG WAH / CRY WAH / LOOSE WAH / 847 WAH — `POSITION → RANGE → VOLUME`
  (POSITION changes when assigned to the expression pedal)

### 2. DS/OD (overdrives, distortions, fuzzes)

Overdrives:
- PURE BOOST / FLEX BOOST / GREEN 808 / TUBESCRI / JUICER OD / CHU OD / MARSHA OD — `GAIN → TONE → VOLUME`
- GOLD RATIO / BE OD — `GAIN → TONE → VOLUME → TIGHT`

Distortion & fuzz:
- DIST+ / CRUNCH / RIOT DIST / SHRED / FACEFUZZ / MUFF FUZZ / FULL DRV — `GAIN → TONE → VOLUME`
- POCKET TONE — `GAIN → MID → VOLUME`
- METAL MASTER — `GAIN → TONE → VOLUME → SCOOP`

### 3. AMP (all 55 models share one 3-page setup)

- Page 1: `GAIN → BASS → MID`
- Page 2: `TREBLE → PRESENCE → VOLUME`
- Page 3: `MASTER → SAG → BIAS`

### 4. CAB

- `CAB MODEL → MIC → MIC DIST → LOW CUT → HIGH CUT → LATENCY → LEVEL`

### 5. NS

- `THRESH → DECAY`

### 6. EQ

- Graphic EQ (guitar) — `100Hz → 250Hz → 630Hz → 1.4kHz → 4kHz → LEVEL`
- Bass EQ — `50Hz → 120Hz → 400Hz → 800Hz → 4.5kHz → LEVEL`
- Parametric EQ — `FREQ 1 → GAIN 1 → Q 1 → FREQ 2 → GAIN 2 → Q 2 → LEVEL`

### 7. MOD

- ANA CHORUS / TRI CHORUS / RING MOD — `SPEED → DEPTH → MIX`
- PHASER / STEP PHASER / FAT PHASER — `SPEED → DEPTH → FEEDBACK`
- FLANGER / JET-FLANGER — `SPEED → DEPTH → FEEDBACK → MIX`
- TREMOLO / VIBRATO — `SPEED → DEPTH`
- PITCH SHIFT — `PITCH (-12..+12 semitones) → MIX`
- DETUNE — `AMOUNT → MIX`
- ROTARY — `SPEED → MIX`
- Q-FILTER / HIGH PASS / LOW PASS — `FREQ → Q → MIX`
- SLOW GEAR — `SENS → RAMP`
- LOFI — `SAMPLE RATE → BIT CRUSH → MIX`

### 8. DELAY

- DIGITAL / ANALOG / TAPE / ECHO — `TIME → FEEDBACK → MIX → TRAIL (on/off)`
- MOD DELAY — `TIME → FEEDBACK → MIX → MOD SPEED → MOD DEPTH → TRAIL`
- REVERSE DELAY — `TIME → FEEDBACK → MIX → TRAIL`
- PING PONG DELAY — `TIME → FEEDBACK → MIX → STEREO WIDTH → TRAIL`

### 9. REVERB

- ROOM / HALL / CHURCH / PLATE — `DECAY → PRE-DELAY → LOW CUT → HIGH CUT → MIX`
- SPRING REVERB — `DECAY → PRE-DELAY → SPRING TONE → MIX`

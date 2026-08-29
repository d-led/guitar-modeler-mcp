# Attribution

This package (`internal/qc`) supports the Neural DSP Quad Cortex. The Quad
Cortex and Neural DSP are trademarks of their owner; this project is not
affiliated with, endorsed by, or supported by Neural DSP Technologies.

The following files and facts are derived from third-party sources:

## `modelrepo.xml`

The device block catalog (`ModelRepo.xml`), copied from the OpenCortex project
(https://github.com/VanIseghemThomas/OpenCortex, file
`Model Repositories/ModelRepo[2.1.0].xml`). OpenCortex republishes the catalog
that the Quad Cortex itself ships; the catalog content is Neural DSP's product
data. OpenCortex has no repository-wide license file.

## `preset.proto` / `preset.pb.go`

The `BinaryPreset` protobuf schema, copied from OpenCortex
(`desktop_editor/public/protos/Preset.proto`). A wire schema describing the
device's own preset format. `preset.pb.go` is generated from it.

## Firmware constants and the parameter scale law

The symbolic bounds (`MIN_CABSIM_DB`, `MIN_MIXER_DB`, `MIN_TEMPO`, …) and the
encode law `wire = ((real - min) / (max - min)) ** skew`, including
`LOG_SKEW = 0.3`, come from the pyquadcortex project
(https://github.com/stokes-audio/pyquadcortex), MIT License,
Copyright (c) 2026 Stokes. The numbers are hardware measurements and facts;
the code here is an independent Go implementation.

## AES key material and derivation

The 57-byte `KEY_MATERIAL` constant and the key derivation
(`EVP_BytesToKey`, SHA-1, 10 iterations, no salt, AES-128-CTR) are from
OpenCortex's `File-decryption/qc_decrypt.c`
(https://github.com/VanIseghemThomas/OpenCortex), GPL-3.0,
Copyright (c) 2023 Simone Margaritelli. The key material is a published
constant; `crypto.go` is an independent reimplementation using the Go
standard library (`crypto/aes`, `crypto/sha1`, `crypto/cipher`).

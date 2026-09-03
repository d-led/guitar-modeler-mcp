# Test fixtures

The `.prst` files in this directory are unmodified device preset files copied
from public community repositories and used only as round-trip / golden-master
test fixtures.

| File                    | Source                                                          | License   |
|-------------------------|-----------------------------------------------------------------|-----------|
| `factory-50s-plexi.prst` | PRSTDecoder `examples/FactoryPresets200/01-B 50s Plexi.prst`    | MIT       |
| `factory-wild-fruit.prst` | PRSTDecoder `examples/FactoryPresets200/02-B Wild Fruit (Wah).prst` | MIT |
| `user-fender-twin.prst` | presets-valeton-gp200 `01-A Fender twin 2x12.prst`              | unstated  |
| `user-dark-twin.prst`   | presets-valeton-gp200 `01-D Dist Dark Twin.prst`                | unstated  |

Sources:

- https://github.com/mikeliddle/PRSTDecoder (MIT License, Copyright (c) 2023 Mike)
- https://github.com/victorhugo7691/presets-valeton-gp200 (community-shared
  preset files; repository has no license file)

These are Valeton GP-200 preset data files, not source code. They are included
solely to verify that our importer parses and our exporter reproduces real
device files byte-for-byte.

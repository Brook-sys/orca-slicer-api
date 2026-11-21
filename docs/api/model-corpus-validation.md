# Local real-model validation (2026-11-21)

## Scope and reproducibility

Local HTTP API → official OrcaSlicer 2.5.0 → strict plate/quantity validation → toolpath parser → thumbnail response. No printer execution or production deployment. Runtime used an isolated Ubuntu 24.04 userspace through PRoot on Linux amd64; executable help banner was checked. This is not the final Docker image and durations are not production benchmarks.

Nine downloaded STLs (~33.4 MB total), originals preserved locally:

- Stanford bunny and 40-tooth gear from [ad-si/stl-models](https://github.com/ad-si/stl-models), commit `7a8b0de5405b7be2667feb24b01a6823168173ca`.
- Official 3DBenchy: complete boat, 17-shell assembly, bridge walls, roof and rod holder. Sources, pinned revisions, checksums and license evidence retained in the local corpus manifest. The official license page stated CC0 when retrieved.
- Prusa calibration temperature-step and an ASCII 20 mm box control. The temperature-step is not a complete temperature-changing print.

Unknown asset licenses were not inferred; no external meshes or private profiles are versioned. Local manifest records URLs, SHA-256, triangles, finite coordinates and bounds. Structural STL checks do not establish manifoldness. Bunny alone was scaled uniformly ×400 in a derived input; the source remained unchanged. Other inputs were byte copies. Tests used private copies of existing printer/process/filament profiles; these are not portable public fixtures.

## Results

Initial matrix: each model with one piece, one piece with support/brim enabled, and three pieces with arrangement and explicit zero Z rotation. Requests were sequential within the matrix. Enabling support/brim does not prove Orca generated them for every geometry.

- **24/27 successful previews**, exact requested quantity and complete source parsing; 12 explicitly truncated at the 20,000 retained-segment limit.
- Successful HTTP durations in the final matrix: **0.675–9.410 seconds**; single observations, not latency percentiles. Models generated up to **238,928 segments**.
- Bunny failed in all three initial orientations inside Orca: empty layers between 38.28 and 39.48 mm. **Auto-orient and X=90° each subsequently succeeded**, with 307/308 layers and 180,086/182,929 segments. No mesh repair was applied. Orientation dependence is not a parser failure or a physical printability guarantee.
- Explicit spiral-lift Benchy subsequently succeeded in **3.995 seconds**. A simultaneous request during the bunny test returned expected **409 busy**; sequential retry succeeded.
- Clean published revision `5f6b6d2` was built separately: all three Benchy scenarios reproduced **500 / unsupported P1**, at source lines 1817, 1915 and 5702. Corrected builds succeeded on those scenarios.
- An earlier binary labeled "before" accidentally included concurrent P1 edits. Its results are **not** regression evidence; the clean-revision comparison above supersedes that label. Repeated Benchy requests in that earlier run included cache hits and are not evidence of fresh slicing.

## Parser fixes and boundaries

Orca 2.5.0 emits `G3 Z… I… J… P1` for spiral lift. Numeric P1 is accepted only on G2/G02/G3/G03 without adding another revolution; other P values and P on other commands fail closed. Firmware dialects differ; multi-turn arcs are not implemented.

A full existing reference G-code also exposed thumbnail base64 starting with V being mistaken for an embedded command. Recognized PNG thumbnail envelopes now validate dimensions, encoded length, alphabet/padding and closure without buffering/decoding. Executable lines inside image blocks and embedded V commands outside them still fail. Full unmodified reference parsed: 14,811 lines, 75 layers, 10,264 segments, no truncation. Personal reference data is not versioned.

Synthetic regression tests cover helical CW/CCW/full/partial arcs, positive/no extrusion, endpoints and following moves, unsupported P values, bounded thumbnail parsing and malformed envelopes. Go tests, race tests and vet passed locally.

## Remaining gaps

This finite corpus is not exhaustive firmware compatibility, collision validation, physical printing, sustained concurrency/load, peak CLI memory measurement, or browser rendering verification. No-fit/overflow, async lifecycle, timeouts and restarts require their separate existing gates. Errors from Orca can remain generic at HTTP level; detailed empty-layer diagnostics were found in debug output. PRoot shares the host kernel/network and is not a security sandbox. External assets, logs, inputs and manifests stay local.

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

## Full/v2 response and memory measurements

The optional full preview was subsequently exercised through HTTP in fresh, sequential local API processes. Every row below returned HTTP 200, `version=2`, `geometryComplete=true`, `sourceComplete=true`, `truncated=false`, all listed layers/segments, zero omitted segments and explicit width/height metadata on every segment. Durations are client-observed curl totals; RSS is the Go API process's `/proc` `VmHWM` sampled after the response. These are single observations, not latency percentiles or whole-container peaks; Orca child-process and browser memory are not included.

| Real model/configuration | Layers | Segments | Response bytes | HTTP seconds | API peak RSS (KiB) |
|---|---:|---:|---:|---:|---:|
| 3DBenchy, quantity 1, arrange | 240/240 | 76,160/76,160 | 14,156,412 | 4.160419 | 149,728 |
| 3DBenchy, quantity 3, arrange | 240/240 | 228,444/228,444 | 42,253,033 | 8.160165 | 259,572 |
| 17-shell 3DBenchy, quantity 1, explicit support/brim | 240/240 | 79,718/79,718 | 14,801,567 | 4.858713 | 184,356 |

These three rows were rerun with the final 250,000-segment cap. Earlier pre-cap measurements included the same Benchy quantity-3 geometry at 42,253,034 response bytes, 8.102157 seconds and 283,664 KiB API peak RSS, and a 17-shell quantity-3 case with 238,928/238,928 segments across 240/240 layers at 44,146,832 bytes, 9.364525 seconds and 215,844 KiB. Those responses advertised the temporary 1,000,000-segment limit; they are cap-selection evidence, not exact final-wire measurements.

The first Benchy runs retained verbatim `Bridge`, `Internal Bridge`, `Overhang wall`, `Support`, `Support interface`, wall and infill categories. The explicit support/brim case contained 9,029 support segments (including support interface), 76 brim segments, and separately 484 `Bridge` and 920 `Internal Bridge` segments. None of the three final-cap responses had null `widthMm` or `heightMm`; synthetic tests separately verify explicit nulls when comments are absent and that subsequent metadata changes do not mutate earlier segments. This demonstrates category/metadata coverage for these outputs, not exhaustive Orca category coverage.

An opt-in synthetic benchmark parsed compact relative-motion G-code and encoded the resulting object JSON to a counting writer, excluding Orca and HTTP transfer. Using `json.Encoder` here does not mean bounded-memory incremental segment streaming; the standard encoder buffers the encoded value:

| Retained segments | G-code bytes | JSON bytes | Parse + encode seconds | Cumulative allocated bytes | Benchmark process peak RSS (KiB) |
|---|---:|---:|---:|---:|---:|
| 250,000 | 4,250,041 | 38,746,814 | 1.021529768 | 432,486,688 | 191,068 |
| 1,000,000 (temporary pre-cap stress run) | 17,000,041 | 154,320,352 | 4.069407015 | 1,862,989,640 | 707,072 |

Cumulative allocated bytes are Go benchmark `B/op`, not resident memory. The million-segment run succeeded only when that temporary ceiling was configured; it is not an accepted production response after the measurements. The final full-mode segment ceiling is **250,000**, with the independent 4,096-layer ceiling. This covers the largest real case measured here and avoids treating the 1M/154 MB/707,072 KiB point as safe. It is still not a universal memory guarantee: the real 228,444-segment HTTP case had higher observed RSS than the synthetic 250,000-segment benchmark, concurrent response encoders are outside the slicing lock, and no container memory limit or frontend parse/render budget was established. Raw local responses and logs remain outside the repository.

An initial HTTP 500 in the isolated launcher was caused by its temporary work directory not being exposed to the CLI userspace. The successful reruns corrected that launcher setup; the failure was not a full-preview retention-budget rejection.

### Backend verification and interpretation

The reviewed backend uses independent full-mode ceilings of 250,000 retained segments and 4,096 retained layers. Crossing either ceiling returns HTTP 422 with no partial geometry, whereas the default legacy response still retains a prefix at 20,000 segments/512 layers. Both modes also retain the independent 32 MiB source, 1 MiB line, 4,000,000-line and arc-work limits. Full mode does not bypass unsupported-command, malformed-metadata or source-reading failures; `geometryComplete` describes successfully parsed positive-extrusion geometry, not travel moves, physical bead shape or a printer simulation.

Local review gates passed: `go test ./...`, `go vet ./...`, and targeted `go test -race ./internal/slicer ./internal/docs -run 'Full|Toolpath|Preview' -count=1`. The race run exercised the slicer package; no docs tests matched that name filter (the docs contract test ran in the ordinary suite). Synthetic tests cover retention beyond the legacy ceilings, exact small-budget acceptance followed by explicit over-budget errors, full/legacy JSON separation, pure-Z positive extrusion, feature/object preservation and invalid dimensions. These tests do not establish sustained HTTP concurrency safety or container/browser memory budgets. The recorded synthetic performance measurements used a single benchmark iteration; do not interpret them as repeated-run statistics.

## Parser fixes and boundaries

Orca 2.5.0 emits `G3 Z… I… J… P1` for spiral lift. Numeric P1 is accepted only on G2/G02/G3/G03 without adding another revolution; other P values and P on other commands fail closed. Firmware dialects differ; multi-turn arcs are not implemented.

A full existing reference G-code also exposed thumbnail base64 starting with V being mistaken for an embedded command. Recognized PNG thumbnail envelopes now validate dimensions, encoded length, alphabet/padding and closure without buffering/decoding. Executable lines inside image blocks and embedded V commands outside them still fail. Full unmodified reference parsed: 14,811 lines, 75 layers, 10,264 segments, no truncation. Personal reference data is not versioned.

Synthetic regression tests cover helical CW/CCW/full/partial arcs, positive/no extrusion, endpoints and following moves, unsupported P values, bounded thumbnail parsing and malformed envelopes. Go tests, race tests and vet passed locally.

## Remaining gaps

This finite corpus is not exhaustive firmware compatibility, collision validation, physical printing, sustained concurrency/load, peak CLI memory measurement, or browser rendering verification. No-fit/overflow, async lifecycle, timeouts and restarts require their separate existing gates. Errors from Orca can remain generic at HTTP level; detailed empty-layer diagnostics were found in debug output. PRoot shares the host kernel/network and is not a security sandbox. External assets, logs, inputs and manifests stay local.

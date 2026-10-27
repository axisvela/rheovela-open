# RHEO IR Conformance Corpus

This directory contains the public conformance corpus for the
[RHEO IR v1.0 specification](../docs/RHEO-IR.md).

## Layout

```
rheo-ir/v1.0/
├── manifest.json          # case index
├── valid/<case>.rheo      # accepted source
├── valid/<case>.json      # expected WorkflowDefinition
├── invalid/<case>.rheo    # rejected source
└── invalid/<case>.json    # expected {code, line, message_contains}
```

The Go package [`rheoirv1`](./conformance.go) embeds the corpus and exposes it
as `conformance.RHEOIRv1`, so downstream implementations can run the same
fixtures without copying files.

## Running

```sh
# open: corpus structure and expected JSON
go test ./conformance/... -count=1

# core: parser + compiler against every fixture
cd ../Rheovela
go test ./internal/rheoir -run TestConformanceCorpus -count=1
```

## Adding a case

1. Add the source and expected output under `valid/` or `invalid/`.
2. Add an entry to `manifest.json`.
3. Run both commands above.

Invalid cases must use one of the stable `RHEOIRxxx` codes documented in
`docs/RHEO-IR.md` §7. `valid/*.json` must satisfy
[`api/workflow.schema.json`](../api/workflow.schema.json).

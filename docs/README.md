# Protoflow Documentation

This folder groups the deeper guides that keep the top-level `README.md` focused on the essentials. Use these docs whenever you need to wire transports, tune middleware, or explore handler ergonomics beyond the quick start.

## Contents

- [Handlers guide](handlers/README.md) — strongly typed protobuf and JSON handlers, publishing helpers, and metadata patterns.
- [Configuration guide](configuration/README.md) — broker setup, common knobs, middleware wiring, and custom transport factories.
- [Development guide](development/README.md) — local broker tips, test commands, and ways to use the Taskfile-driven workflow.

## Related resources

- `examples/` showcases runnable scenarios (`simple`, `json`, `proto`, `full`) that complement the docs.
- Runtime source code lives under `internal/runtime/` if you want to inspect transports, middleware, or helpers directly.

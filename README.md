# protoextract

`protoextract` scans a compiled Go binary for embedded `FileDescriptorProto`
messages and reconstructs every descriptor it can recover.

## Usage

```sh
go build -o protoextract .
./protoextract /path/to/language_server /path/to/output
```

The command accepts exactly two positional arguments: the source binary and the
output directory. Before extraction, the output directory is deleted in full
and recreated. Do not point it at a directory containing unrelated files.

## Outputs

- `all-protos.proto`: every recovered declaration flattened into one directly
  compilable proto file. The `exa.api_server_pb` package is retained when it is
  present, while symbols from other packages receive deterministic prefixes.
- `descriptors.pb`: the complete machine-readable `FileDescriptorSet`.
- `manifest.json`: file counts, comment coverage, missing dependencies, caveats,
  and mappings from original fully-qualified symbols to flattened names.

The flattened file uses proto2 syntax so a mixed proto2/proto3 input set can
retain required fields, extensions, maps, oneofs, and packed wire encoding in a
single syntax. It is a wire-compatible analysis view, not a replacement for the
original generated API: non-root service paths and type names change, and
proto3 presence/open-enum behavior cannot be represented exactly. Use
`descriptors.pb` whenever the original file boundaries, packages, options, or
generated API semantics matter.

## Comments

Proto comments exist in compiled descriptors only when the producer retained
`FileDescriptorProto.source_code_info`. When present, leading, trailing, and
detached comments attached to declarations are remapped into the flattened
file. Comments attached only to removed imports, package declarations, syntax,
or custom options are retained as file-level provenance comments. When the
binary stripped source information, original comments cannot be recovered;
`manifest.json` reports zero comment locations instead of inventing comments.

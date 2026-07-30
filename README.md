# cha-k

`cha-k` is a Go service under development for translating a provider-independent
conversation model to these compatible HTTP APIs:

- OpenAI `/v1/chat/completions`
- OpenAI `/v1/responses`
- Anthropic `/v1/messages`

The HTTP service entry provides `GET /healthz` and `POST /v1/responses`. The
Responses route is wired to an adapter boundary; the concrete upstream provider
adapter is not configured yet.

## Layout

- `internal/llm`: provider-independent request messages, response messages, and
  streaming events derived from Pi's common type layer.
- `cmd/protoextract`: side command for recovering embedded protobuf descriptors
  from a compiled Go binary.
- `pi`: Git submodule used as the reference implementation for the common
  message model.

Initialize the reference submodule after cloning:

```sh
git submodule update --init --depth 1
```

## Service entry

```sh
go run ./cmd/cha-k --config ./config.yaml
```

All service settings come from YAML. The service does not use environment
variables or hot reload. Copy `config.example.yaml` when creating a new
deployment configuration.

## Proto extractor

```sh
go build -o bin/protoextract ./cmd/protoextract
./bin/protoextract /path/to/language_server /path/to/output
```

The extractor accepts exactly two positional arguments: the source binary and
the output directory. Before extraction, the output directory is deleted in
full and recreated. Do not point it at a directory containing unrelated files.

It creates:

- `all-protos.proto`: every recovered declaration flattened into one directly
  compilable proto file.
- `descriptors.pb`: the complete machine-readable `FileDescriptorSet`.
- `manifest.json`: extraction counts, comment coverage, missing dependencies,
  caveats, and original-to-flattened symbol mappings.

The flattened file is a wire-compatible analysis view, not a replacement for
the original generated API. Use `descriptors.pb` when original file boundaries,
packages, options, or generated API semantics matter.

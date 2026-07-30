# cha-k

`cha-k` is a Go service under development for translating a provider-independent
conversation model to these compatible HTTP APIs:

- OpenAI `/v1/chat/completions`
- OpenAI `/v1/responses`
- Anthropic `/v1/messages`

The HTTP service entry provides `GET /healthz` and `POST /v1/responses`. When
the YAML `devin.token` is populated, the Responses route uses the Devin
Connect `ApiServerService/GetChatMessage` adapter.

## Layout

- `internal/llm`: provider-independent request messages, response messages, and
  streaming events derived from Pi's common type layer.
- `internal/adapter/devin`: converts the common request model to Devin Connect
  protobuf and converts the server stream back to common response events.
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

Each `POST /v1/responses` request creates a debug directory under `logs` next
to the selected YAML configuration file. Directories use the request entry
time, for example `logs/20270101-235454/`; same-second requests receive `-02`,
`-03`, and later suffixes. A complete Devin request can contain:

- `meta.json`: request timing, status, model, provider, and completion result.
- `01-http-request.json`: the incoming HTTP request with a header allowlist.
- `02-request-messages.json`: the provider-independent request context.
- `03-devin-request.json`: the redacted Devin protobuf request.
- `04-devin-response.jsonl`: ordered raw Devin protobuf response frames.
- `05-response-events.jsonl`: ordered provider-independent response events.
- `06-http-response.jsonl`: ordered OpenAI response or SSE events.
- `error.json`: the first failure stage and error summary, when present.
- `attachments/`: decoded image payloads referenced by the JSON logs.

Credentials and device fingerprints are redacted. Logging is best effort and
does not change the API result when disk writes fail.

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

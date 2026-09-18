# BUKOWSKIREAL fork: request fidelity

This fork fixes information loss in the upstream adapter. The upstream project's
README below predates several endpoints; this document describes this fork's
behavior. It does not promise complete compatibility with every OpenAI or
Anthropic feature.

## Changes

- Message text is preserved, including `<permissions instructions>` blocks in
  system instructions, user content, assistant history, and tool results. The
  gateway does not remove permission instructions to work around provider policy.
  Provider refusals remain visible to callers.
- Native tool fields retain the upstream adapter's compatible name-only description and
  annotation-free schema. The **complete** original tool description and schema
  are appended as a JSON tool-reference section instead of being lost. Description
  strings retain their formatting; they are not split into sentences or renumbered.
  The native schema traversal preserves property/definition names and literal
  values, including large integers. This duplicates some schema tokens, a deliberate
  tradeoff for retaining parameter semantics. No claim is made that the provider
  enforces every JSON Schema constraint.
- Generation controls travel through the provider-neutral message model into
  the upstream protobuf request, including explicit zero temperature.
- Chat `reasoning_content` history and Anthropic thinking text/signatures are
  preserved. This does not imply support for every provider's opaque reasoning
  format or for a complete Responses reasoning-item round trip.
- Invalid historical tool arguments are rejected. Invalid completed upstream
  tool arguments produce an error, never a fabricated empty object. Streaming
  callers must discard an incomplete tool call when the stream ends in error.
- The hardcoded 400-newline generation limit is removed.

## Control support

| Control | Behavior |
| --- | --- |
| Chat `max_tokens` / `max_completion_tokens` | Forwarded; `max_completion_tokens` wins when both are supplied |
| Responses `max_output_tokens` | Forwarded |
| Messages `max_tokens` | Required, positive, forwarded |
| `temperature`, `top_p` | Forwarded on all three protocols |
| Messages `top_k` | Forwarded |
| `tool_choice` | auto / none / required (Messages any), or a named function, mapped to native tool choice |
| Chat `reasoning_effort`, Responses `reasoning.effort` | SWE-2 only: medium / high / max selects the corresponding `swe-2-*` UID; explicit effort overrides the UID suffix |
| SWE-2 off / none / low / xhigh, or effort on another model | HTTP 400; never silently treated as another effort |
| Stop sequences, response formatting, parallel tool controls | HTTP 400 when supplied; exact upstream semantics have not been verified |
| Responses `previous_response_id`, `text`, reasoning summaries | HTTP 400; send full history instead |
| Messages `thinking` / `output_config` | HTTP 400; select a model UID rather than pretending budget/adaptive controls are supported |

Absent max tokens / temperature / top-p use the existing upstream adapter
defaults (128000 / 1 / 0.95). Forwarding a field means its value reaches the
upstream request, not that every upstream model guarantees identical behavior.
Unknown metadata fields are still ignored; this is not a fully strict protocol
validator. Existing image-history restrictions remain.

## Kimi Code

Use `type = "openai"` and the Chat Completions endpoint. Choose one of
`swe-2-medium`, `swe-2-high`, `swe-2-max`, or send an explicit supported
`reasoning_effort`. A UI toggle must not be interpreted as a guaranteed way to
disable SWE-2 reasoning. Kimi Code version-specific effort UI configuration is
separate from this gateway change.

## Build and validation

```sh
# After installing the protoc toolchain documented in CONTRIBUTING.md:
task generate
go test ./...
go build ./...
docker build -t devin-2api-fidelity:local .
```

The upstream Docker Hub image does **not** contain these fork changes. Build
this fork; do not use `leokun123/devin-2api:latest` expecting the fixes.
The fork does not automatically upgrade any existing deployment.

Regression coverage includes protocol-to-protobuf controls, unchanged system /
user / tool text, preserved nested parameter documentation and references,
reasoning replay, malformed tool arguments, and unsupported-control errors.

Live smoke checks on 2026-09-18 used a local build and the account's SWE-2 models:

- Auto tool call interpreted "two seconds" as `timeout: 2000` from the parameter
  documentation, then consumed the synthetic tool result correctly.
- Named tool choice returned the requested tool; `none` returned text after the
  result was replayed.
- Chat reasoning history was included in a successful continuation.
- Explicit `reasoning_effort: medium` selected `swe-2-medium`; `off` returned 400.

A tool request with `max_tokens: 256` and `temperature: 0` received an upstream
`invalid_argument`; the corresponding default-parameter request succeeded.
These controls are forwarded, not guaranteed acceptable to every model in every
combination. The smoke test does not isolate which field caused that rejection.
It also does not establish long-context reliability or full Kimi Code workflow
compatibility. No live tests execute local or remote shell/file-editing tools.

// 本文件验证客户端参数与上下文穿过真实解码器后，仍能到达 Devin protobuf 请求。
package devin

import (
	"encoding/json"
	"github.com/leookun/devin-2api/internal/api/anthropic/messages"
	"github.com/leookun/devin-2api/internal/api/openai/chat"
	"github.com/leookun/devin-2api/internal/api/openai/responses"
	"github.com/leookun/devin-2api/internal/llm"
	"google.golang.org/protobuf/proto"
	devinproto "local/devinproto"
	"strings"
	"testing"
)

func TestChatControlsAndHistoryReachUpstream(t *testing.T) {
	body := `{"model":"swe-2-high","max_tokens":123,"temperature":0,"top_p":0.7,"reasoning_effort":"max","tool_choice":{"type":"function","function":{"name":"probe"}},"tools":[{"type":"function","function":{"name":"probe","description":"Exact. Description.","parameters":{"type":"object","properties":{"timeout":{"type":"integer","description":"milliseconds"}}}}}],"messages":[{"role":"system","content":"<permissions instructions>keep system</permissions instructions>"},{"role":"user","content":"<permissions instructions>quoted text</permissions instructions>"},{"role":"assistant","content":null,"reasoning_content":"retain reasoning","tool_calls":[{"id":"call-1","type":"function","function":{"name":"probe","arguments":"{\"timeout\":1}"}}]},{"role":"tool","tool_call_id":"call-1","content":"<permissions instructions>keep tool output</permissions instructions>"}]}`
	decoded, err := chat.DecodeRequest([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	got, err := buildRequest(decoded.Context, Config{Model: decoded.Context.Model})
	if err != nil {
		t.Fatal(err)
	}
	if got.GetChatModelUid() != "swe-2-max" || got.GetConfiguration().GetMaxTokens() != 123 || got.GetConfiguration().GetTemperature() != 0 || got.GetConfiguration().GetTopP() != 0.7 {
		t.Fatalf("generation lost: %v", got.GetConfiguration())
	}
	if got.GetToolChoice().GetToolName() != "probe" {
		t.Fatal("named tool selection lost")
	}
	if !strings.HasPrefix(got.GetPrompt(), "<permissions instructions>keep system</permissions instructions>\n\n") {
		t.Fatalf("system changed: %q", got.GetPrompt())
	}
	history := got.GetChatMessagePrompts()
	if history[0].GetPrompt() != "<permissions instructions>quoted text</permissions instructions>" || history[1].GetThinking() != "retain reasoning" || history[2].GetPrompt() != "<permissions instructions>keep tool output</permissions instructions>" {
		t.Fatalf("history changed: %v", history)
	}
	if !strings.Contains(got.GetPrompt(), "milliseconds") {
		t.Fatal("parameter documentation lost")
	}
	if !strings.Contains(got.GetPrompt(), "Exact. Description.") {
		t.Fatal("description rewritten")
	}
}

func TestOtherProtocolsCarryGeneration(t *testing.T) {
	r, err := responses.DecodeRequest([]byte(`{"model":"swe-2-high","input":"hi","max_output_tokens":21,"temperature":0.2,"top_p":0.8,"reasoning":{"effort":"medium"},"tool_choice":"none"}`))
	if err != nil {
		t.Fatal(err)
	}
	p, err := buildRequest(r.Context, Config{Model: r.Context.Model})
	if err != nil {
		t.Fatal(err)
	}
	if p.GetConfiguration().GetMaxTokens() != 21 || p.GetChatModelUid() != "swe-2-medium" || p.GetToolChoice().GetOptionName() != "none" {
		t.Fatal("Responses controls lost")
	}
	a, err := messages.DecodeRequest([]byte(`{"model":"swe-2-high","max_tokens":32,"temperature":0,"top_p":0.3,"top_k":7,"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"keep","signature":"opaque"},{"type":"text","text":"answer"}]},{"role":"user","content":"next"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	p, err = buildRequest(a.Context, Config{Model: a.Context.Model})
	if err != nil {
		t.Fatal(err)
	}
	if p.GetConfiguration().GetMaxTokens() != 32 || p.GetConfiguration().GetTopK() != 7 || p.GetChatMessagePrompts()[0].GetSignature() != "opaque" || p.GetChatMessagePrompts()[0].GetThinking() != "keep" {
		t.Fatal("Anthropic controls/history lost")
	}
}

func TestInvalidToolArgumentsProduceErrorEvent(t *testing.T) {
	d := newResponseDecoder("swe-2-high")
	d.start()
	d.decode(&devinproto.GetChatMessageResponse{DeltaToolCalls: []*devinproto.ExaCodeiumCommonPb_ChatToolCall{{Id: proto.String("broken"), Name: proto.String("probe"), ArgumentsJson: proto.String(`{"timeout":`)}}})
	events := d.finish(nil)
	if len(events) != 1 || events[0].Type != llm.ResponseEventError {
		t.Fatalf("invalid tool was reported successful: %v", events)
	}
}

func TestUnsupportedReasoningAndInvalidHistoryFail(t *testing.T) {
	for _, effort := range []string{"off", "low", "xhigh"} {
		if _, err := reasoningModel("swe-2-high", effort); err == nil {
			t.Fatalf("unsupported effort %q accepted", effort)
		}
	}
	if _, err := reasoningModel("another-model", "high"); err == nil {
		t.Fatal("unsupported model accepted")
	}
	for _, args := range []string{"{broken", "[]", "null", ""} {
		body := map[string]any{"model": "swe-2-high", "messages": []any{map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"id": "x", "type": "function", "function": map[string]any{"name": "probe", "arguments": args}}}}}}
		wire, _ := json.Marshal(body)
		if _, err := chat.DecodeRequest(wire); err == nil {
			t.Fatalf("invalid arguments %q accepted", args)
		}
	}
}

func TestUnsupportedControlsFailExplicitly(t *testing.T) {
	for _, extra := range []string{`"stop":["END"]`, `"response_format":{"type":"json_object"}`, `"parallel_tool_calls":true`, `"max_tokens":0`, `"temperature":-1`, `"top_p":2`} {
		_, err := chat.DecodeRequest([]byte(`{"model":"swe-2-high","messages":[{"role":"user","content":"hi"}],` + extra + `}`))
		if err == nil {
			t.Fatalf("unsupported/invalid control accepted: %s", extra)
		}
	}
}

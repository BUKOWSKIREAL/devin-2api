// 本文件区分未知推理用量与真实零值，并校验流式模型标识更新。
package chat

import (
	"encoding/json"
	"github.com/leookun/devin-2api/internal/llm"
	"testing"
)

func TestReasoningUsageUnknownZeroAndPositive(t *testing.T) {
	zero, positive := int64(0), int64(42)
	for _, value := range []*int64{nil, &zero, &positive} {
		message := &llm.AssistantMessage{Model: "swe-2-max", StopReason: llm.StopReasonStop, Usage: llm.Usage{Output: 50, Reasoning: value}}
		wire, err := EncodeResponse(message)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		json.Unmarshal(wire, &decoded)
		checkReasoningUsage(t, decoded["usage"].(map[string]any), value)
		encoder := NewStreamEncoder("swe-2-max", true)
		events, err := encoder.Encode(llm.ResponseEvent{Type: llm.ResponseEventDone, Reason: llm.StopReasonStop, Message: message})
		if err != nil {
			t.Fatal(err)
		}
		var streamed map[string]any
		json.Unmarshal(events[1].Data, &streamed)
		checkReasoningUsage(t, streamed["usage"].(map[string]any), value)
	}
}

func checkReasoningUsage(t *testing.T, usage map[string]any, want *int64) {
	t.Helper()
	details, exists := usage["completion_tokens_details"]
	if want == nil {
		if exists {
			t.Fatalf("unknown reasoning fabricated: %v", details)
		}
		return
	}
	if !exists || details.(map[string]any)["reasoning_tokens"] != float64(*want) {
		t.Fatalf("known reasoning lost: %v", usage)
	}
}

func TestStreamReportsRoutedThenActualModel(t *testing.T) {
	encoder := NewStreamEncoder("client-alias", false)
	events := []llm.ResponseEvent{
		{Type: llm.ResponseEventStart, Partial: &llm.AssistantMessage{Model: "routed-model", StopReason: llm.StopReasonPending}},
		{Type: llm.ResponseEventTextDelta, ContentIndex: 0, Delta: "hi", Partial: &llm.AssistantMessage{Model: "routed-model", ResponseModel: "actual-model", StopReason: llm.StopReasonPending}},
		{Type: llm.ResponseEventDone, Reason: llm.StopReasonStop, Message: &llm.AssistantMessage{Model: "routed-model", StopReason: llm.StopReasonStop}},
	}
	for i, event := range events {
		encoded, err := encoder.Encode(event)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]any
		json.Unmarshal(encoded[0].Data, &payload)
		want := "actual-model"
		if i == 0 {
			want = "routed-model"
		}
		if payload["model"] != want {
			t.Fatalf("event %d model=%v want=%s", i, payload["model"], want)
		}
	}
}

// 本文件验证 Responses 的未知推理用量不会伪造成零，完成事件使用上游模型。
package responses

import (
	"encoding/json"
	"github.com/leookun/devin-2api/internal/llm"
	"testing"
)

func TestResponseReasoningUsageUnknownZeroAndPositive(t *testing.T) {
	zero, positive := int64(0), int64(42)
	for _, value := range []*int64{nil, &zero, &positive} {
		message := &llm.AssistantMessage{Model: "routed-model", ResponseModel: "actual-model", StopReason: llm.StopReasonStop, Usage: llm.Usage{Output: 50, Reasoning: value}}
		wire, err := EncodeResponse(message)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		json.Unmarshal(wire, &decoded)
		checkResponseUsage(t, decoded["usage"].(map[string]any), value)
		encoder := NewStreamEncoder("client-alias")
		created, err := encoder.Encode(llm.ResponseEvent{Type: llm.ResponseEventStart, Partial: &llm.AssistantMessage{Model: "routed-model", StopReason: llm.StopReasonPending}})
		if err != nil {
			t.Fatal(err)
		}
		var start map[string]any
		json.Unmarshal(created[0].Data, &start)
		if start["response"].(map[string]any)["model"] != "routed-model" {
			t.Fatal("start uses client alias")
		}
		events, err := encoder.Encode(llm.ResponseEvent{Type: llm.ResponseEventDone, Reason: llm.StopReasonStop, Message: message})
		if err != nil {
			t.Fatal(err)
		}
		var completed map[string]any
		json.Unmarshal(events[0].Data, &completed)
		response := completed["response"].(map[string]any)
		if response["model"] != "actual-model" {
			t.Fatal("final model was not updated")
		}
		checkResponseUsage(t, response["usage"].(map[string]any), value)
	}
}
func checkResponseUsage(t *testing.T, usage map[string]any, want *int64) {
	t.Helper()
	details, exists := usage["output_tokens_details"]
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

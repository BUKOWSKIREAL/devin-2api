// 本文件验证 Anthropic 起始消息使用当时已知的上游模型而非客户端别名。
package messages

import (
	"encoding/json"
	"github.com/leookun/devin-2api/internal/llm"
	"testing"
)

func TestMessageStartUsesKnownUpstreamModel(t *testing.T) {
	encoder := NewStreamEncoder("client-alias")
	events, err := encoder.Encode(llm.ResponseEvent{Type: llm.ResponseEventStart, Partial: &llm.AssistantMessage{Model: "routed-model", ResponseModel: "actual-model", StopReason: llm.StopReasonPending}})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	json.Unmarshal(events[0].Data, &result)
	if result["message"].(map[string]any)["model"] != "actual-model" {
		t.Fatalf("wrong start model: %s", events[0].Data)
	}
}

// 本文件验证中间响应事件到 OpenAI Responses SSE 的有序转换。
package responses

import (
	"strings"
	"testing"

	"github.com/leookun/cha-k/internal/llm"
)

// TestEncodeEventUsesTypedResponsesEvent 验证文本增量使用 Responses typed event 名称和字段。
func TestEncodeEventUsesTypedResponsesEvent(t *testing.T) {
	event, err := EncodeEvent(llm.ResponseEvent{
		Type:         llm.ResponseEventTextDelta,
		ContentIndex: 0,
		Delta:        "hello",
		Partial:      &llm.AssistantMessage{StopReason: llm.StopReasonPending},
	})
	if err != nil {
		t.Fatal(err)
	}
	if event.Name != "response.output_text.delta" {
		t.Fatalf("event name = %q", event.Name)
	}
	if !strings.Contains(string(event.Data), `"delta":"hello"`) {
		t.Fatalf("event data = %s", event.Data)
	}
}

// TestEncodeEventAllowsEmptyToolDelta 验证空工具参数增量仍会被保留。
func TestEncodeEventAllowsEmptyToolDelta(t *testing.T) {
	event, err := EncodeEvent(llm.ResponseEvent{
		Type:         llm.ResponseEventToolCallDelta,
		ContentIndex: 1,
		Partial:      &llm.AssistantMessage{StopReason: llm.StopReasonPending},
	})
	if err != nil {
		t.Fatal(err)
	}
	if event.Name != "response.function_call_arguments.delta" {
		t.Fatalf("event name = %q", event.Name)
	}
	if !strings.Contains(string(event.Data), `"delta":""`) {
		t.Fatalf("event data = %s", event.Data)
	}
}

// TestEncodeEventRejectsUnknownEvent 验证未知核心事件不会被静默丢弃。
func TestEncodeEventRejectsUnknownEvent(t *testing.T) {
	if _, err := EncodeEvent(llm.ResponseEvent{Type: "unknown"}); err == nil {
		t.Fatal("EncodeEvent() error = nil, want unknown event error")
	}
}

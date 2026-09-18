// 本文件确保未收到完成事件的供应商流不能向客户端宣告成功。
package app

import (
	"github.com/leookun/devin-2api/internal/config"
	"github.com/leookun/devin-2api/internal/llm"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTruncatedProviderEventsEmitErrorWithoutSuccess(t *testing.T) {
	for _, path := range []string{"/v1/chat/completions", "/v1/responses", "/v1/messages"} {
		for _, empty := range []bool{false, true} {
			fake := &fakeAdapter{}
			if !empty {
				fake.events = []llm.ResponseEvent{{Type: llm.ResponseEventStart, Partial: &llm.AssistantMessage{Model: "routed-model", StopReason: llm.StopReasonPending}}}
			}
			app := New(fake, config.ServerConfig{Listen: ":0"}, nil)
			body := `{"model":"swe-2-max","messages":[{"role":"user","content":"hi"}],"input":"hi","max_tokens":64,"stream":true}`
			response := httptest.NewRecorder()
			app.Router().ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
			wire := response.Body.String()
			if !strings.Contains(wire, "upstream event stream ended without a completion event") || strings.Contains(wire, "[DONE]") || strings.Contains(wire, `"type":"response.completed"`) || strings.Contains(wire, `"type":"message_stop"`) {
				t.Fatalf("%s empty=%v: unexpected terminal response: %s", path, empty, wire)
			}
		}
	}
}

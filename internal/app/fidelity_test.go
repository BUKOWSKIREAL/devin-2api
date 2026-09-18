// 本文件验证生成参数不会在 HTTP 路由到适配器之间再次丢失。
package app

import (
	"github.com/leookun/devin-2api/internal/config"
	"github.com/leookun/devin-2api/internal/llm"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPGenerationControlsReachAdapter(t *testing.T) {
	cases := []struct {
		// Path 是被测协议入口。
		Path string
		// Body 是最小有效客户端请求。
		Body string
	}{
		{"/v1/chat/completions", `{"model":"swe-2-high","messages":[{"role":"user","content":"hi"}],"max_tokens":17,"temperature":0,"top_p":0.5,"tool_choice":"none"}`},
		{"/v1/responses", `{"model":"swe-2-high","input":"hi","max_output_tokens":17,"temperature":0,"top_p":0.5,"tool_choice":"none"}`},
		{"/v1/messages", `{"model":"swe-2-high","messages":[{"role":"user","content":"hi"}],"max_tokens":17,"temperature":0,"top_p":0.5,"tool_choice":{"type":"none"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.Path, func(t *testing.T) {
			final := &llm.AssistantMessage{Model: "swe-2-high", Content: []llm.Content{llm.TextContent{Text: "ok"}}, StopReason: llm.StopReasonStop}
			fake := &fakeAdapter{events: []llm.ResponseEvent{{Type: llm.ResponseEventDone, Message: final, Reason: llm.StopReasonStop}}}
			app := New(fake, config.ServerConfig{Listen: ":0"}, nil)
			out := httptest.NewRecorder()
			app.Router().ServeHTTP(out, httptest.NewRequest(http.MethodPost, tc.Path, strings.NewReader(tc.Body)))
			if out.Code != 200 {
				t.Fatalf("status=%d: %s", out.Code, out.Body.String())
			}
			g := fake.lastRequest.Generation
			if g.MaxOutputTokens == nil || *g.MaxOutputTokens != 17 || g.Temperature == nil || *g.Temperature != 0 || g.TopP == nil || *g.TopP != 0.5 || g.ToolChoice == nil || g.ToolChoice.Mode != "none" {
				t.Fatalf("lost controls: %+v", g)
			}
		})
	}
}

func TestUnsupportedControlReturns400(t *testing.T) {
	app := New(&fakeAdapter{}, config.ServerConfig{Listen: ":0"}, nil)
	out := httptest.NewRecorder()
	app.Router().ServeHTTP(out, httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"swe-2-high","messages":[{"role":"user","content":"hi"}],"response_format":{"type":"json_object"}}`)))
	if out.Code != 400 || !strings.Contains(out.Body.String(), "response_format") {
		t.Fatalf("status=%d: %s", out.Code, out.Body.String())
	}
}

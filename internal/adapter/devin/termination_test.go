// 本文件通过真实 Connect 客户端校验截断帧、结束帧和模型停止原因的组合。
package devin

import (
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leookun/devin-2api/internal/llm"
	"google.golang.org/protobuf/proto"
	devinproto "local/devinproto"
)

func TestConnectTerminationValidation(t *testing.T) {
	cases := []struct {
		// Name 标识协议故障场景。
		Name string
		// Stop 是模型返回的停止原因；零表示省略。
		Stop devinproto.ExaCodeiumCommonPb_StopReason
		// End 为 Connect 结束帧；空表示 HTTP 提前结束。
		End string
		// Partial 模拟一帧 Protobuf 未接收完整。
		Partial bool
		// Fail 标记应产生错误而非成功终止。
		Fail bool
	}{
		{Name: "clean EndStream without optional stop", End: `{}`},
		{Name: "HTTP EOF without EndStream", Fail: true},
		{Name: "stop cannot mask missing EndStream", Stop: 2, Fail: true},
		{Name: "truncated protobuf envelope", Partial: true, Fail: true},
		{Name: "late trailer error overrides stop", Stop: 2, End: `{"error":{"code":"internal","message":"late upstream failure"}}`, Fail: true},
		{Name: "valid stop and EndStream", Stop: 2, End: `{}`},
		{Name: "unknown stop is not success", Stop: 999, End: `{}`, Fail: true},
		{Name: "content filter is not success", Stop: 11, End: `{}`, Fail: true},
		{Name: "numerical failure is not success", Stop: 7, End: `{}`, Fail: true},
		{Name: "tool stop requires tool call", Stop: 10, End: `{}`, Fail: true},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/connect+proto")
				payload, _ := proto.Marshal(&devinproto.GetChatMessageResponse{DeltaText: proto.String("partial or complete"), StopReason: tc.Stop.Enum()})
				header := make([]byte, 5)
				binary.BigEndian.PutUint32(header[1:], uint32(len(payload)))
				w.Write(header)
				if tc.Partial {
					w.Write(payload[:len(payload)-1])
					return
				}
				w.Write(payload)
				if tc.End != "" {
					header[0] = 2
					binary.BigEndian.PutUint32(header[1:], uint32(len(tc.End)))
					w.Write(header)
					io.WriteString(w, tc.End)
				}
			}))
			defer server.Close()
			adapter, err := New(Config{BaseURL: server.URL, Token: "test-token", Model: "swe-2-max"})
			if err != nil {
				t.Fatal(err)
			}
			stream, err := adapter.Stream(context.Background(), llm.RequestMessages{Model: "swe-2-max"})
			if err != nil {
				t.Fatal(err)
			}
			var done, failed bool
			for {
				e, err := stream.Recv(context.Background())
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				done = done || e.Type == llm.ResponseEventDone
				failed = failed || e.Type == llm.ResponseEventError
			}
			if failed != tc.Fail || done == tc.Fail {
				t.Fatalf("done=%v failed=%v expected failure=%v", done, failed, tc.Fail)
			}
		})
	}
}

func TestMissingModelStopInfersValidatedToolTurn(t *testing.T) {
	d := newResponseDecoder("swe-2-max")
	d.start()
	d.decode(&devinproto.GetChatMessageResponse{DeltaToolCalls: []*devinproto.ExaCodeiumCommonPb_ChatToolCall{{Id: proto.String("c1"), Name: proto.String("probe"), ArgumentsJson: proto.String(`{}`)}}})
	events := d.finish(nil)
	last := events[len(events)-1]
	if last.Type != llm.ResponseEventDone || last.Reason != llm.StopReasonToolUse {
		t.Fatalf("tool turn ended incorrectly: %+v", last)
	}
}

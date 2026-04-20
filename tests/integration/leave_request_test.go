package integration_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOneLeaveRequestJSON(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Basic token" {
			t.Fatalf("expected Authorization header, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"body":"{\"code\":\"00000\",\"message\":\"ok\"}"}`))
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()

	cmd := newGoRunCommand(t, "./examples/one-leave/cmd/one-leave", "request",
		"--start-time", "2026-03-25 09:00",
		"--end-time", "2026-03-25 18:00",
		"--vacation-type", "annual_leave",
		"--leave-time-type", "0",
		"--reason", "事假",
		"--json",
	)
	cmd.Env = append(cmd.Env,
		"ONE_AI_AUTH_TOKEN=Basic token",
		"ONE_AI_JOB_NO=415327",
		"ONE_AI_LEAVE_CREATE_URL="+server.URL,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected command to succeed, got error: %v, output: %s", err, string(out))
	}
	if !strings.Contains(string(out), "\"ok\":true") || !strings.Contains(string(out), "\"status\":\"submitted\"") {
		t.Fatalf("unexpected output: %s", string(out))
	}
}

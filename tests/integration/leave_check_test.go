package integration_test

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOneLeaveCheckJSON(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if got := payload["job_no"]; got != "415327" {
			t.Fatalf("expected job_no in payload, got %#v", got)
		}

		leaveList, ok := payload["leave_list"].([]any)
		if !ok || len(leaveList) != 1 {
			t.Fatalf("expected leave_list with one item, got %#v", payload["leave_list"])
		}

		entry, ok := leaveList[0].(map[string]any)
		if !ok {
			t.Fatalf("expected leave_list entry object, got %#v", leaveList[0])
		}
		if got := entry["start_time"]; got != "2026-03-25 09:00" {
			t.Fatalf("unexpected start_time: %#v", got)
		}
		if got := entry["end_time"]; got != "2026-03-25 18:00" {
			t.Fatalf("unexpected end_time: %#v", got)
		}
		if got := entry["vacation_type"]; got != "annual_leave" {
			t.Fatalf("unexpected vacation_type: %#v", got)
		}
		if got := entry["leave_time_type"]; got != "0" {
			t.Fatalf("unexpected leave_time_type: %#v", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"body":"{\"code\":\"00000\",\"data\":{\"success\":[{\"computeResult\":{\"readableValue\":\"1天\"}}]}}"}`))
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()

	cmd := newGoRunCommand(t, "./examples/one-leave/cmd/one-leave", "check",
		"--start-time", "2026-03-25 09:00",
		"--end-time", "2026-03-25 18:00",
		"--vacation-type", "annual_leave",
		"--leave-time-type", "0",
		"--json",
	)
	cmd.Env = append(cmd.Env,
		"ONE_AI_AUTH_TOKEN=Basic token",
		"ONE_AI_JOB_NO=415327",
		"ONE_AI_LEAVE_CHECK_URL="+server.URL,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected command to succeed, got error: %v, output: %s", err, string(out))
	}
	if !strings.Contains(string(out), "\"ok\":true") || !strings.Contains(string(out), "1天") {
		t.Fatalf("unexpected output: %s", string(out))
	}
}

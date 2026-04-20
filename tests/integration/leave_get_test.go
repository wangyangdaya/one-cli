package integration_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOneLeaveGetJSON(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Basic token" {
			t.Fatalf("expected Authorization header, got %q", got)
		}
		if got := r.URL.Query().Get("job_no"); got != "415327" {
			t.Fatalf("expected job_no query param, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"0","data":[{"vacationTypeName":"带薪年假","vacationType":"annual_leave","description":"desc","readableQuotaValue":"5天"},{"vacationTypeName":"调休","vacationType":"day_off","description":"desc","readableQuotaValue":"8小时"}]}`))
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()

	cmd := newGoRunCommand(t, "./examples/one-leave/cmd/one-leave", "get", "--json")
	cmd.Env = append(os.Environ(),
		"ONE_AI_JOB_NO=415327",
		"ONE_AI_AUTH_TOKEN=Basic token",
		"ONE_AI_LEAVE_LIST_URL="+server.URL,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected command to succeed, got error: %v, output: %s", err, string(out))
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	jsonOut := lines[len(lines)-1]
	if !strings.Contains(jsonOut, "\"ok\":true") || !strings.Contains(jsonOut, "annual_leave") {
		t.Fatalf("unexpected output: %s", string(out))
	}
	if strings.Contains(jsonOut, "8小时") {
		t.Fatalf("expected readableQuotaValue to be removed for 调休, got: %s", string(out))
	}
}

func newGoRunCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()

	cacheDir := t.TempDir()
	cmd := exec.Command("go", append([]string{"run"}, args...)...)
	cmd.Dir = "../.."
	cmd.Env = append(os.Environ(),
		"GOCACHE="+filepath.Join(cacheDir, "gocache"),
		"GOTOOLCHAIN=local",
	)

	return cmd
}

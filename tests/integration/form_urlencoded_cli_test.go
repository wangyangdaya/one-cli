package integration_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/app"
)

func TestGeneratedGoCLIEncodesGETFormFieldsAsQueryParameters(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:     filepath.Join("..", "..", "examples", "bpm.yaml"),
		Output:    dir,
		Module:    "github.com/acme/bpm-cli",
		AppName:   "bpm-cli",
		SkillLang: "zh",
	}); err != nil {
		t.Fatalf("run generate: %v", err)
	}
	skill, err := os.ReadFile(filepath.Join(dir, "skills", "bpm", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill: %v", err)
	}
	for _, want := range []string{
		`--tqm '{"owner":"<user-uid>","taskState":1}'`,
		"TaskQueryModel 的 JSON 字符串；owner 必填（用户 uid），beginAfter/beginBefore 为毫秒时间戳，taskState=1 表示待办（查询参数；string）",
		"**`--tqm` JSON 字段（`TaskQueryModel`）：**",
		"| `owner` | 是 | `string` | 用户 uid，查询该用户的待办 |",
	} {
		if !strings.Contains(string(skill), want) {
			t.Fatalf("generated skill missing %q", want)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "" {
			t.Errorf("Content-Type = %q, want empty for GET query request", got)
		}
		if r.Body != nil && r.ContentLength != 0 {
			t.Errorf("GET request body length = %d, want 0", r.ContentLength)
		}
		for name, want := range map[string]string{
			"tqm":      `{"status":"pending & review"}`,
			"firstRow": "0",
			"rowCount": "20",
		} {
			if got := r.URL.Query().Get(name); got != want {
				t.Errorf("query parameter %s = %q, want %q", name, got, want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer server.Close()

	cmd := exec.Command(
		"go", "run", "./cmd/bpm-cli", "bpm", "task-query",
		"--tqm", `{"status":"pending & review"}`,
		"--first-row", "0",
		"--row-count", "20",
		"--header", "Job-No: encrypted-job-number",
		"--header", "Authorization: tool-token",
	)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"OPENCLI_BASE_URL="+server.URL,
		"GOCACHE="+filepath.Join(dir, ".gocache"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated task-query failed: %v, output: %s", err, out)
	}
	if !strings.Contains(string(out), "ok") {
		t.Fatalf("expected success output, got %s", out)
	}
}

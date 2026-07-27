package command_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/app"
)

func TestPackageCommandCreatesInstallableGroupedSkillsBundle(t *testing.T) {
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "vehicle-api")

	writePackageFixture(t, project, "bin/vehicle-cli", "#!/bin/sh\nexit 0\n", 0o755)
	writePackageFixture(t, project, "config/runtime.yaml", "base_url: https://api.example.test\n", 0o644)
	writePackageFixture(t, project, "skills/SKILL.md", "# Vehicle router\n\n- [export](export/SKILL.md)\n- [vbt_vehicle_info](vbt_vehicle_info/SKILL.md)\n", 0o644)
	writePackageFixture(t, project, "skills/README.md", "# Vehicle skills\n", 0o644)
	writePackageFixture(t, project, "skills/export/SKILL.md", "# Export\n\nvehicle-cli export list --json\n", 0o644)
	writePackageFixture(t, project, "skills/export/README.md", "export business notes\n", 0o644)
	writePackageFixture(t, project, "skills/export/assets/demo-request.json", "{}\n", 0o644)
	writePackageFixture(t, project, "skills/export/references/commands.md", "# Commands\n`vehicle-cli export list --help`\n", 0o644)
	writePackageFixture(t, project, "skills/export/references/workflows.md", "export workflow\n", 0o644)
	writePackageFixture(t, project, "skills/export/generation-report.md", "generated export report\n", 0o644)
	writePackageFixture(t, project, "skills/vbt_vehicle_info/SKILL.md", "# Vehicle info\n\nvehicle-cli vbt_vehicle_info get --json\n", 0o644)
	writePackageFixture(t, project, "skills/vbt_vehicle_info/references/commands.md", "# Commands\n`vehicle-cli vbt_vehicle_info get --help`\n", 0o644)

	binary := filepath.Join(t.TempDir(), "vehicle-cli")
	writePackageFixture(t, filepath.Dir(binary), filepath.Base(binary), "prebuilt vehicle cli\n", 0o755)

	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{"package", "--project", project, "--output", output, "--binary", binary})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute package: %v", err)
	}

	for _, rel := range []string{
		"SKILL.md",
		"README.md",
		"bin/vehicle-cli",
		"config/runtime.yaml",
		"libexec/python",
		"libexec/node",
		"export/SKILL.md",
		"export/README.md",
		"export/assets/demo-request.json",
		"export/references/commands.md",
		"export/references/workflows.md",
		"export/generation-report.md",
		"vbt_vehicle_info/SKILL.md",
		"vbt_vehicle_info/references/commands.md",
	} {
		if _, err := os.Stat(filepath.Join(output, rel)); err != nil {
			t.Errorf("missing packaged path %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(output, "manifest.yaml")); !os.IsNotExist(err) {
		t.Fatalf("first package version must not generate manifest.yaml, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(output, "libexec", "vehicle-cli")); !os.IsNotExist(err) {
		t.Fatalf("bundle must not add a redundant CLI directory below libexec, got err=%v", err)
	}
}

func TestPackageCommandBuildsGoCLIAndLoadsPackagedRuntimeConfig(t *testing.T) {
	wantAuth := "Bearer packaged-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != wantAuth {
			t.Errorf("Authorization = %q, want %q", got, wantAuth)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer server.Close()

	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "pet-skills")
	runtimeSource := filepath.Join(t.TempDir(), "runtime.yaml")
	writePackageFixture(t, filepath.Dir(runtimeSource), filepath.Base(runtimeSource), fmt.Sprintf("base_url: %s\nauth:\n  type: bearer\n", server.URL), 0o644)
	t.Setenv("OPENCLI_AUTH_TOKEN", "packaged-token")
	if err := app.RunGenerate(app.GenerateOptions{
		Input:             filepath.Join("..", "..", "examples", "petstore.yaml"),
		Output:            project,
		Module:            "github.com/acme/packaged-petcli",
		AppName:           "petcli",
		Auth:              "token",
		RuntimeConfigPath: runtimeSource,
	}); err != nil {
		t.Fatalf("generate project: %v", err)
	}

	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{"package", "--project", project, "--output", output})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("package generated Go project: %v", err)
	}

	run := exec.Command(filepath.Join(output, "bin", "petcli"), "pet", "list", "--limit", "10")
	run.Dir = t.TempDir()
	run.Env = packageRuntimeEnv()
	result, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run packaged CLI outside bundle directory: %v\n%s", err, result)
	}
	if !strings.Contains(string(result), "ok") {
		t.Fatalf("packaged CLI output = %s", result)
	}
}

func writePackageFixture(t *testing.T, root, rel, content string, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write fixture %s: %v", rel, err)
	}
}

func packageRuntimeEnv() []string {
	env := make([]string, 0, len(os.Environ()))
	for _, value := range os.Environ() {
		name, _, _ := strings.Cut(value, "=")
		switch name {
		case "OPENCLI_BASE_URL", "OPENCLI_AUTH_TOKEN", "OPENCLI_API_KEY", "OPENCLI_CONFIG":
			continue
		}
		env = append(env, value)
	}
	return env
}

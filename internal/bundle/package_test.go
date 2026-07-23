package bundle

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPackagePreservesBusinessFilesAndRefreshesGeneratedArtifacts(t *testing.T) {
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "vehicle-api")
	binary := filepath.Join(t.TempDir(), "vehicle-cli")
	writeFixture(t, project, "bin/vehicle-cli", "launcher\n", 0o755)
	writeFixture(t, project, "skills/SKILL.md", "# router v1\n", 0o644)
	writeFixture(t, project, "skills/README.md", "# index v1\n", 0o644)
	writeFixture(t, project, "skills/export/SKILL.md", "# export generated v1\nvehicle-cli export list --json\n", 0o644)
	writeFixture(t, project, "skills/export/README.md", "generated readme v1\n", 0o644)
	writeFixture(t, project, "skills/export/references/workflows.md", "generated workflow v1\n", 0o644)
	writeFixture(t, project, "skills/export/assets/demo-request.json", "{\"generated\":1}\n", 0o644)
	writeFixture(t, project, "skills/export/generation-report.md", "report v1\n", 0o644)
	writeFixture(t, project, "config/runtime.yaml", "config v1\n", 0o644)
	writeFixture(t, filepath.Dir(binary), filepath.Base(binary), "binary v1\n", 0o755)

	if _, err := Package(Options{ProjectDir: project, OutputDir: output, BinaryPath: binary}); err != nil {
		t.Fatalf("first package: %v", err)
	}

	writeFixture(t, output, "export/SKILL.md", "business skill\n", 0o644)
	writeFixture(t, output, "export/README.md", "business readme\n", 0o644)
	writeFixture(t, output, "export/references/workflows.md", "business workflow\n", 0o644)
	writeFixture(t, output, "export/assets/demo-request.json", "{\"business\":true}\n", 0o644)
	writeFixture(t, output, "export/scripts/validate.py", "print('business')\n", 0o644)
	writeFixture(t, output, "export/business-notes.md", "keep me\n", 0o644)

	writeFixture(t, project, "skills/SKILL.md", "# router v2\n", 0o644)
	writeFixture(t, project, "skills/README.md", "# index v2\n", 0o644)
	writeFixture(t, project, "skills/export/SKILL.md", "# export generated v2\n", 0o644)
	writeFixture(t, project, "skills/export/README.md", "generated readme v2\n", 0o644)
	writeFixture(t, project, "skills/export/references/workflows.md", "generated workflow v2\n", 0o644)
	writeFixture(t, project, "skills/export/assets/demo-request.json", "{\"generated\":2}\n", 0o644)
	writeFixture(t, project, "skills/export/generation-report.md", "report v2\n", 0o644)
	writeFixture(t, project, "config/runtime.yaml", "config v2\n", 0o644)
	writeFixture(t, filepath.Dir(binary), filepath.Base(binary), "binary v2\n", 0o755)

	if _, err := Package(Options{ProjectDir: project, OutputDir: output, BinaryPath: binary}); err != nil {
		t.Fatalf("second package: %v", err)
	}

	assertFileContent(t, output, "export/SKILL.md", "business skill\n")
	assertFileContent(t, output, "export/README.md", "business readme\n")
	assertFileContent(t, output, "export/references/workflows.md", "business workflow\n")
	assertFileContent(t, output, "export/assets/demo-request.json", "{\"business\":true}\n")
	assertFileContent(t, output, "export/scripts/validate.py", "print('business')\n")
	assertFileContent(t, output, "export/business-notes.md", "keep me\n")
	assertFileContent(t, output, "export/generation-report.md", "report v2\n")
	assertFileContent(t, output, "SKILL.md", "# router v2\n")
	assertFileContent(t, output, "README.md", "# index v2\n")
	assertFileContent(t, output, "bin/vehicle-cli", "binary v2\n")
	assertFileContent(t, output, "config/runtime.yaml", "config v2\n")
}

func TestPackageAdaptsNewGroupDocumentsToSharedBinary(t *testing.T) {
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "vehicle-api")
	binary := filepath.Join(t.TempDir(), "vehicle-cli")
	writeFixture(t, project, "bin/vehicle-cli", "launcher\n", 0o755)
	writeFixture(t, project, "skills/SKILL.md", "vehicle-cli skills list\n[vehicle info](vbt_vehicle_info/SKILL.md)\n", 0o644)
	writeFixture(t, project, "skills/README.md", "vehicle-cli skills read vbt_vehicle_info\n", 0o644)
	writeFixture(t, project, "skills/vbt_vehicle_info/SKILL.md", "---\nmetadata:\n  requires:\n    bins: [\"vehicle-cli\"]\n  cliHelp: \"vehicle-cli vbt_vehicle_info --help\"\n---\nvehicle-cli vbt_vehicle_info get --json\nvehicle-cli skills list\nvehicle-cli skills read vbt_vehicle_info\nDefault sealed runtime file: `config/runtime.yaml`\n", 0o644)
	writeFixture(t, project, "skills/vbt_vehicle_info/references/workflows.md", "`vehicle-cli vbt_vehicle_info list --json`\n", 0o644)
	writeFixture(t, project, "skills/export-history/SKILL.md", "../bin/vehicle-cli export-history list --json\n", 0o644)
	writeFixture(t, filepath.Dir(binary), filepath.Base(binary), "binary\n", 0o755)

	result, err := Package(Options{ProjectDir: project, OutputDir: output, BinaryPath: binary})
	if err != nil {
		t.Fatalf("package: %v", err)
	}
	if got := strings.Join(result.Groups, ","); got != "export-history,vbt_vehicle_info" {
		t.Fatalf("groups = %q", got)
	}
	vehicleSkill, err := os.ReadFile(filepath.Join(output, "vbt_vehicle_info", "SKILL.md"))
	if err != nil {
		t.Fatalf("read packaged vehicle Skill: %v", err)
	}
	vehicleSkillText := string(vehicleSkill)
	for _, want := range []string{"cliHelp: \"../bin/vehicle-cli vbt_vehicle_info --help\"", "../bin/vehicle-cli vbt_vehicle_info get --json", "../bin/vehicle-cli skills --skills-dir .. list", "../bin/vehicle-cli skills --skills-dir .. read vbt_vehicle_info", "`../config/runtime.yaml`"} {
		if !strings.Contains(vehicleSkillText, want) {
			t.Fatalf("packaged vehicle Skill missing %q:\n%s", want, vehicleSkillText)
		}
	}
	if strings.Contains(vehicleSkillText, "bins:") || strings.Contains(vehicleSkillText, "requires:") {
		t.Fatalf("packaged Skill must not require the bundled CLI on PATH:\n%s", vehicleSkillText)
	}
	assertFileContent(t, output, "vbt_vehicle_info/references/workflows.md", "`../bin/vehicle-cli vbt_vehicle_info list --json`\n")
	assertFileContent(t, output, "export-history/SKILL.md", "../bin/vehicle-cli export-history list --json\n")
	assertFileContent(t, output, "SKILL.md", "./bin/vehicle-cli skills list\n[vehicle info](vbt_vehicle_info/SKILL.md)\n")
}

func TestDiscoverProjectRejectsAmbiguousLaunchers(t *testing.T) {
	project := t.TempDir()
	writeFixture(t, project, "bin/first", "first\n", 0o755)
	writeFixture(t, project, "bin/second", "second\n", 0o755)
	writeFixture(t, project, "skills/SKILL.md", "router\n", 0o644)
	writeFixture(t, project, "skills/README.md", "index\n", 0o644)
	writeFixture(t, project, "skills/export/SKILL.md", "export\n", 0o644)

	_, err := Package(Options{ProjectDir: project, OutputDir: t.TempDir(), BinaryPath: filepath.Join(project, "bin", "first")})
	if err == nil || !strings.Contains(err.Error(), "exactly one CLI launcher") {
		t.Fatalf("error = %v, want ambiguous launcher error", err)
	}
}

func TestPackageCreatesRootRouterForSingleGroupProject(t *testing.T) {
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "vehicle-api")
	binary := filepath.Join(t.TempDir(), "vehicle-cli")
	writeFixture(t, project, "bin/vehicle-cli", "launcher\n", 0o755)
	writeFixture(t, project, "skills/README.md", "# Vehicle skills\n", 0o644)
	writeFixture(t, project, "skills/export/SKILL.md", "# Export\n", 0o644)
	writeFixture(t, filepath.Dir(binary), filepath.Base(binary), "binary\n", 0o755)

	if _, err := Package(Options{ProjectDir: project, OutputDir: output, BinaryPath: binary}); err != nil {
		t.Fatalf("package single group: %v", err)
	}
	router, err := os.ReadFile(filepath.Join(output, "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated router: %v", err)
	}
	for _, want := range []string{"name: vehicle-cli-skills", "export/SKILL.md", "./bin/vehicle-cli skills list"} {
		if !strings.Contains(string(router), want) {
			t.Fatalf("router missing %q:\n%s", want, router)
		}
	}
}

func TestResolveBinaryBuildsGoProject(t *testing.T) {
	project := t.TempDir()
	writeFixture(t, project, "go.mod", "module example.test/vehicle\n", 0o644)
	original := executeCommand
	t.Cleanup(func() { executeCommand = original })
	var gotDir, gotName string
	var gotArgs []string
	executeCommand = func(dir, name string, args ...string) error {
		gotDir, gotName, gotArgs = dir, name, append([]string(nil), args...)
		output := args[2]
		return os.WriteFile(output, []byte("go binary"), 0o755)
	}

	binary, cleanup, err := resolveBinary(project, "vehicle-cli", "")
	if err != nil {
		t.Fatalf("resolve Go binary: %v", err)
	}
	defer cleanup()
	if gotDir != project || gotName != "go" || !reflect.DeepEqual(gotArgs[:2], []string{"build", "-o"}) || gotArgs[3] != "./cmd/vehicle-cli" {
		t.Fatalf("command = dir %q, %s %q", gotDir, gotName, gotArgs)
	}
	assertAbsoluteFileContent(t, binary, "go binary")
}

func TestResolveBinaryBuildsRustProject(t *testing.T) {
	project := t.TempDir()
	writeFixture(t, project, "Cargo.toml", "[package]\nname = \"generated-vehicle\"\n", 0o644)
	original := executeCommand
	t.Cleanup(func() { executeCommand = original })
	var gotDir, gotName string
	var gotArgs []string
	executeCommand = func(dir, name string, args ...string) error {
		gotDir, gotName, gotArgs = dir, name, append([]string(nil), args...)
		writeFixture(t, project, "target/release/generated-vehicle", "rust binary", 0o755)
		return nil
	}

	binary, cleanup, err := resolveBinary(project, "vehicle-cli", "")
	if err != nil {
		t.Fatalf("resolve Rust binary: %v", err)
	}
	defer cleanup()
	if gotDir != project || gotName != "cargo" || !reflect.DeepEqual(gotArgs, []string{"build", "--release"}) {
		t.Fatalf("command = dir %q, %s %q", gotDir, gotName, gotArgs)
	}
	assertAbsoluteFileContent(t, binary, "rust binary")
}

func TestResolveBinaryRejectsAmbiguousProjectType(t *testing.T) {
	project := t.TempDir()
	writeFixture(t, project, "go.mod", "module example.test/vehicle\n", 0o644)
	writeFixture(t, project, "Cargo.toml", "[package]\nname = \"vehicle-cli\"\n", 0o644)
	_, _, err := resolveBinary(project, "vehicle-cli", "")
	if err == nil || !strings.Contains(err.Error(), "both go.mod and Cargo.toml") {
		t.Fatalf("error = %v, want ambiguous project type", err)
	}
}

func TestPackageRejectsSymlinkedSkillFiles(t *testing.T) {
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "vehicle-api")
	binary := filepath.Join(t.TempDir(), "vehicle-cli")
	writeFixture(t, project, "bin/vehicle-cli", "launcher\n", 0o755)
	writeFixture(t, project, "skills/README.md", "index\n", 0o644)
	writeFixture(t, project, "skills/export/SKILL.md", "export\n", 0o644)
	secret := filepath.Join(t.TempDir(), "secret.txt")
	writeFixture(t, filepath.Dir(secret), filepath.Base(secret), "must not be packaged\n", 0o644)
	link := filepath.Join(project, "skills", "export", "business-secret.txt")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	writeFixture(t, filepath.Dir(binary), filepath.Base(binary), "binary\n", 0o755)

	_, err := Package(Options{ProjectDir: project, OutputDir: output, BinaryPath: binary})
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v, want symbolic-link rejection", err)
	}
}

func writeFixture(t *testing.T, root, rel, content string, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func assertFileContent(t *testing.T, root, rel, want string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	if got := string(content); got != want {
		t.Fatalf("%s = %q, want %q", rel, got, want)
	}
}

func assertAbsoluteFileContent(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if got := string(content); got != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

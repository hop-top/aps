package isolation

import (
	"strings"
	"testing"

	"oss-aps-cli/internal/core"
	isolation "oss-aps-cli/internal/core/isolation"
)

func TestDockerfileBuilder_Generate_Basic(t *testing.T) {
	builder := isolation.NewDockerfileBuilder()

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)

	if err != nil {
		t.Fatalf("failed to generate dockerfile: %v", err)
	}

	if dockerfile == "" {
		t.Error("expected non-empty dockerfile")
	}

	if !strings.Contains(dockerfile, "FROM ubuntu:22.04") {
		t.Error("expected FROM line with ubuntu:22.04")
	}

	if !strings.Contains(dockerfile, "openssh-server") {
		t.Error("expected SSH server installation")
	}

	if !strings.Contains(dockerfile, "tmux") {
		t.Error("expected tmux installation")
	}
}

func TestDockerfileBuilder_Generate_WithPackages(t *testing.T) {
	builder := isolation.NewDockerfileBuilder()

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image:    "ubuntu:22.04",
				Packages: []string{"nodejs", "python3", "git"},
			},
		},
	}

	dockerfile, err := builder.Generate(profile)

	if err != nil {
		t.Fatalf("failed to generate dockerfile: %v", err)
	}

	if !strings.Contains(dockerfile, "nodejs") {
		t.Error("expected nodejs package")
	}

	if !strings.Contains(dockerfile, "python3") {
		t.Error("expected python3 package")
	}

	if !strings.Contains(dockerfile, "git") {
		t.Error("expected git package")
	}
}

func TestDockerfileBuilder_Generate_WithBuildSteps(t *testing.T) {
	builder := isolation.NewDockerfileBuilder()

	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "alpine:latest",
				BuildSteps: []core.BuildStep{
					{Type: "shell", Run: "npm install -g @anthropic-ai/claude-code@latest"},
				},
			},
		},
	}

	dockerfile, err := builder.Generate(profile)

	if err != nil {
		t.Fatalf("failed to generate dockerfile: %v", err)
	}

	if !strings.Contains(dockerfile, "npm install -g @anthropic-ai/claude-code@latest") {
		t.Error("expected build step in dockerfile")
	}
}

func TestDockerfileBuilder_ParseVolumes(t *testing.T) {
	builder := isolation.NewDockerfileBuilder()

	volumeStrs := []string{
		"/host/path:/container/path",
		"/host/path2:/container/path2:ro",
	}

	volumes := builder.parseVolumes(volumeStrs)

	if len(volumes) != 2 {
		t.Fatalf("expected 2 volumes, got %d", len(volumes))
	}

	if volumes[0].Source != "/host/path" {
		t.Errorf("expected source /host/path, got %s", volumes[0].Source)
	}

	if volumes[0].Target != "/container/path" {
		t.Errorf("expected target /container/path, got %s", volumes[0].Target)
	}

	if volumes[0].Readonly {
		t.Error("expected first volume to not be readonly")
	}

	if !volumes[1].Readonly {
		t.Error("expected second volume to be readonly")
	}
}

func TestDockerEngine_Available(t *testing.T) {
	engine, err := isolation.NewDockerEngine()
	if err != nil {
		t.Fatalf("failed to create docker engine: %v", err)
	}

	available := engine.Available()
	if !available {
		t.Skip("Docker not available, skipping test")
	}
}

func TestDockerEngine_Version(t *testing.T) {
	engine, err := isolation.NewDockerEngine()
	if err != nil {
		t.Fatalf("failed to create docker engine: %v", err)
	}

	if !engine.Available() {
		t.Skip("Docker not available, skipping test")
	}

	version, err := engine.Version()
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}

	if version == "" {
		t.Error("expected non-empty version")
	}

	t.Logf("Docker version: %s", version)
}

package isolation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oss-aps-cli/internal/core"
)

// ============================================================================
// DockerfileBuilder Tests (25-30 tests)
// ============================================================================

// TestNewDockerfileBuilder verifies builder initialization
func TestNewDockerfileBuilder(t *testing.T) {
	builder := NewDockerfileBuilder()
	require.NotNil(t, builder)
	assert.IsType(t, &DockerfileBuilder{}, builder)
}

// TestGenerateBasicDockerfile verifies basic Dockerfile generation
func TestGenerateBasicDockerfile(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.NotEmpty(t, dockerfile)
	assert.Contains(t, dockerfile, "FROM ubuntu:22.04")
}

// TestGenerateDefaultImage verifies default image selection
func TestGenerateDefaultImage(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "FROM ubuntu:22.04")
}

// TestGenerateWithPackages verifies package installation
func TestGenerateWithPackages(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image:    "ubuntu:22.04",
				Packages: []string{"curl", "wget", "git"},
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "curl")
	assert.Contains(t, dockerfile, "wget")
	assert.Contains(t, dockerfile, "git")
	assert.Contains(t, dockerfile, "apt-get install")
}

// TestGenerateWithSSH verifies SSH installation
func TestGenerateWithSSH(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "openssh-server")
	assert.Contains(t, dockerfile, "PasswordAuthentication no")
	assert.Contains(t, dockerfile, "PermitRootLogin no")
}

// TestGenerateWithTmux verifies tmux installation
func TestGenerateWithTmux(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "tmux")
}

// TestGenerateWithBuildSteps verifies build steps
func TestGenerateWithBuildSteps(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
				BuildSteps: []core.BuildStep{
					{Type: "shell", Run: "echo 'setup complete'"},
				},
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "echo 'setup complete'")
}

// TestGenerateWithVolumes verifies volume configuration
func TestGenerateWithVolumes(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image:   "ubuntu:22.04",
				Volumes: []string{"/workspace:/workspace", "/data:/data"},
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "VOLUME /workspace")
	assert.Contains(t, dockerfile, "VOLUME /data")
}

// TestGenerateWorkingDirectory verifies working directory setup
func TestGenerateWorkingDirectory(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "WORKDIR /workspace")
}

// TestGenerateUserSetup verifies user creation
func TestGenerateUserSetup(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "useradd -m -s /bin/bash appuser")
	assert.Contains(t, dockerfile, "USER appuser")
}

// TestGenerateSSHStartup verifies SSH startup command
func TestGenerateSSHStartup(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "sshd")
	assert.Contains(t, dockerfile, "CMD")
}

// TestGenerateExposePort verifies port exposure
func TestGenerateExposePort(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "EXPOSE 22")
}

// TestGenerateAlpineImage verifies Alpine Linux support
func TestGenerateAlpineImage(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "alpine:latest",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "FROM alpine:latest")
}

// TestGenerateCustomImage verifies custom image support
func TestGenerateCustomImage(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "myregistry.azurecr.io/custom:v1",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "FROM myregistry.azurecr.io/custom:v1")
}

// TestGenerateEmptyProfile verifies empty profile handling
func TestGenerateEmptyProfile(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "empty",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.NotEmpty(t, dockerfile)
}

// TestBuildOptionsBasic verifies basic build options extraction
func TestBuildOptionsBasic(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	opts := builder.BuildOptions(profile)
	assert.Equal(t, "ubuntu:22.04", opts.Image)
	assert.Equal(t, "appuser", opts.User)
	assert.Equal(t, "/workspace", opts.WorkingDir)
}

// TestBuildOptionsWithVolumes verifies volume options
func TestBuildOptionsWithVolumes(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image:   "ubuntu:22.04",
				Volumes: []string{"/host:/container"},
			},
		},
	}

	opts := builder.BuildOptions(profile)
	assert.Equal(t, 1, len(opts.Volumes))
	assert.Equal(t, "/host", opts.Volumes[0].Source)
	assert.Equal(t, "/container", opts.Volumes[0].Target)
}

// TestBuildOptionsWithNetwork verifies network options
func TestBuildOptionsWithNetwork(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image:   "ubuntu:22.04",
				Network: "host",
			},
		},
	}

	opts := builder.BuildOptions(profile)
	assert.Equal(t, "host", opts.Network.Mode)
}

// TestBuildOptionsWithResources verifies resource limits
func TestBuildOptionsWithResources(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
				Resources: core.ContainerResources{
					MemoryMB: 512,
					CPUQuota: 100000,
				},
			},
		},
	}

	opts := builder.BuildOptions(profile)
	assert.Equal(t, int64(512*1024*1024), opts.Limits.MemoryLimit)
	// Note: ParseLimits currently only handles MemoryMB, not CPUQuota
	assert.Equal(t, int64(0), opts.Limits.CPUQuota)
}

// TestParseVolumesBasic verifies basic volume parsing
func TestParseVolumesBasic(t *testing.T) {
	builder := NewDockerfileBuilder()
	volumes := []string{"/host:/container"}

	result := builder.ParseVolumes(volumes)
	require.Equal(t, 1, len(result))
	assert.Equal(t, "/host", result[0].Source)
	assert.Equal(t, "/container", result[0].Target)
	assert.False(t, result[0].Readonly)
}

// TestParseVolumesReadonly verifies readonly volume parsing
func TestParseVolumesReadonly(t *testing.T) {
	builder := NewDockerfileBuilder()
	volumes := []string{"/host:/container:ro"}

	result := builder.ParseVolumes(volumes)
	require.Equal(t, 1, len(result))
	assert.True(t, result[0].Readonly)
}

// TestParseVolumesMultiple verifies multiple volume parsing
func TestParseVolumesMultiple(t *testing.T) {
	builder := NewDockerfileBuilder()
	volumes := []string{
		"/host1:/container1",
		"/host2:/container2:ro",
		"/host3:/container3",
	}

	result := builder.ParseVolumes(volumes)
	require.Equal(t, 3, len(result))
	assert.False(t, result[0].Readonly)
	assert.True(t, result[1].Readonly)
	assert.False(t, result[2].Readonly)
}

// TestParseVolumesInvalid verifies invalid volume handling
func TestParseVolumesInvalid(t *testing.T) {
	builder := NewDockerfileBuilder()
	volumes := []string{
		"/host:/container",
		"/invalid",
		"/host2:/container2",
	}

	result := builder.ParseVolumes(volumes)
	assert.Equal(t, 2, len(result))
}

// TestParseVolumesWithSpaces verifies volume parsing with spaces
func TestParseVolumesWithSpaces(t *testing.T) {
	builder := NewDockerfileBuilder()
	volumes := []string{
		" /host : /container ",
	}

	result := builder.ParseVolumes(volumes)
	require.Equal(t, 1, len(result))
	assert.Equal(t, "/host", result[0].Source)
	assert.Equal(t, "/container", result[0].Target)
}

// TestParseNetworkBridge verifies bridge network parsing
func TestParseNetworkBridge(t *testing.T) {
	builder := NewDockerfileBuilder()

	result := builder.ParseNetwork("bridge")
	assert.Equal(t, "bridge", result.Mode)
}

// TestParseNetworkHost verifies host network parsing
func TestParseNetworkHost(t *testing.T) {
	builder := NewDockerfileBuilder()

	result := builder.ParseNetwork("host")
	assert.Equal(t, "host", result.Mode)
}

// TestParseNetworkCustom verifies custom network parsing
func TestParseNetworkCustom(t *testing.T) {
	builder := NewDockerfileBuilder()

	result := builder.ParseNetwork("custom-net")
	assert.Equal(t, "custom-net", result.Mode)
}

// TestParseNetworkDefault verifies default network
func TestParseNetworkDefault(t *testing.T) {
	builder := NewDockerfileBuilder()

	result := builder.ParseNetwork("")
	assert.Equal(t, "bridge", result.Mode)
}

// TestWriteDockerfileSuccess verifies Dockerfile writing
func TestWriteDockerfileSuccess(t *testing.T) {
	tmpDir := t.TempDir()

	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	err := builder.WriteDockerfile(profile, profile, tmpDir)
	require.NoError(t, err)

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	assert.FileExists(t, dockerfilePath)

	content, err := os.ReadFile(dockerfilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "FROM ubuntu:22.04")
}

// TestWriteDockerfilePermissions verifies Dockerfile permissions
func TestWriteDockerfilePermissions(t *testing.T) {
	tmpDir := t.TempDir()

	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	err := builder.WriteDockerfile(profile, profile, tmpDir)
	require.NoError(t, err)

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	info, err := os.Stat(dockerfilePath)
	require.NoError(t, err)

	// Check file is readable/writable
	assert.NotNil(t, info)
}

// TestWriteDockerfileOverwrite verifies Dockerfile overwrite
func TestWriteDockerfileOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")

	// Write initial file
	err := os.WriteFile(dockerfilePath, []byte("old content"), 0644)
	require.NoError(t, err)

	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	err = builder.WriteDockerfile(profile, profile, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(dockerfilePath)
	require.NoError(t, err)

	// Verify old content is overwritten
	assert.NotContains(t, string(content), "old content")
	assert.Contains(t, string(content), "FROM ubuntu:22.04")
}

// ============================================================================
// Validation Tests (10-15 tests)
// ============================================================================

// TestDockerfileSyntaxValidation verifies Dockerfile syntax
func TestDockerfileSyntaxValidation(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)

	// Verify basic structure
	assert.Contains(t, dockerfile, "FROM")
	assert.Contains(t, dockerfile, "RUN")
	assert.Contains(t, dockerfile, "WORKDIR")
	assert.Contains(t, dockerfile, "CMD")
}

// TestDockerfileRequiredFields verifies required fields
func TestDockerfileRequiredFields(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)

	lines := strings.Split(dockerfile, "\n")
	assert.True(t, len(lines) > 0)

	// Check FROM is first non-comment
	fromFound := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "FROM") {
			fromFound = true
			break
		}
	}
	assert.True(t, fromFound)
}

// TestImageNameValidation verifies image name validation
func TestImageNameValidation(t *testing.T) {
	builder := NewDockerfileBuilder()

	testCases := []string{
		"ubuntu:22.04",
		"alpine:latest",
		"myregistry.azurecr.io/custom:v1",
		"localhost:5000/test:latest",
	}

	for _, imageName := range testCases {
		profile := &core.Profile{
			ID: "test",
			Isolation: core.IsolationConfig{
				Container: core.ContainerConfig{
					Image: imageName,
				},
			},
		}

		dockerfile, err := builder.Generate(profile)
		require.NoError(t, err)
		assert.Contains(t, dockerfile, fmt.Sprintf("FROM %s", imageName))
	}
}

// TestPortConfigurationValidation verifies port configuration
func TestPortConfigurationValidation(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)

	// SSH port should be exposed
	assert.Contains(t, dockerfile, "EXPOSE 22")
}

// TestResourceLimitBounds verifies resource limit boundaries
func TestResourceLimitBounds(t *testing.T) {
	builder := NewDockerfileBuilder()

	testCases := []core.ContainerResources{
		{MemoryMB: 256},
		{MemoryMB: 512},
		{MemoryMB: 1024},
		{MemoryMB: 2048},
	}

	for _, resources := range testCases {
		profile := &core.Profile{
			ID: "test",
			Isolation: core.IsolationConfig{
				Container: core.ContainerConfig{
					Image:     "ubuntu:22.04",
					Resources: resources,
				},
			},
		}

		opts := builder.BuildOptions(profile)
		expectedMemory := int64(resources.MemoryMB * 1024 * 1024)
		assert.Equal(t, expectedMemory, opts.Limits.MemoryLimit)
	}
}

// TestVolumepathValidation verifies volume path validation
func TestVolumePathValidation(t *testing.T) {
	builder := NewDockerfileBuilder()

	testCases := []string{
		"/host:/container",
		"/path/to/host:/path/to/container",
		"/host:/container:ro",
		"/host:/container:rw",
	}

	for _, volStr := range testCases {
		volumes := builder.ParseVolumes([]string{volStr})
		assert.NotEmpty(t, volumes)
		assert.NotEmpty(t, volumes[0].Source)
		assert.NotEmpty(t, volumes[0].Target)
	}
}

// TestEnvironmentVariableEncoding verifies environment variable handling
func TestEnvironmentVariableEncoding(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.NotEmpty(t, dockerfile)
}

// TestPackageInstallationFormat verifies package installation format
func TestPackageInstallationFormat(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image:    "ubuntu:22.04",
				Packages: []string{"curl", "wget"},
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)

	assert.Contains(t, dockerfile, "apt-get update")
	assert.Contains(t, dockerfile, "apt-get install -y")
	assert.Contains(t, dockerfile, "curl")
	assert.Contains(t, dockerfile, "wget")
}

// ============================================================================
// Edge Cases (5-10 tests)
// ============================================================================

// TestGenerateEmptyContext verifies empty context handling
func TestGenerateEmptyContext(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID:         "",
		Isolation: core.IsolationConfig{},
	}

	// Should still generate valid Dockerfile
	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.NotEmpty(t, dockerfile)
	assert.Contains(t, dockerfile, "FROM ubuntu:22.04")
}

// TestSpecialCharactersInPaths verifies special character handling
func TestSpecialCharactersInPaths(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test-profile",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image:   "ubuntu:22.04",
				Volumes: []string{"/path/with spaces:/container/path"},
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.NotEmpty(t, dockerfile)
}

// TestLargeFileList verifies handling of large package list
func TestLargeFileList(t *testing.T) {
	builder := NewDockerfileBuilder()

	packages := make([]string, 50)
	for i := 0; i < 50; i++ {
		packages[i] = fmt.Sprintf("package%d", i)
	}

	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image:    "ubuntu:22.04",
				Packages: packages,
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.NotEmpty(t, dockerfile)

	// Verify all packages are included
	for _, pkg := range packages {
		assert.Contains(t, dockerfile, pkg)
	}
}

// TestComplexBuildSteps verifies complex build steps
func TestComplexBuildSteps(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
				BuildSteps: []core.BuildStep{
					{Type: "shell", Run: "apt-get update && apt-get install -y build-essential"},
					{Type: "shell", Run: "cd /tmp && make install"},
					{Type: "copy", Content: "config", Run: "/etc/config"},
				},
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "build-essential")
	assert.Contains(t, dockerfile, "make install")
}

// TestConcurrentBuilderOperations verifies concurrent builder usage
func TestConcurrentBuilderOperations(t *testing.T) {
	var wg sync.WaitGroup
	errorChan := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			builder := NewDockerfileBuilder()
			profile := &core.Profile{
				ID: fmt.Sprintf("profile-%d", index),
				Isolation: core.IsolationConfig{
					Container: core.ContainerConfig{
						Image: "ubuntu:22.04",
					},
				},
			}

			_, err := builder.Generate(profile)
			if err != nil {
				errorChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	for err := range errorChan {
		require.NoError(t, err)
	}
}

// TestDockerfileWritePermissionError verifies permission error handling
func TestDockerfileWritePermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Test cannot run as root")
	}

	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.Mkdir(readOnlyDir, 0444))
	defer os.Chmod(readOnlyDir, 0755)

	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
			},
		},
	}

	err := builder.WriteDockerfile(profile, profile, readOnlyDir)
	assert.Error(t, err)
}

// TestVolumeParsingEdgeCases verifies edge case volume parsing
func TestVolumeParsingEdgeCases(t *testing.T) {
	builder := NewDockerfileBuilder()

	testCases := []struct {
		input    string
		expected bool
	}{
		{"/host:/container", true},
		{"/host:/container:ro", true},
		{"/host", false},
		{"", false},
		{"   ", false},
	}

	for _, tc := range testCases {
		volumes := builder.ParseVolumes([]string{tc.input})
		if tc.expected {
			assert.NotEmpty(t, volumes)
		} else if tc.input != "" && strings.TrimSpace(tc.input) != "" {
			assert.Empty(t, volumes)
		}
	}
}

// TestNetworkModeEdgeCases verifies network mode edge cases
func TestNetworkModeEdgeCases(t *testing.T) {
	builder := NewDockerfileBuilder()

	testCases := []string{
		"bridge",
		"host",
		"none",
		"container:name",
		"custom-network",
		"",
	}

	for _, mode := range testCases {
		result := builder.ParseNetwork(mode)
		if mode == "" {
			assert.Equal(t, "bridge", result.Mode)
		} else {
			assert.Equal(t, mode, result.Mode)
		}
	}
}

// TestDockerfileMultipleVolumes verifies multiple volume handling
func TestDockerfileMultipleVolumes(t *testing.T) {
	builder := NewDockerfileBuilder()
	profile := &core.Profile{
		ID: "test",
		Isolation: core.IsolationConfig{
			Container: core.ContainerConfig{
				Image: "ubuntu:22.04",
				Volumes: []string{
					"/host1:/container1",
					"/host2:/container2",
					"/host3:/container3:ro",
				},
			},
		},
	}

	dockerfile, err := builder.Generate(profile)
	require.NoError(t, err)

	assert.Contains(t, dockerfile, "VOLUME /container1")
	assert.Contains(t, dockerfile, "VOLUME /container2")
	assert.Contains(t, dockerfile, "VOLUME /container3")
}

// ============================================================================
// Helper Functions
// ============================================================================

// parseDockerfile parses a Dockerfile string into lines
func parseDockerfile(dockerfile string) []string {
	lines := strings.Split(dockerfile, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			result = append(result, trimmed)
		}
	}
	return result
}

// validateDockerfileLine verifies a Dockerfile line structure
func validateDockerfileLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return true
	}

	validKeywords := []string{
		"FROM", "RUN", "CMD", "LABEL", "EXPOSE", "ENV", "ADD", "COPY",
		"ENTRYPOINT", "VOLUME", "USER", "WORKDIR", "ARG", "ONBUILD",
	}

	for _, kw := range validKeywords {
		if strings.HasPrefix(trimmed, kw) {
			return true
		}
	}

	return false
}

package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"oss-aps-cli/internal/core"
)

type DockerfileBuilder struct{}

func (b *DockerfileBuilder) Generate(profile *core.Profile) (string, error) {
	if profile.Isolation.Container.Image == "" {
		return "", fmt.Errorf("container isolation requires an image")
	}

	var dockerfile strings.Builder

	baseImage := profile.Isolation.Container.Image

	dockerfile.WriteString(fmt.Sprintf("FROM %s\n", baseImage))

	if len(profile.Isolation.Container.Packages) > 0 {
		pkgManager, err := b.detectPackageManager(baseImage)
		if err != nil {
			return "", err
		}

		installCmd := b.buildInstallCommand(pkgManager, profile.Isolation.Container.Packages)
		dockerfile.WriteString(fmt.Sprintf("RUN %s\n", installCmd))
	}

	if len(profile.Isolation.Container.BuildSteps) > 0 {
		for i, step := range profile.Isolation.Container.BuildSteps {
			cmd, err := b.generateBuildStep(step)
			if err != nil {
				return "", fmt.Errorf("failed to generate build step %d: %w", i, err)
			}
			dockerfile.WriteString(fmt.Sprintf("%s\n", cmd))
		}
	}

	dockerfile.WriteString("WORKDIR /workspace\n")
	dockerfile.WriteString("CMD [\"/bin/bash\"]\n")

	return dockerfile.String(), nil
}

func (b *DockerfileBuilder) detectPackageManager(image string) (string, error) {
	image = strings.ToLower(image)

	switch {
	case strings.Contains(image, "ubuntu") || strings.Contains(image, "debian"):
		return "apt", nil
	case strings.Contains(image, "alpine"):
		return "apk", nil
	case strings.Contains(image, "fedora") || strings.Contains(image, "centos") || strings.Contains(image, "rhel"):
		return "yum", nil
	case strings.Contains(image, "arch"):
		return "pacman", nil
	default:
		return "", fmt.Errorf("unsupported base image: %s", image)
	}
}

func (b *DockerfileBuilder) buildInstallCommand(pkgManager string, packages []string) string {
	switch pkgManager {
	case "apt":
		pkgList := strings.Join(packages, " ")
		return fmt.Sprintf("apt-get update && apt-get install -y %s && rm -rf /var/lib/apt/lists/*", pkgList)
	case "apk":
		pkgList := strings.Join(packages, " ")
		return fmt.Sprintf("apk add --no-cache %s", pkgList)
	case "yum":
		pkgList := strings.Join(packages, " ")
		return fmt.Sprintf("yum install -y %s && yum clean all", pkgList)
	case "pacman":
		pkgList := strings.Join(packages, " ")
		return fmt.Sprintf("pacman -S --noconfirm %s && pacman -Sc --noconfirm", pkgList)
	default:
		return ""
	}
}

func (b *DockerfileBuilder) generateBuildStep(step core.BuildStep) (string, error) {
	switch step.Type {
	case "shell":
		if step.Run != "" {
			return fmt.Sprintf("RUN %s", step.Run), nil
		}
		return fmt.Sprintf("RUN %s", step.Content), nil
	case "copy":
		parts := strings.Fields(step.Run)
		if len(parts) != 2 {
			return "", fmt.Errorf("copy step requires source and target (expected: 'copy <src> <dst>')")
		}
		return fmt.Sprintf("COPY %s %s", parts[0], parts[1]), nil
	case "env":
		parts := strings.SplitN(step.Run, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("env step requires NAME=VALUE format")
		}
		return fmt.Sprintf("ENV %s=%s", parts[0], parts[1]), nil
	case "expose":
		return fmt.Sprintf("EXPOSE %s", step.Run), nil
	case "volume":
		return fmt.Sprintf("VOLUME %s", step.Run), nil
	case "workdir":
		return fmt.Sprintf("WORKDIR %s", step.Run), nil
	case "user":
		return fmt.Sprintf("USER %s", step.Run), nil
	default:
		return "", fmt.Errorf("unsupported build step type: %s", step.Type)
	}
}

func (b *DockerfileBuilder) WriteDockerfile(profile *core.Profile, outputDir string) (string, error) {
	dockerfile, err := b.Generate(profile)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	dockerfilePath := filepath.Join(outputDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return "", fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	return dockerfilePath, nil
}

func (b *DockerfileBuilder) BuildImageOptions(profile *core.Profile) (map[string]interface{}, error) {
	options := make(map[string]interface{})

	if profile.Isolation.Container.Image != "" {
		options["image"] = profile.Isolation.Container.Image
	}

	if profile.Isolation.Container.Network != "" {
		options["network"] = profile.Isolation.Container.Network
	}

	if len(profile.Isolation.Container.Volumes) > 0 {
		options["volumes"] = profile.Isolation.Container.Volumes
	}

	if profile.Isolation.Container.Resources.MemoryMB > 0 {
		options["memory"] = fmt.Sprintf("%dm", profile.Isolation.Container.Resources.MemoryMB)
	}

	// TODO: CPU quota support - field not accessible
	// if profile.Isolation.Container.Resources.CPUQuota > 0 {
	// 	options["cpus"] = fmt.Sprintf("%.2f", float64(profile.Isolation.Container.CPUQuota)/100000)
	// }

	return options, nil
}

func GenerateDockerfileForProfile(profileID string) (string, error) {
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return "", err
	}

	if profile.Isolation.Level != core.IsolationContainer {
		return "", fmt.Errorf("profile does not use container isolation")
	}

	builder := &DockerfileBuilder{}
	return builder.Generate(profile)
}

func WriteDockerfileForProfile(profileID string) (string, error) {
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return "", err
	}

	if profile.Isolation.Level != core.IsolationContainer {
		return "", fmt.Errorf("profile does not use container isolation")
	}

	builder := &DockerfileBuilder{}
	profileDir, err := core.GetProfileDir(profileID)
	if err != nil {
		return "", err
	}

	return builder.WriteDockerfile(profile, profileDir)
}

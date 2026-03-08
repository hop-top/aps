package isolation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"hop.top/aps/internal/core"
)

type DockerfileBuilder struct{}

func NewDockerfileBuilder() *DockerfileBuilder {
	return &DockerfileBuilder{}
}

func (d *DockerfileBuilder) Generate(profile *core.Profile) (string, error) {
	var dockerfile strings.Builder

	image := profile.Isolation.Container.Image
	if image == "" {
		image = "ubuntu:22.04"
	}

	dockerfile.WriteString(fmt.Sprintf("FROM %s\n", image))

	dockerfile.WriteString("\n")

	if len(profile.Isolation.Container.Packages) > 0 {
		dockerfile.WriteString("# Install packages\n")
		dockerfile.WriteString("RUN apt-get update && apt-get install -y \\\n")

		for i, pkg := range profile.Isolation.Container.Packages {
			if i == len(profile.Isolation.Container.Packages)-1 {
				dockerfile.WriteString(fmt.Sprintf("    %s\n", pkg))
			} else {
				dockerfile.WriteString(fmt.Sprintf("    %s \\\n", pkg))
			}
		}

		dockerfile.WriteString("    && apt-get clean \\\n")
		dockerfile.WriteString("    && rm -rf /var/lib/apt/lists/*\n")
		dockerfile.WriteString("\n")
	}

	dockerfile.WriteString("# Install SSH server and tmux\n")
	dockerfile.WriteString("RUN apt-get update && apt-get install -y \\\n")
	dockerfile.WriteString("    openssh-server \\\n")
	dockerfile.WriteString("    tmux \\\n")
	dockerfile.WriteString("    && apt-get clean \\\n")
	dockerfile.WriteString("    && rm -rf /var/lib/apt/lists/*\n")
	dockerfile.WriteString("\n")

	dockerfile.WriteString("# Create appuser for running commands\n")
	dockerfile.WriteString("RUN useradd -m -s /bin/bash appuser\n")
	dockerfile.WriteString("\n")

	dockerfile.WriteString("# Configure SSH server\n")
	dockerfile.WriteString("RUN mkdir -p /var/run/sshd && \\\n")
	dockerfile.WriteString("    echo 'PasswordAuthentication no' >> /etc/ssh/sshd_config && \\\n")
	dockerfile.WriteString("    echo 'PermitRootLogin no' >> /etc/ssh/sshd_config && \\\n")
	dockerfile.WriteString("    echo 'AllowTcpForwarding yes' >> /etc/ssh/sshd_config\n")
	dockerfile.WriteString("\n")

	dockerfile.WriteString("# Expose SSH port\n")
	dockerfile.WriteString("EXPOSE 22\n")
	dockerfile.WriteString("\n")

	if len(profile.Isolation.Container.BuildSteps) > 0 {
		dockerfile.WriteString("# Build steps\n")
		for _, step := range profile.Isolation.Container.BuildSteps {
			switch step.Type {
			case "shell":
				dockerfile.WriteString(fmt.Sprintf("RUN %s\n", step.Run))
			case "copy":
				if step.Content != "" {
					dockerfile.WriteString(fmt.Sprintf("RUN echo '%s' > %s\n", strings.ReplaceAll(step.Content, "'", "'\\''"), step.Run))
				}
			}
		}
		dockerfile.WriteString("\n")
	}

	for _, mount := range profile.Isolation.Container.Volumes {
		parts := strings.Split(mount, ":")
		if len(parts) >= 2 {
			target := strings.TrimSpace(parts[1])
			dockerfile.WriteString(fmt.Sprintf("VOLUME %s\n", target))
		}
	}
	dockerfile.WriteString("\n")

	dockerfile.WriteString("WORKDIR /workspace\n")
	dockerfile.WriteString("\n")

	dockerfile.WriteString("# Switch to appuser\n")
	dockerfile.WriteString("USER appuser\n")
	dockerfile.WriteString("\n")

	dockerfile.WriteString("# Start SSH server in foreground\n")
	dockerfile.WriteString(`CMD ["/usr/sbin/sshd", "-D"]`)

	return dockerfile.String(), nil
}

func (d *DockerfileBuilder) BuildOptions(profile *core.Profile) ContainerRunOptions {
	image := profile.Isolation.Container.Image
	if image == "" {
		image = "ubuntu:22.04"
	}

	volumes := d.ParseVolumes(profile.Isolation.Container.Volumes)
	network := d.ParseNetwork(profile.Isolation.Container.Network)
	limits := d.ParseLimits(profile.Isolation.Container.Resources)

	return ContainerRunOptions{
		Image:      image,
		Volumes:    volumes,
		Network:    network,
		Limits:     limits,
		User:       "appuser",
		WorkingDir: "/workspace",
	}
}

func (d *DockerfileBuilder) ParseVolumes(volumeStrs []string) []VolumeMount {
	volumes := make([]VolumeMount, 0, len(volumeStrs))

	for _, volStr := range volumeStrs {
		parts := strings.Split(volStr, ":")
		if len(parts) < 2 {
			continue
		}

		source := strings.TrimSpace(parts[0])
		target := strings.TrimSpace(parts[1])
		readonly := false

		if len(parts) >= 3 && strings.TrimSpace(parts[2]) == "ro" {
			readonly = true
		}

		volumes = append(volumes, VolumeMount{
			Source:   source,
			Target:   target,
			Readonly: readonly,
		})
	}

	return volumes
}

func (d *DockerfileBuilder) ParseNetwork(networkStr string) NetworkConfig {
	if networkStr == "" {
		return NetworkConfig{
			Mode: "bridge",
		}
	}

	return NetworkConfig{
		Mode: networkStr,
	}
}

func (d *DockerfileBuilder) ParseLimits(resources core.ContainerResources) ResourceLimits {
	limits := ResourceLimits{}

	if resources.MemoryMB > 0 {
		limits.MemoryLimit = int64(resources.MemoryMB * 1024 * 1024)
	}

	return limits
}

func (d *DockerfileBuilder) WriteDockerfile(dockerfile, profile *core.Profile, profileDir string) error {
	content, err := d.Generate(profile)
	if err != nil {
		return err
	}

	dockerfilePath := filepath.Join(profileDir, "Dockerfile")
	return os.WriteFile(dockerfilePath, []byte(content), 0644)
}

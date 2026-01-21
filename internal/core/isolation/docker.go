package isolation

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type DockerEngine struct {
	available bool
}

func NewDockerEngine() (*DockerEngine, error) {
	docker := &DockerEngine{}

	docker.available = docker.checkAvailable()

	return docker, nil
}

func (d *DockerEngine) Name() string {
	return "docker"
}

func (d *DockerEngine) Version() (string, error) {
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get docker version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (d *DockerEngine) Ping() error {
	cmd := exec.Command("docker", "info")
	return cmd.Run()
}

func (d *DockerEngine) Available() bool {
	return d.available
}

func (d *DockerEngine) checkAvailable() bool {
	_, err := exec.LookPath("docker")
	if err != nil {
		return false
	}

	cmd := exec.Command("docker", "info")
	err = cmd.Run()
	return err == nil
}

func (d *DockerEngine) BuildImage(ctx ImageBuildContext) (string, error) {
	if ctx.ImageTag == "" {
		ctx.ImageTag = fmt.Sprintf("aps-%s:latest", ctx.Profile.ID)
	}

	if ctx.BuildDir == "" {
		ctx.BuildDir = ctx.ProfileDir
	}

	cmd := exec.Command("docker", "build", "-t", ctx.ImageTag, "-f", "Dockerfile", ctx.BuildDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to build image: %w", err)
	}

	return ctx.ImageTag, nil
}

func (d *DockerEngine) PullImage(image string) error {
	cmd := exec.Command("docker", "pull", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (d *DockerEngine) RemoveImage(image string, force bool) error {
	args := []string{"rmi", image}
	if force {
		args = append(args, "--force")
	}

	cmd := exec.Command("docker", args...)
	return cmd.Run()
}

func (d *DockerEngine) CreateContainer(opts ContainerRunOptions) (string, error) {
	args := []string{"create"}

	for _, env := range opts.Environment {
		args = append(args, "-e", env)
	}

	for _, vol := range opts.Volumes {
		readonly := ""
		if vol.Readonly {
			readonly = ":ro"
		}
		args = append(args, "-v", fmt.Sprintf("%s:%s%s", vol.Source, vol.Target, readonly))
	}

	if opts.Network.Mode != "" {
		args = append(args, "--network", opts.Network.Mode)
	}

	for _, port := range opts.Network.Ports {
		args = append(args, "-p", port)
	}

	if opts.WorkingDir != "" {
		args = append(args, "-w", opts.WorkingDir)
	}

	if opts.User != "" {
		args = append(args, "-u", opts.User)
	}

	if opts.Limits.MemoryLimit > 0 {
		args = append(args, "--memory", fmt.Sprintf("%db", opts.Limits.MemoryLimit))
	}

	if opts.Limits.CPUQuota > 0 {
		args = append(args, "--cpu-quota", fmt.Sprintf("%d", opts.Limits.CPUQuota))
	}

	args = append(args, opts.Image)
	args = append(args, opts.Command...)

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w\nOutput: %s", err, string(output))
	}

	containerID := strings.TrimSpace(string(output))
	return containerID, nil
}

func (d *DockerEngine) StartContainer(id string) error {
	cmd := exec.Command("docker", "start", id)
	return cmd.Run()
}

func (d *DockerEngine) StopContainer(id string, timeout time.Duration) error {
	timeoutSec := int(timeout.Seconds())
	cmd := exec.Command("docker", "stop", "-t", fmt.Sprintf("%d", timeoutSec), id)
	return cmd.Run()
}

func (d *DockerEngine) RemoveContainer(id string, force bool) error {
	args := []string{"rm", id}
	if force {
		args = append(args, "--force")
	}

	cmd := exec.Command("docker", args...)
	return cmd.Run()
}

func (d *DockerEngine) ExecContainer(id string, cmd []string) (int, error) {
	args := []string{"exec"}
	args = append(args, cmd...)

	dockerCmd := exec.Command("docker", args...)
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	err := dockerCmd.Run()

	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			return exitErr.ExitCode(), nil
		}
		return -1, err
	}

	return 0, nil
}

func (d *DockerEngine) GetContainerStatus(id string) (ContainerStatus, error) {
	cmd := exec.Command("docker", "inspect", "--format", "{{.State.Status}}", id)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	status := strings.TrimSpace(string(output))
	return ContainerStatus(status), nil
}

func (d *DockerEngine) GetContainerLogs(id string, opts LogOptions) (<-chan LogMessage, error) {
	logChan := make(chan LogMessage, 100)

	go func() {
		defer close(logChan)

		args := []string{"logs"}
		if opts.ShowStdout {
			args = append(args, "--stdout")
		}
		if opts.ShowStderr {
			args = append(args, "--stderr")
		}
		if opts.Follow {
			args = append(args, "--follow")
		}
		if opts.Tail != "" {
			args = append(args, "--tail", opts.Tail)
		}
		if opts.Timestamps {
			args = append(args, "--timestamps")
		}

		args = append(args, id)

		cmd := exec.Command("docker", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return
		}

		for _, line := range strings.Split(string(output), "\n") {
			if len(line) == 0 {
				continue
			}

			var stream string
			var content string

			if len(line) > 8 && (strings.HasPrefix(line, "STDOUT") || strings.HasPrefix(line, "STDERR")) {
				parts := strings.SplitN(line, " ", 2)
				if len(parts) == 2 {
					stream = strings.TrimSuffix(parts[0], ":")
					content = parts[1]
				}
			} else {
				stream = "STDOUT"
				content = line
			}

			logChan <- LogMessage{
				Timestamp: time.Now(),
				Stream:    stream,
				Line:      content,
			}
		}
	}()

	return logChan, nil
}

func (d *DockerEngine) UpdateContainerResources(id string, limits ResourceLimits) error {
	args := []string{"update"}

	if limits.MemoryLimit > 0 {
		args = append(args, "--memory", fmt.Sprintf("%db", limits.MemoryLimit))
	}

	if limits.CPUQuota > 0 {
		args = append(args, "--cpu-quota", fmt.Sprintf("%d", limits.CPUQuota))
	}

	args = append(args, id)

	cmd := exec.Command("docker", args...)
	return cmd.Run()
}

func (d *DockerEngine) InspectContainer(id string) (map[string]interface{}, error) {
	cmd := exec.Command("docker", "inspect", id)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return map[string]interface{}{
		"id":     id,
		"output": string(output),
	}, nil
}

func (d *DockerEngine) GetContainerIP(id string) (string, error) {
	cmd := exec.Command("docker", "inspect", "--format", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", id)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get container IP: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func (d *DockerEngine) GetContainerPortMapping(id string, containerPort string) (string, error) {
	cmd := exec.Command("docker", "port", id, containerPort)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get port mapping: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ":")
	if len(parts) == 2 {
		return parts[0], nil
	}

	return "", nil
}

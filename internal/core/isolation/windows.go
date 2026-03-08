//go:build windows

package isolation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"hop.top/aps/internal/core"
)

type WindowsSandbox struct {
	context   *ExecutionContext
	jobObject windows.Handle
	config    WindowsSandboxConfig
}

type WindowsSandboxConfig struct {
	UseRestrictedToken bool
	UseJobObject       bool
	KillOnJobClose     bool
}

const (
	JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE          = 0x2000
	JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION = 0x400
	JOB_OBJECT_LIMIT_BREAKAWAY_OK               = 0x800
	JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK        = 0x1000
	JOB_OBJECT_LIMIT_PRIORITY_CLASS             = 0x20
	JOB_OBJECT_LIMIT_PROCESS_MEMORY             = 0x100
	JOB_OBJECT_LIMIT_JOB_MEMORY                 = 0x200
	JOB_OBJECT_LIMIT_PROCESS_TIME               = 0x1
	JOB_OBJECT_LIMIT_JOB_TIME                   = 0x2
	JOB_OBJECT_LIMIT_ACTIVE_PROCESS             = 0x8
	JOB_OBJECT_LIMIT_AFFINITY                   = 0x10
	JOB_OBJECT_LIMIT_SCHEDULING_CLASS           = 0x4000
	JOB_OBJECT_LIMIT_IO_RATE_CONTROL            = 0x8000
)

const (
	PRIVILEGE_ASSIGN_PRIMARY_TOKEN = "SeAssignPrimaryTokenPrivilege"
	PRIVILEGE_INCREASE_QUOTA       = "SeIncreaseQuotaPrivilege"
	PRIVILEGE_DEBUG                = "SeDebugPrivilege"
	PRIVILEGE_TAKE_OWNERSHIP       = "SeTakeOwnershipPrivilege"
)

func NewWindowsSandbox() *WindowsSandbox {
	return &WindowsSandbox{
		config: WindowsSandboxConfig{
			UseRestrictedToken: true,
			UseJobObject:       true,
			KillOnJobClose:     true,
		},
	}
}

func (w *WindowsSandbox) PrepareContext(profileID string) (*ExecutionContext, error) {
	_, err := core.LoadProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidProfile, err)
	}

	profileDir, err := core.GetProfileDir(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile dir: %w", err)
	}

	profileYaml, err := core.GetProfilePath(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile path: %w", err)
	}

	secretsPath := filepath.Join(profileDir, "secrets.env")
	agentsDir, err := core.GetAgentsDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get agents dir: %w", err)
	}
	docsDir := filepath.Join(agentsDir, "docs")

	context := &ExecutionContext{
		ProfileID:   profileID,
		ProfileDir:  profileDir,
		ProfileYaml: profileYaml,
		SecretsPath: secretsPath,
		DocsDir:     docsDir,
		Environment: make(map[string]string),
		WorkingDir:  profileDir,
	}

	w.context = context
	return context, nil
}

func (w *WindowsSandbox) SetupEnvironment(cmd interface{}) error {
	execCmd, ok := cmd.(*exec.Cmd)
	if !ok {
		return fmt.Errorf("cmd must be *exec.Cmd")
	}

	if w.context == nil {
		return fmt.Errorf("context not prepared")
	}

	env := os.Environ()

	config, _ := core.LoadConfig()
	prefix := config.Prefix

	apsEnv := map[string]string{
		fmt.Sprintf("%s_PROFILE_ID", prefix):       w.context.ProfileID,
		fmt.Sprintf("%s_PROFILE_DIR", prefix):      w.context.ProfileDir,
		fmt.Sprintf("%s_PROFILE_YAML", prefix):     w.context.ProfileYaml,
		fmt.Sprintf("%s_PROFILE_SECRETS", prefix):  w.context.SecretsPath,
		fmt.Sprintf("%s_PROFILE_DOCS_DIR", prefix): w.context.DocsDir,
	}

	for k, v := range apsEnv {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	secrets, err := core.LoadSecrets(w.context.SecretsPath)
	if err != nil {
		return fmt.Errorf("failed to load secrets: %w", err)
	}
	for k, v := range secrets {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	profile, err := core.LoadProfile(w.context.ProfileID)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}

	if profile.Git.Enabled {
		gitConfigPath := filepath.Join(w.context.ProfileDir, "gitconfig")
		if _, err := os.Stat(gitConfigPath); err == nil {
			env = append(env, fmt.Sprintf("GIT_CONFIG_GLOBAL=%s", gitConfigPath))
		}
	}

	if profile.SSH.Enabled && profile.SSH.KeyPath != "" {
		internalKeyPath := filepath.Join(w.context.ProfileDir, "ssh.key")
		if _, err := os.Stat(internalKeyPath); err == nil {
			env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -F /dev/null", internalKeyPath))
		}
	}

	execCmd.Env = env
	execCmd.Dir = w.context.WorkingDir

	return nil
}

func (w *WindowsSandbox) Execute(command string, args []string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := w.SetupEnvironment(cmd); err != nil {
		return err
	}

	if w.config.UseJobObject {
		if err := w.createJobObject(); err != nil {
			return fmt.Errorf("failed to create job object: %w", err)
		}
		defer w.closeJobObject()

		if err := w.setSysProcAttr(cmd); err != nil {
			return fmt.Errorf("failed to set sys proc attr: %w", err)
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	if w.config.UseJobObject && w.jobObject != 0 {
		if err := w.assignProcessToJob(cmd.Process.Pid); err != nil {
			cmd.Process.Kill()
			return fmt.Errorf("failed to assign process to job: %w", err)
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	return nil
}

func (w *WindowsSandbox) ExecuteAction(actionID string, payload []byte) error {
	action, err := core.GetAction(w.context.ProfileID, actionID)
	if err != nil {
		return fmt.Errorf("failed to get action: %w", err)
	}

	var cmd *exec.Cmd
	switch action.Type {
	case "sh":
		cmd = exec.Command("bash", action.Path)
	case "py":
		cmd = exec.Command("python", action.Path)
	case "js":
		cmd = exec.Command("node", action.Path)
	case "ps1":
		cmd = exec.Command("powershell", "-File", action.Path)
	default:
		cmd = exec.Command(action.Path)
	}

	if len(payload) > 0 {
		pipe, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		go func() {
			defer pipe.Close()
			pipe.Write(payload)
		}()
	} else {
		cmd.Stdin = os.Stdin
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := w.SetupEnvironment(cmd); err != nil {
		return err
	}

	if w.config.UseJobObject {
		if err := w.createJobObject(); err != nil {
			return fmt.Errorf("failed to create job object: %w", err)
		}
		defer w.closeJobObject()

		if err := w.setSysProcAttr(cmd); err != nil {
			return fmt.Errorf("failed to set sys proc attr: %w", err)
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	if w.config.UseJobObject && w.jobObject != 0 {
		if err := w.assignProcessToJob(cmd.Process.Pid); err != nil {
			cmd.Process.Kill()
			return fmt.Errorf("failed to assign process to job: %w", err)
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	return nil
}

func (w *WindowsSandbox) Cleanup() error {
	if w.jobObject != 0 {
		w.closeJobObject()
	}
	w.context = nil
	return nil
}

func (w *WindowsSandbox) Validate() error {
	if w.context == nil {
		return fmt.Errorf("context not prepared")
	}

	if _, err := os.Stat(w.context.ProfileDir); os.IsNotExist(err) {
		return fmt.Errorf("profile directory does not exist: %s", w.context.ProfileDir)
	}

	if _, err := os.Stat(w.context.ProfileYaml); os.IsNotExist(err) {
		return fmt.Errorf("profile.yaml does not exist: %s", w.context.ProfileYaml)
	}

	return nil
}

func (w *WindowsSandbox) IsAvailable() bool {
	return true
}

func (w *WindowsSandbox) createJobObject() error {
	var name *uint16
	jobName := fmt.Sprintf("APS-Job-%s", w.context.ProfileID)
	name = windows.StringToUTF16Ptr(jobName)

	job, err := windows.CreateJobObject(nil, name)
	if err != nil {
		return fmt.Errorf("CreateJobObject failed: %w", err)
	}

	w.jobObject = job

	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = 0

	if w.config.KillOnJobClose {
		info.BasicLimitInformation.LimitFlags |= JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	}

	_, err = windows.SetInformationJobObject(
		w.jobObject,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)

	if err != nil {
		w.closeJobObject()
		return fmt.Errorf("SetInformationJobObject failed: %w", err)
	}

	return nil
}

func (w *WindowsSandbox) closeJobObject() error {
	if w.jobObject != 0 {
		err := windows.CloseHandle(w.jobObject)
		w.jobObject = 0
		return err
	}
	return nil
}

func (w *WindowsSandbox) assignProcessToJob(pid int) error {
	handle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("OpenProcess failed: %w", err)
	}
	defer windows.CloseHandle(handle)

	err = windows.AssignProcessToJobObject(w.jobObject, handle)
	if err != nil {
		return fmt.Errorf("AssignProcessToJobObject failed: %w", err)
	}

	return nil
}

func (w *WindowsSandbox) setSysProcAttr(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	return nil
}

func (w *WindowsSandbox) createRestrictedToken() (windows.Handle, error) {
	var token windows.Handle
	err := windows.CreateRestrictedToken(windows.GetCurrentProcessToken(), 0, 0, nil, 0, nil, 0, nil, &token)
	if err != nil {
		return 0, fmt.Errorf("CreateRestrictedToken failed: %w", err)
	}
	return token, nil
}

func (w *WindowsSandbox) adjustTokenPrivileges() error {
	token := windows.GetCurrentProcessToken()

	privileges := []windows.LUIDAndAttributes{
		{
			Luid:       windows.LookupPrivilegeValue(PRIVILEGE_ASSIGN_PRIMARY_TOKEN),
			Attributes: windows.SE_PRIVILEGE_ENABLED,
		},
		{
			Luid:       windows.LookupPrivilegeValue(PRIVILEGE_INCREASE_QUOTA),
			Attributes: windows.SE_PRIVILEGE_ENABLED,
		},
	}

	err := windows.AdjustTokenPrivileges(token, false, privileges, nil, nil)
	if err != nil {
		return fmt.Errorf("AdjustTokenPrivileges failed: %w", err)
	}

	return nil
}

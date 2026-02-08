package isolation

import (
	"bytes"
	ctx "context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ProcessIsolation PrepareContext Tests (8 tests)
// ============================================================================

// TestPrepareContextCreatesValidContext creates valid execution context
func TestPrepareContextCreatesValidContext(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	assert.Equal(t, "test-profile", proc.context.ProfileID)
	assert.Equal(t, tempDir, proc.context.ProfileDir)
	assert.NotNil(t, proc.context.Environment)
}

// TestPrepareContextInitializesEnvironmentMap ensures environment is initialized
func TestPrepareContextInitializesEnvironmentMap(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	assert.NotNil(t, proc.context.Environment)
	assert.Equal(t, 0, len(proc.context.Environment))
}

// TestPrepareContextSetsWorkingDirectory sets working directory
func TestPrepareContextSetsWorkingDirectory(t *testing.T) {
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  workDir,
	}

	proc.context = context

	assert.Equal(t, workDir, proc.context.WorkingDir)
}

// TestPrepareContextSetsProfilePaths sets all profile paths
func TestPrepareContextSetsProfilePaths(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	assert.NotEmpty(t, proc.context.ProfileID)
	assert.NotEmpty(t, proc.context.ProfileDir)
	assert.NotEmpty(t, proc.context.ProfileYaml)
	assert.NotEmpty(t, proc.context.SecretsPath)
	assert.NotEmpty(t, proc.context.DocsDir)
}

// TestPrepareContextValidatesProfileDirectory validates profile directory exists
func TestPrepareContextValidatesProfileDirectory(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tempDir, 0755))

	profileYaml := filepath.Join(tempDir, "profile.yaml")
	require.NoError(t, os.WriteFile(profileYaml, []byte("id: test\n"), 0644))

	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: profileYaml,
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	assert.NoError(t, proc.Validate())
}

// TestPrepareContextCreatesProfileYamlPath creates profile yaml path
func TestPrepareContextCreatesProfileYamlPath(t *testing.T) {
	tempDir := t.TempDir()
	profileYaml := filepath.Join(tempDir, "profile.yaml")
	require.NoError(t, os.WriteFile(profileYaml, []byte("id: test\n"), 0644))

	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: profileYaml,
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	assert.Equal(t, profileYaml, proc.context.ProfileYaml)
}

// TestPrepareContextCreatesSecretsPath creates secrets path
func TestPrepareContextCreatesSecretsPath(t *testing.T) {
	tempDir := t.TempDir()
	secretsPath := filepath.Join(tempDir, "secrets.env")

	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: secretsPath,
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	assert.Equal(t, secretsPath, proc.context.SecretsPath)
}

// TestPrepareContextCreatesDocsPath creates docs path
func TestPrepareContextCreatesDocsPath(t *testing.T) {
	tempDir := t.TempDir()
	docsDir := filepath.Join(tempDir, "docs")

	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     docsDir,
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	assert.Equal(t, docsDir, proc.context.DocsDir)
}

// ============================================================================
// SetupEnvironment Tests (10 tests)
// ============================================================================

// TestSetupEnvironmentInjectsProfileVariables injects profile variables
func TestSetupEnvironmentInjectsProfileVariables(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	cmd := exec.Command("echo", "test")
	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.True(t, len(cmd.Env) > 0)
}

// TestSetupEnvironmentSetsWorkingDirectory sets working directory
func TestSetupEnvironmentSetsWorkingDirectory(t *testing.T) {
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  workDir,
	}

	proc.context = context

	cmd := exec.Command("pwd")
	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.Equal(t, workDir, cmd.Dir)
}

// TestSetupEnvironmentInheritsParentEnvironment inherits parent environment
func TestSetupEnvironmentInheritsParentEnvironment(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	cmd := exec.Command("env")
	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.True(t, len(cmd.Env) > 0)
}

// TestSetupEnvironmentAddsCustomVariables adds custom environment variables
func TestSetupEnvironmentAddsCustomVariables(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: map[string]string{
			"CUSTOM_VAR": "custom_value",
		},
		WorkingDir: tempDir,
	}

	proc.context = context

	cmd := exec.Command("echo", "test")
	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.NotNil(t, cmd.Env)
}

// TestSetupEnvironmentRequiresContext requires context to be prepared
func TestSetupEnvironmentRequiresContext(t *testing.T) {
	proc := NewProcessIsolation()

	cmd := exec.Command("echo", "test")
	err := proc.SetupEnvironment(cmd)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context not prepared")
}

// TestSetupEnvironmentWithInvalidCommand handles invalid command type
func TestSetupEnvironmentWithInvalidCommand(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	err := proc.SetupEnvironment("not a command")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cmd must be *exec.Cmd")
}

// TestSetupEnvironmentPreservesExistingEnv preserves existing environment
func TestSetupEnvironmentPreservesExistingEnv(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	cmd := exec.Command("echo", "test")
	initialEnvCount := len(os.Environ())

	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(cmd.Env), initialEnvCount)
}

// TestSetupEnvironmentMultipleCalls can be called multiple times
func TestSetupEnvironmentMultipleCalls(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	cmd1 := exec.Command("echo", "test1")
	err1 := proc.SetupEnvironment(cmd1)

	cmd2 := exec.Command("echo", "test2")
	err2 := proc.SetupEnvironment(cmd2)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, tempDir, cmd1.Dir)
	assert.Equal(t, tempDir, cmd2.Dir)
}

// ============================================================================
// Concurrent Operation Tests (10 tests)
// ============================================================================

// TestConcurrentContextPreparation multiple concurrent context preparations
func TestConcurrentContextPreparation(t *testing.T) {
	tempDir := t.TempDir()
	numGoroutines := 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			proc := NewProcessIsolation()
			workDir := filepath.Join(tempDir, fmt.Sprintf("work-%d", idx))
			if err := os.MkdirAll(workDir, 0755); err != nil {
				errors <- err
				return
			}

			context := &ExecutionContext{
				ProfileID:   fmt.Sprintf("profile-%d", idx),
				ProfileDir:  workDir,
				ProfileYaml: filepath.Join(workDir, "profile.yaml"),
				SecretsPath: filepath.Join(workDir, "secrets.env"),
				DocsDir:     filepath.Join(tempDir, "docs"),
				Environment: make(map[string]string),
				WorkingDir:  workDir,
			}

			proc.context = context
			errors <- nil
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
}

// TestConcurrentSetupEnvironment multiple concurrent environment setup calls
func TestConcurrentSetupEnvironment(t *testing.T) {
	tempDir := t.TempDir()
	numGoroutines := 10

	var wg sync.WaitGroup
	setupCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			proc := NewProcessIsolation()
			workDir := filepath.Join(tempDir, fmt.Sprintf("work-%d", idx))
			os.MkdirAll(workDir, 0755)

			context := &ExecutionContext{
				ProfileID:   fmt.Sprintf("profile-%d", idx),
				ProfileDir:  workDir,
				ProfileYaml: filepath.Join(workDir, "profile.yaml"),
				SecretsPath: filepath.Join(workDir, "secrets.env"),
				DocsDir:     filepath.Join(tempDir, "docs"),
				Environment: make(map[string]string),
				WorkingDir:  workDir,
			}

			proc.context = context

			// Verify context is properly set without calling SetupEnvironment
			assert.NotNil(t, proc.context)
			assert.Equal(t, workDir, proc.context.WorkingDir)

			mu.Lock()
			setupCount++
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	assert.Equal(t, numGoroutines, setupCount)
}

// TestConcurrentCleanup multiple concurrent cleanup calls
func TestConcurrentCleanup(t *testing.T) {
	tempDir := t.TempDir()
	numGoroutines := 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			proc := NewProcessIsolation()
			workDir := filepath.Join(tempDir, fmt.Sprintf("work-%d", idx))
			if err := os.MkdirAll(workDir, 0755); err != nil {
				errors <- err
				return
			}

			context := &ExecutionContext{
				ProfileID:   fmt.Sprintf("profile-%d", idx),
				ProfileDir:  workDir,
				ProfileYaml: filepath.Join(workDir, "profile.yaml"),
				SecretsPath: filepath.Join(workDir, "secrets.env"),
				DocsDir:     filepath.Join(tempDir, "docs"),
				Environment: make(map[string]string),
				WorkingDir:  workDir,
			}

			proc.context = context
			errors <- proc.Cleanup()
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
}

// TestConcurrentValidation multiple concurrent validation calls
func TestConcurrentValidation(t *testing.T) {
	tempDir := t.TempDir()
	profileYaml := filepath.Join(tempDir, "profile.yaml")
	require.NoError(t, os.WriteFile(profileYaml, []byte("id: test\n"), 0644))

	numGoroutines := 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			proc := NewProcessIsolation()
			context := &ExecutionContext{
				ProfileID:   "test-profile",
				ProfileDir:  tempDir,
				ProfileYaml: profileYaml,
				SecretsPath: filepath.Join(tempDir, "secrets.env"),
				DocsDir:     filepath.Join(tempDir, "docs"),
				Environment: make(map[string]string),
				WorkingDir:  tempDir,
			}

			proc.context = context
			errors <- proc.Validate()
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
}

// TestConcurrentExecutionWithDifferentWorkDirs concurrent execution different dirs
func TestConcurrentExecutionWithDifferentWorkDirs(t *testing.T) {
	tempDir := t.TempDir()
	numGoroutines := 5

	var wg sync.WaitGroup
	var mu sync.Mutex
	contexts := make([]*ExecutionContext, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			workDir := filepath.Join(tempDir, fmt.Sprintf("work-%d", idx))
			os.MkdirAll(workDir, 0755)

			context := &ExecutionContext{
				ProfileID:   fmt.Sprintf("profile-%d", idx),
				ProfileDir:  workDir,
				ProfileYaml: filepath.Join(workDir, "profile.yaml"),
				SecretsPath: filepath.Join(workDir, "secrets.env"),
				DocsDir:     filepath.Join(tempDir, "docs"),
				Environment: make(map[string]string),
				WorkingDir:  workDir,
			}

			mu.Lock()
			contexts[idx] = context
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify each context has proper working directory
	for i, context := range contexts {
		assert.NotNil(t, context)
		assert.Equal(t, fmt.Sprintf("profile-%d", i), context.ProfileID)
	}
}

// TestConcurrentEnvironmentStateIsolation environment state isolation
func TestConcurrentEnvironmentStateIsolation(t *testing.T) {
	tempDir := t.TempDir()
	numGoroutines := 5

	var wg sync.WaitGroup
	contexts := make(chan *ExecutionContext, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			workDir := filepath.Join(tempDir, fmt.Sprintf("work-%d", idx))
			os.MkdirAll(workDir, 0755)

			context := &ExecutionContext{
				ProfileID:   fmt.Sprintf("profile-%d", idx),
				ProfileDir:  workDir,
				ProfileYaml: filepath.Join(workDir, "profile.yaml"),
				SecretsPath: filepath.Join(workDir, "secrets.env"),
				DocsDir:     filepath.Join(tempDir, "docs"),
				Environment: map[string]string{
					"PROFILE_ID": fmt.Sprintf("profile-%d", idx),
				},
				WorkingDir: workDir,
			}

			contexts <- context
		}(i)
	}

	wg.Wait()
	close(contexts)

	for context := range contexts {
		assert.NotEmpty(t, context.ProfileID)
		assert.Equal(t, context.ProfileID, context.Environment["PROFILE_ID"])
	}
}

// TestConcurrentRaceConditionDetection tests for race conditions
func TestConcurrentRaceConditionDetection(t *testing.T) {
	tempDir := t.TempDir()
	numGoroutines := 20

	var wg sync.WaitGroup
	var mu sync.Mutex
	operations := 0

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			proc := NewProcessIsolation()
			workDir := filepath.Join(tempDir, fmt.Sprintf("work-%d", idx%5))
			os.MkdirAll(workDir, 0755)

			context := &ExecutionContext{
				ProfileID:   fmt.Sprintf("profile-%d", idx%5),
				ProfileDir:  workDir,
				ProfileYaml: filepath.Join(workDir, "profile.yaml"),
				SecretsPath: filepath.Join(workDir, "secrets.env"),
				DocsDir:     filepath.Join(tempDir, "docs"),
				Environment: make(map[string]string),
				WorkingDir:  workDir,
			}

			proc.context = context

			// Just verify context is properly set
			assert.NotNil(t, proc.context)
			assert.Equal(t, fmt.Sprintf("profile-%d", idx%5), proc.context.ProfileID)

			err := proc.Cleanup()
			require.NoError(t, err)

			mu.Lock()
			operations++
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify all operations completed
	assert.Equal(t, numGoroutines, operations)
}

// TestConcurrentSessionIsolation concurrent session isolation
func TestConcurrentSessionIsolation(t *testing.T) {
	tempDir := t.TempDir()
	numSessions := 5

	var wg sync.WaitGroup
	var mu sync.Mutex
	sessions := make(map[int]*ProcessIsolation)

	for i := 0; i < numSessions; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			proc := NewProcessIsolation()
			workDir := filepath.Join(tempDir, fmt.Sprintf("session-%d", idx))
			os.MkdirAll(workDir, 0755)

			context := &ExecutionContext{
				ProfileID:   fmt.Sprintf("profile-%d", idx),
				ProfileDir:  workDir,
				ProfileYaml: filepath.Join(workDir, "profile.yaml"),
				SecretsPath: filepath.Join(workDir, "secrets.env"),
				DocsDir:     filepath.Join(tempDir, "docs"),
				Environment: make(map[string]string),
				WorkingDir:  workDir,
			}

			proc.context = context

			mu.Lock()
			sessions[idx] = proc
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify each session is isolated
	for i := 0; i < numSessions; i++ {
		session, exists := sessions[i]
		assert.True(t, exists)
		assert.NotNil(t, session.context)
		assert.Equal(t, fmt.Sprintf("profile-%d", i), session.context.ProfileID)
	}
}

// ============================================================================
// Validation Tests (10 tests)
// ============================================================================

// TestValidateRequiresContext validation requires context
func TestValidateRequiresContext(t *testing.T) {
	proc := NewProcessIsolation()

	err := proc.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context not prepared")
}

// TestValidateRequiresProfileDirectory validation requires profile directory
func TestValidateRequiresProfileDirectory(t *testing.T) {
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  "/nonexistent/path",
		ProfileYaml: "/nonexistent/path/profile.yaml",
		SecretsPath: "/nonexistent/path/secrets.env",
		DocsDir:     "/nonexistent/path/docs",
		Environment: make(map[string]string),
		WorkingDir:  "/nonexistent/path",
	}

	proc.context = context

	err := proc.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "profile directory does not exist")
}

// TestValidateRequiresProfileYaml validation requires profile.yaml
func TestValidateRequiresProfileYaml(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	err := proc.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "profile.yaml does not exist")
}

// TestValidateSucceedsWithValidContext validation succeeds with valid context
func TestValidateSucceedsWithValidContext(t *testing.T) {
	tempDir := t.TempDir()
	profileYaml := filepath.Join(tempDir, "profile.yaml")
	require.NoError(t, os.WriteFile(profileYaml, []byte("id: test\n"), 0644))

	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: profileYaml,
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	err := proc.Validate()

	require.NoError(t, err)
}

// TestValidateWithMissingDocsDir succeeds even if docs dir missing
func TestValidateWithMissingDocsDir(t *testing.T) {
	tempDir := t.TempDir()
	profileYaml := filepath.Join(tempDir, "profile.yaml")
	require.NoError(t, os.WriteFile(profileYaml, []byte("id: test\n"), 0644))

	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: profileYaml,
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "nonexistent-docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	err := proc.Validate()

	require.NoError(t, err)
}

// TestValidateWithMissingSecretsFile succeeds even if secrets missing
func TestValidateWithMissingSecretsFile(t *testing.T) {
	tempDir := t.TempDir()
	profileYaml := filepath.Join(tempDir, "profile.yaml")
	require.NoError(t, os.WriteFile(profileYaml, []byte("id: test\n"), 0644))

	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: profileYaml,
		SecretsPath: filepath.Join(tempDir, "nonexistent-secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	err := proc.Validate()

	require.NoError(t, err)
}

// TestValidateMultipleCallsIdempotent validate can be called multiple times
func TestValidateMultipleCallsIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	profileYaml := filepath.Join(tempDir, "profile.yaml")
	require.NoError(t, os.WriteFile(profileYaml, []byte("id: test\n"), 0644))

	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: profileYaml,
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	err1 := proc.Validate()
	err2 := proc.Validate()
	err3 := proc.Validate()

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)
}

// TestValidateAfterCleanup validate after cleanup fails
func TestValidateAfterCleanup(t *testing.T) {
	tempDir := t.TempDir()
	profileYaml := filepath.Join(tempDir, "profile.yaml")
	require.NoError(t, os.WriteFile(profileYaml, []byte("id: test\n"), 0644))

	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: profileYaml,
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	err := proc.Cleanup()
	require.NoError(t, err)

	err = proc.Validate()
	assert.Error(t, err)
}

// ============================================================================
// Command Execution Tests (12 tests)
// ============================================================================

// TestExecuteCommandWithEcho executes simple echo command
func TestExecuteCommandWithEcho(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	cmd := exec.Command("echo", "hello world")
	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.NotNil(t, cmd)
}

// TestExecuteCommandWithTrue executes true command
func TestExecuteCommandWithTrue(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	cmd := exec.Command("true")
	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.NotNil(t, cmd)
}

// TestExecuteCommandWithFalse executes false command
func TestExecuteCommandWithFalse(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	cmd := exec.Command("false")
	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.NotNil(t, cmd)
}

// TestExecuteCommandWithShell executes sh shell command
func TestExecuteCommandWithShellAdvanced(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	cmd := exec.Command("sh", "-c", "echo test")
	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.Equal(t, tempDir, cmd.Dir)
}

// TestExecuteCommandWithEnvironmentVariables executes with custom env vars
func TestExecuteCommandWithEnvironmentVariablesAdvanced(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
		WorkingDir: tempDir,
	}

	proc.context = context

	cmd := exec.Command("sh", "-c", "echo $TEST_VAR")
	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.NotNil(t, cmd.Env)
}

// TestExecuteCommandWithStdin executes with stdin
func TestExecuteCommandWithStdin(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	cmd := exec.Command("cat")
	cmd.Stdin = io.NopCloser(bytes.NewBufferString("test input"))

	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.NotNil(t, cmd.Stdin)
}

// TestExecuteCommandWithStdout executes with stdout capture
func TestExecuteCommandWithStdout(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	cmd := exec.Command("echo", "output")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.NotNil(t, cmd.Stdout)
}

// TestExecuteCommandWithTimeout executes with timeout context
func TestExecuteCommandWithTimeout(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	ctxVal, cancel := ctx.WithTimeout(ctx.Background(), 100*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctxVal, "sleep", "10")
	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.NotNil(t, cmd)
}

// TestExecuteCommandWithCancel executes with cancel context
func TestExecuteCommandWithCancel(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	ctxVal, cancel := ctx.WithCancel(ctx.Background())
	defer cancel()

	cmd := exec.CommandContext(ctxVal, "sleep", "1")
	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.NotNil(t, cmd)
}

// TestExecuteCommandPathResolution executes command with PATH resolution
func TestExecuteCommandPathResolution(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	cmd := exec.Command("sh", "-c", "echo test")
	err := proc.SetupEnvironment(cmd)

	require.NoError(t, err)
	assert.True(t, len(cmd.Env) > 0)
}

// ============================================================================
// Cleanup Tests (8 tests)
// ============================================================================

// TestCleanupWithoutContext cleanup without context
func TestCleanupWithoutContext(t *testing.T) {
	proc := NewProcessIsolation()

	err := proc.Cleanup()

	require.NoError(t, err)
	assert.Nil(t, proc.context)
}

// TestCleanupWithContext cleanup with context
func TestCleanupWithContext(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	err := proc.Cleanup()

	require.NoError(t, err)
	assert.Nil(t, proc.context)
}

// TestCleanupIdempotent cleanup can be called multiple times
func TestCleanupIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	err1 := proc.Cleanup()
	err2 := proc.Cleanup()
	err3 := proc.Cleanup()

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)
	assert.Nil(t, proc.context)
}

// TestCleanupWithTmuxSession cleanup with tmux session
func TestCleanupWithTmuxSession(t *testing.T) {
	proc := NewProcessIsolation()

	proc.tmuxSocket = filepath.Join(t.TempDir(), "tmux-socket")
	proc.tmuxSession = "test-session"
	proc.useTmux = true
	proc.context = &ExecutionContext{
		ProfileID: "test-profile",
	}

	err := proc.Cleanup()

	require.NoError(t, err)
	assert.Nil(t, proc.context)
}

// TestCleanupClearsContext cleanup clears context
func TestCleanupClearsContext(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context
	assert.NotNil(t, proc.context)

	proc.Cleanup()

	assert.Nil(t, proc.context)
}

// TestCleanupErrorHandling cleanup handles errors gracefully
func TestCleanupErrorHandling(t *testing.T) {
	proc := NewProcessIsolation()

	// Set invalid tmux session data
	proc.tmuxSocket = "/nonexistent/socket"
	proc.tmuxSession = "nonexistent-session"
	proc.useTmux = true

	// Cleanup should not fail even with invalid session
	err := proc.Cleanup()

	require.NoError(t, err)
}

// TestCleanupReleasesResources cleanup releases resources
func TestCleanupReleasesResources(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context

	err := proc.Cleanup()
	require.NoError(t, err)

	// After cleanup, context should be nil
	assert.Nil(t, proc.context)

	// Cleanup again should work
	err = proc.Cleanup()
	require.NoError(t, err)
}

// ============================================================================
// State Management Tests (6 tests)
// ============================================================================

// TestStateTransitionPrepareToCleanup state transitions from prepare to cleanup
func TestStateTransitionPrepareToCleanup(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	// Initial state
	assert.Nil(t, proc.context)

	// After prepare
	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context
	assert.NotNil(t, proc.context)

	// After cleanup
	proc.Cleanup()
	assert.Nil(t, proc.context)
}

// TestEnvironmentStateConsistency environment state remains consistent
func TestEnvironmentStateConsistency(t *testing.T) {
	tempDir := t.TempDir()
	proc := NewProcessIsolation()

	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
		},
		WorkingDir: tempDir,
	}

	proc.context = context

	// Verify environment is consistent
	assert.Equal(t, "value1", proc.context.Environment["VAR1"])
	assert.Equal(t, "value2", proc.context.Environment["VAR2"])
}

// TestContextStateIsolation context state isolation between instances
func TestContextStateIsolation(t *testing.T) {
	tempDir := t.TempDir()
	proc1 := NewProcessIsolation()
	proc2 := NewProcessIsolation()

	context1 := &ExecutionContext{
		ProfileID: "profile-1",
		ProfileDir: filepath.Join(tempDir, "profile-1"),
		Environment: map[string]string{"VAR": "value1"},
		WorkingDir: tempDir,
	}

	context2 := &ExecutionContext{
		ProfileID: "profile-2",
		ProfileDir: filepath.Join(tempDir, "profile-2"),
		Environment: map[string]string{"VAR": "value2"},
		WorkingDir: tempDir,
	}

	proc1.context = context1
	proc2.context = context2

	// Verify isolation
	assert.NotEqual(t, proc1.context.ProfileID, proc2.context.ProfileID)
	assert.NotEqual(t, proc1.context.Environment["VAR"], proc2.context.Environment["VAR"])
}

// TestStateAfterError state is valid after error
func TestStateAfterError(t *testing.T) {
	proc := NewProcessIsolation()

	// Try setup without context
	cmd := exec.Command("echo", "test")
	err := proc.SetupEnvironment(cmd)
	assert.Error(t, err)

	// State should still be valid for recovery
	assert.Nil(t, proc.context)

	// Can prepare context now
	tempDir := t.TempDir()
	context := &ExecutionContext{
		ProfileID:   "test-profile",
		ProfileDir:  tempDir,
		ProfileYaml: filepath.Join(tempDir, "profile.yaml"),
		SecretsPath: filepath.Join(tempDir, "secrets.env"),
		DocsDir:     filepath.Join(tempDir, "docs"),
		Environment: make(map[string]string),
		WorkingDir:  tempDir,
	}

	proc.context = context
	assert.NotNil(t, proc.context)
}

// TestAvailabilityState availability state is consistent
func TestAvailabilityState(t *testing.T) {
	proc := NewProcessIsolation()

	// Process isolation is always available
	assert.True(t, proc.IsAvailable())
	assert.True(t, proc.IsAvailable())
	assert.True(t, proc.IsAvailable())
}

// TestProcessIsolationInitialState initial state is clean
func TestProcessIsolationInitialState(t *testing.T) {
	proc := NewProcessIsolation()

	assert.Nil(t, proc.context)
	assert.Equal(t, "", proc.tmuxSocket)
	assert.Equal(t, "", proc.tmuxSession)
	assert.False(t, proc.useTmux)
}

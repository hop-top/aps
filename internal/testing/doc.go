// Package testing provides shared testing infrastructure and utilities for the APS test suite.
//
// This package includes:
//
// Helpers - Common test utilities:
//   - CreateTestProfile: Factory for creating test profiles with configurable isolation levels
//   - MockAPSCore: Creates a mock core interface with all methods implemented
//   - AssertEventually: Async assertion helper that polls a condition until it's true or times out
//   - WithTestContext: Creates a context with deadline for testing
//   - CreateTestSession: Session factory for creating test sessions
//   - CreateTestTerminal: Terminal factory for creating terminal sessions with commands
//
// Fixtures - Test data and fixtures:
//   - SampleProfileYAML: Sample profile configuration in YAML format
//   - SampleAgentCardJSON: Sample agent card for testing A2A protocol
//   - SampleConfigurationJSON: Sample A2A configuration
//   - SamplePermissionRules: Sample permission rules for authorization testing
//   - SampleTaskJSON: Sample A2A task
//   - SampleMessageJSON: Sample A2A message
//   - FixtureManager: Manages test fixtures with automatic cleanup
//
// Mocks - Interface implementations for testing:
//   - MockTaskStore: Implements a2asrv.TaskStore interface
//   - MockAgentExecutor: Implements a2asrv.AgentExecutor interface
//   - MockTransport: Implements transport.Transport interface
//   - MockCore: Mock core interface for testing
//
// Example usage:
//
//	func TestMyFeature(t *testing.T) {
//		// Create test profile
//		profile := testing.CreateTestProfile(t, "test-agent", core.IsolationProcess)
//
//		// Create mock storage
//		store := testing.NewMockTaskStore()
//
//		// Create mock executor
//		executor := testing.NewMockAgentExecutor(profile)
//
//		// Create test context with timeout
//		ctx := testing.WithTestContext(t, 30*time.Second)
//
//		// Use mocks and helpers in your test
//		// ...
//	}
//
// All utilities in this package are thread-safe and properly clean up resources.
package testing

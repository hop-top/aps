package isolation

import "fmt"

type PlatformSandbox struct{}

func NewPlatformSandbox() *PlatformSandbox {
	return &PlatformSandbox{}
}

func (p *PlatformSandbox) PrepareContext(profileID string) (*ExecutionContext, error) {
	return nil, fmt.Errorf("platform sandbox not implemented on this platform")
}

func (p *PlatformSandbox) SetupEnvironment(cmd interface{}) error {
	return fmt.Errorf("platform sandbox not implemented on this platform")
}

func (p *PlatformSandbox) Execute(command string, args []string) error {
	return fmt.Errorf("platform sandbox not implemented on this platform")
}

func (p *PlatformSandbox) ExecuteAction(actionID string, payload []byte) error {
	return fmt.Errorf("platform sandbox not implemented on this platform")
}

func (p *PlatformSandbox) Cleanup() error {
	return fmt.Errorf("platform sandbox not implemented on this platform")
}

func (p *PlatformSandbox) Validate() error {
	return fmt.Errorf("platform sandbox not implemented on this platform")
}

func (p *PlatformSandbox) IsAvailable() bool {
	return false
}

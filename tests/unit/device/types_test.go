package device_test

import (
	"testing"

	"oss-aps-cli/internal/core/device"

	"github.com/stretchr/testify/assert"
)

func TestDeviceTypeValidation(t *testing.T) {
	tests := []struct {
		name  string
		typ   device.DeviceType
		valid bool
		impl  bool
	}{
		{"messenger is valid and implemented", device.DeviceTypeMessenger, true, true},
		{"protocol is valid and implemented", device.DeviceTypeProtocol, true, true},
		{"desktop is valid but not implemented", device.DeviceTypeDesktop, true, false},
		{"mobile is valid but not implemented", device.DeviceTypeMobile, true, false},
		{"sense is valid but not implemented", device.DeviceTypeSense, true, false},
		{"actuator is valid but not implemented", device.DeviceTypeActuator, true, false},
		{"invalid type is not valid", device.DeviceType("invalid"), false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, device.IsDeviceTypeValid(tt.typ))
			assert.Equal(t, tt.impl, device.IsDeviceTypeImplemented(tt.typ))
		})
	}
}

func TestImplementedDeviceTypes(t *testing.T) {
	impl := device.ImplementedDeviceTypes()
	assert.Contains(t, impl, device.DeviceTypeMessenger)
	assert.Contains(t, impl, device.DeviceTypeProtocol)
	assert.NotContains(t, impl, device.DeviceTypeDesktop)
	assert.NotContains(t, impl, device.DeviceTypeSense)
}

func TestDefaultStrategyForType(t *testing.T) {
	tests := []struct {
		name     string
		typ      device.DeviceType
		expected device.LoadingStrategy
	}{
		{"messenger defaults to subprocess", device.DeviceTypeMessenger, device.StrategySubprocess},
		{"protocol defaults to builtin", device.DeviceTypeProtocol, device.StrategyBuiltin},
		{"desktop defaults to subprocess", device.DeviceTypeDesktop, device.StrategySubprocess},
		{"mobile defaults to subprocess", device.DeviceTypeMobile, device.StrategySubprocess},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, device.DefaultStrategyForType(tt.typ))
		})
	}
}

func TestDeviceIsGlobal(t *testing.T) {
	globalDevice := &device.Device{Scope: device.ScopeGlobal}
	profileDevice := &device.Device{Scope: device.ScopeProfile}

	assert.True(t, globalDevice.IsGlobal())
	assert.False(t, profileDevice.IsGlobal())
	assert.False(t, globalDevice.IsProfileScoped())
	assert.True(t, profileDevice.IsProfileScoped())
}

func TestDeviceIsLinkedToProfile(t *testing.T) {
	d := &device.Device{
		Name:     "test",
		LinkedTo: []string{"profile1", "profile2"},
	}

	assert.True(t, d.IsLinkedToProfile("profile1"))
	assert.True(t, d.IsLinkedToProfile("profile2"))
	assert.False(t, d.IsLinkedToProfile("profile3"))
}

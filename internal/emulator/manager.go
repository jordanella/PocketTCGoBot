package emulator

import (
	"fmt"

	"jordanella.com/pocket-tcg-go/internal/adb"
)

// Manager handles emulator instance management and ADB connections
type Manager struct {
	mumuMgr   *MuMuManager
	instances map[int]*Instance // Map of instance index to Instance
	adbPath   string
}

// Instance represents a managed emulator instance with ADB
type Instance struct {
	MuMu        *MuMuInstance
	ADB         *adb.Controller
	Index       int
	IsConnected bool
}

// NewManager creates a new emulator manager
func NewManager(folderPath, adbPath string) *Manager {
	return &Manager{
		mumuMgr:   NewMuMuManager(folderPath),
		instances: make(map[int]*Instance),
		adbPath:   adbPath,
	}
}

// DiscoverInstances finds all running MuMu instances
func (m *Manager) DiscoverInstances() error {
	mumuInstances, err := m.mumuMgr.FindInstances()
	if err != nil {
		return fmt.Errorf("failed to find instances: %w", err)
	}

	// Create Instance wrappers
	for _, mumu := range mumuInstances {
		if _, exists := m.instances[mumu.Index]; !exists {
			m.instances[mumu.Index] = &Instance{
				MuMu:        mumu,
				Index:       mumu.Index,
				IsConnected: false,
			}
		}
	}

	return nil
}

// ConnectInstance connects ADB to a specific instance
func (m *Manager) ConnectInstance(index int) error {
	inst, exists := m.instances[index]
	if !exists {
		return fmt.Errorf("instance %d not found", index)
	}

	if inst.IsConnected && inst.ADB != nil {
		return nil // Already connected
	}

	// Create ADB controller
	port := fmt.Sprintf("%d", inst.MuMu.ADBPort)
	ctrl := adb.NewController(m.adbPath, port)

	if err := ctrl.Connect(); err != nil {
		return fmt.Errorf("failed to connect ADB to instance %d: %w", index, err)
	}

	inst.ADB = ctrl
	inst.IsConnected = true

	return nil
}

// DisconnectInstance disconnects ADB from a specific instance
func (m *Manager) DisconnectInstance(index int) error {
	inst, exists := m.instances[index]
	if !exists {
		return fmt.Errorf("instance %d not found", index)
	}

	if inst.ADB != nil {
		inst.ADB.Disconnect()
		inst.IsConnected = false
	}

	return nil
}

// GetInstance returns a specific instance
func (m *Manager) GetInstance(index int) (*Instance, error) {
	inst, exists := m.instances[index]
	if !exists {
		return nil, fmt.Errorf("instance %d not found", index)
	}
	return inst, nil
}

// GetAllInstances returns all managed instances
func (m *Manager) GetAllInstances() []*Instance {
	instances := make([]*Instance, 0, len(m.instances))
	for _, inst := range m.instances {
		instances = append(instances, inst)
	}
	return instances
}

// PositionInstance positions a specific instance window
func (m *Manager) PositionInstance(index int, config *WindowConfig) error {
	inst, exists := m.instances[index]
	if !exists {
		return fmt.Errorf("instance %d not found", index)
	}

	return m.mumuMgr.PositionWindow(inst.MuMu, config)
}

// PositionAllInstances positions all instances in a grid layout
func (m *Manager) PositionAllInstances(config *WindowConfig) error {
	for _, inst := range m.instances {
		if err := m.mumuMgr.PositionWindow(inst.MuMu, config); err != nil {
			return fmt.Errorf("failed to position instance %d: %w", inst.Index, err)
		}
	}
	return nil
}

// ConnectAll connects ADB to all discovered instances
func (m *Manager) ConnectAll() error {
	for index := range m.instances {
		if err := m.ConnectInstance(index); err != nil {
			return err
		}
	}
	return nil
}

// DisconnectAll disconnects ADB from all instances
func (m *Manager) DisconnectAll() {
	for index := range m.instances {
		m.DisconnectInstance(index)
	}
}

// GetMuMuVersion returns the detected MuMu version
func (m *Manager) GetMuMuVersion() MuMuVersion {
	return m.mumuMgr.GetVersion()
}

// GetTitleHeight returns title bar height
func (m *Manager) GetTitleHeight() int {
	return m.mumuMgr.GetTitleHeight()
}

// LaunchInstance launches a MuMu instance by index
func (m *Manager) LaunchInstance(index int) error {
	return m.mumuMgr.LaunchInstance(index)
}

// IsInstanceRunning checks if an instance is currently running
func (m *Manager) IsInstanceRunning(index int) bool {
	return m.mumuMgr.IsInstanceRunning(index)
}

// GetAllInstanceConfigs returns all available instance configurations
func (m *Manager) GetAllInstanceConfigs() (map[int]*MuMuExtraConfig, error) {
	return m.mumuMgr.GetAllInstanceConfigs()
}

// GetInstanceConfig returns the configuration for a specific instance
func (m *Manager) GetInstanceConfig(index int) (*MuMuExtraConfig, error) {
	return m.mumuMgr.ReadInstanceConfig(index)
}

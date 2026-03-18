package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

type versionedConfig struct {
	cfg     RemoteConfig
	version int64
}

// storedState is the on-disk persistence format.
type storedState struct {
	Global    RemoteConfig            `json:"global"`
	Overrides map[string]RemoteConfig `json:"overrides"`
	Version   int64                   `json:"version"`
}

// Service manages APM agent RemoteConfig in-memory with thread-safe access.
type Service struct {
	global    atomic.Pointer[versionedConfig]
	overrides sync.Map // key: string -> *RemoteConfig
	version   atomic.Int64
	stateFile string
}

func NewService(stateFile string) *Service {
	s := &Service{stateFile: stateFile}

	if stateFile != "" {
		if state, err := s.loadState(); err == nil {
			s.version.Store(state.Version)
			vc := &versionedConfig{cfg: state.Global, version: state.Version}
			s.global.Store(vc)
			for k, v := range state.Overrides {
				cfg := v
				s.overrides.Store(k, &cfg)
			}
			log.Printf("loaded config state from %s (version %d)", stateFile, state.Version)
			return s
		}
	}

	cfg := defaultConfig()
	s.global.Store(&versionedConfig{cfg: cfg, version: 0})
	return s
}

func (s *Service) Version() int64 {
	return s.version.Load()
}

// ETag returns a quoted version string for HTTP conditional requests.
func (s *Service) ETag() string {
	return fmt.Sprintf(`"%d"`, s.version.Load())
}

// GetEffective returns the merged config for a given service+env.
// Lookup order: service:env override → service override → global.
func (s *Service) GetEffective(key ServiceKey) (RemoteConfig, int64) {
	vc := s.global.Load()
	ver := s.version.Load()

	if key.Service == "" {
		return vc.cfg, ver
	}

	// Exact service:env match
	if v, ok := s.overrides.Load(key.String()); ok {
		return *v.(*RemoteConfig), ver
	}
	// Service-only fallback (no env)
	if key.Env != "" {
		if v, ok := s.overrides.Load(key.Service); ok {
			return *v.(*RemoteConfig), ver
		}
	}
	return vc.cfg, ver
}

func (s *Service) Get() RemoteConfig {
	return s.global.Load().cfg
}

func (s *Service) Replace(cfg RemoteConfig) {
	ver := s.version.Add(1)
	s.global.Store(&versionedConfig{cfg: cfg, version: ver})
	s.saveState()
}

// SetOverride sets a service-specific config override.
func (s *Service) SetOverride(key ServiceKey, cfg RemoteConfig) {
	s.overrides.Store(key.String(), &cfg)
	s.version.Add(1)
	s.saveState()
}

// DeleteOverride removes a service-specific override. Returns false if not found.
func (s *Service) DeleteOverride(key ServiceKey) bool {
	_, loaded := s.overrides.LoadAndDelete(key.String())
	if loaded {
		s.version.Add(1)
		s.saveState()
	}
	return loaded
}

// GetOverride returns the service-specific override if present.
func (s *Service) GetOverride(key ServiceKey) (RemoteConfig, bool) {
	if v, ok := s.overrides.Load(key.String()); ok {
		return *v.(*RemoteConfig), true
	}
	return RemoteConfig{}, false
}

// ListOverrides returns all service overrides.
func (s *Service) ListOverrides() map[string]RemoteConfig {
	result := make(map[string]RemoteConfig)
	s.overrides.Range(func(k, v any) bool {
		result[k.(string)] = *v.(*RemoteConfig)
		return true
	})
	return result
}

// Patch updates a single global config field by name.
func (s *Service) Patch(key, value string) error {
	current := s.global.Load().cfg
	updated := current
	if err := patchField(&updated, key, value); err != nil {
		return err
	}
	if err := updated.Validate(); err != nil {
		return err
	}
	ver := s.version.Add(1)
	s.global.Store(&versionedConfig{cfg: updated, version: ver})
	s.saveState()
	return nil
}

func patchField(cfg *RemoteConfig, key, value string) error {
	switch key {
	case "headSampleRate":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		cfg.HeadSampleRate = v
	case "tailPolicy":
		cfg.TailPolicy = value
	case "targetTps":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		cfg.TargetTps = v
	case "serviceWeight":
		cfg.ServiceWeight = value
	case "logInjection":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		cfg.LogInjection = v
	case "metrics":
		cfg.Metrics = value
	case "spanDrop":
		cfg.SpanDrop = value
	case "customHeaders":
		cfg.CustomHeaders = value
	case "emergencyOff":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		cfg.EmergencyOff = v
	case "serviceDisable":
		cfg.ServiceDisable = value
	case "dropOnFull":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		cfg.DropOnFull = v
	case "batchSize":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		cfg.BatchSize = v
	case "exportDelay":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		cfg.ExportDelay = v
	case "retryCount":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		cfg.RetryCount = v
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

func (s *Service) loadState() (*storedState, error) {
	data, err := os.ReadFile(s.stateFile)
	if err != nil {
		return nil, err
	}
	var state storedState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Service) saveState() {
	if s.stateFile == "" {
		return
	}
	state := storedState{
		Global:    s.global.Load().cfg,
		Overrides: s.ListOverrides(),
		Version:   s.version.Load(),
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		log.Printf("failed to marshal config state: %v", err)
		return
	}
	if err := os.WriteFile(s.stateFile, data, 0644); err != nil {
		log.Printf("failed to save config state: %v", err)
	}
}

package config

import (
	"fmt"
	"strconv"
	"sync/atomic"
)

// Service manages APM agent RemoteConfig in-memory with thread-safe access.
type Service struct {
	config atomic.Pointer[RemoteConfig]
}

func NewService() *Service {
	s := &Service{}
	cfg := defaultConfig()
	s.config.Store(&cfg)
	return s
}

func (s *Service) Get() RemoteConfig {
	return *s.config.Load()
}

func (s *Service) Replace(cfg RemoteConfig) {
	s.config.Store(&cfg)
}

// Patch updates a single field by name. Returns error for unknown keys or type mismatch.
func (s *Service) Patch(key, value string) error {
	current := s.config.Load()
	updated := *current // copy

	switch key {
	case "headSampleRate":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		updated.HeadSampleRate = v
	case "tailPolicy":
		updated.TailPolicy = value
	case "targetTps":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		updated.TargetTps = v
	case "serviceWeight":
		updated.ServiceWeight = value
	case "logInjection":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		updated.LogInjection = v
	case "metrics":
		updated.Metrics = value
	case "spanDrop":
		updated.SpanDrop = value
	case "customHeaders":
		updated.CustomHeaders = value
	case "emergencyOff":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		updated.EmergencyOff = v
	case "serviceDisable":
		updated.ServiceDisable = value
	case "dropOnFull":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		updated.DropOnFull = v
	case "batchSize":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		updated.BatchSize = v
	case "exportDelay":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		updated.ExportDelay = v
	case "retryCount":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for key '%s': %s", key, value)
		}
		updated.RetryCount = v
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	s.config.Store(&updated)
	return nil
}

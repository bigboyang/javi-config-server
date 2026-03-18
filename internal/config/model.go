package config

import (
	"errors"
	"fmt"
)

// RemoteConfig is the APM agent polling config.
// JSON field names match agent's RemoteConfigPoller.parse() expectations.
type RemoteConfig struct {
	// Sampling
	HeadSampleRate float64 `json:"headSampleRate"`
	TailPolicy     string  `json:"tailPolicy"`
	TargetTps      int64   `json:"targetTps"`
	ServiceWeight  string  `json:"serviceWeight"`

	// Instrumentation
	LogInjection  bool   `json:"logInjection"`
	Metrics       string `json:"metrics"`
	SpanDrop      string `json:"spanDrop"`
	CustomHeaders string `json:"customHeaders"`

	// Emergency
	EmergencyOff   bool   `json:"emergencyOff"`
	ServiceDisable string `json:"serviceDisable"`

	// Performance
	DropOnFull  bool  `json:"dropOnFull"`
	BatchSize   int   `json:"batchSize"`
	ExportDelay int64 `json:"exportDelay"`
	RetryCount  int   `json:"retryCount"`
}

// AgentConfigResponse wraps RemoteConfig with versioning metadata for agent polling.
type AgentConfigResponse struct {
	RemoteConfig
	ConfigVersion int64 `json:"configVersion"`
}

// ServiceKey identifies a service+environment pair for scoped config lookup.
type ServiceKey struct {
	Service string
	Env     string
}

func (k ServiceKey) String() string {
	if k.Env == "" {
		return k.Service
	}
	return k.Service + ":" + k.Env
}

func defaultConfig() RemoteConfig {
	return RemoteConfig{
		HeadSampleRate: 1.0,
		TailPolicy:     "error,slow,cluster",
		TargetTps:      0,
		ServiceWeight:  "",
		LogInjection:   true,
		Metrics:        "all",
		SpanDrop:       "",
		CustomHeaders:  "",
		EmergencyOff:   false,
		ServiceDisable: "",
		DropOnFull:     true,
		BatchSize:      512,
		ExportDelay:    5000,
		RetryCount:     3,
	}
}

// Validate checks that all fields are within acceptable ranges.
func (cfg RemoteConfig) Validate() error {
	var errs []error
	if cfg.HeadSampleRate < 0.0 || cfg.HeadSampleRate > 1.0 {
		errs = append(errs, fmt.Errorf("headSampleRate must be between 0 and 1, got %v", cfg.HeadSampleRate))
	}
	if cfg.BatchSize < 1 || cfg.BatchSize > 10000 {
		errs = append(errs, fmt.Errorf("batchSize must be between 1 and 10000, got %d", cfg.BatchSize))
	}
	if cfg.ExportDelay < 100 || cfg.ExportDelay > 60000 {
		errs = append(errs, fmt.Errorf("exportDelay must be between 100 and 60000 ms, got %d", cfg.ExportDelay))
	}
	if cfg.RetryCount < 0 || cfg.RetryCount > 10 {
		errs = append(errs, fmt.Errorf("retryCount must be between 0 and 10, got %d", cfg.RetryCount))
	}
	if cfg.TargetTps < 0 {
		errs = append(errs, fmt.Errorf("targetTps must be >= 0, got %d", cfg.TargetTps))
	}
	return errors.Join(errs...)
}

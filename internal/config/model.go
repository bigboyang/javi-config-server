package config

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

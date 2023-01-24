package container

import "time"

type Config struct {
	// ServiceStartMaxTime is a maximum time to wait for a service to start.
	ServiceStartMaxTime time.Duration
	// ServiceStartMaxInterval is a maximum interval between service start attempts.
	ServiceStartMaxInterval time.Duration

	//ServiceNameRandomSuffix is a flag to enable random suffixes for service names.
	ServiceNameRandomSuffix bool
	// ServiceNameRandomSuffixLength is a length of random suffixes for service names.
	ServiceNameRandomSuffixLength int
}

// NewConfig returns a new Config with sane defaults.
func NewConfig() *Config {
	return &Config{
		ServiceStartMaxTime:     80 * time.Second,
		ServiceStartMaxInterval: 1 * time.Second,

		ServiceNameRandomSuffix:       true,
		ServiceNameRandomSuffixLength: 8,
	}
}

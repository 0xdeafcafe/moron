package config

// Config holds application configuration.
type Config struct {
	RepoPath string
}

// Default returns the default configuration.
func Default() Config {
	return Config{}
}

package config

// Config holds information about how the peregrine backend is configured.
type Config struct {
	Server struct {
		Address string `json:"address"`
		Origin  string `json:"origin"`
	} `json:"server"`
}

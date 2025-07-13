package configuration

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

// Configuration represents the application configuration.
type Configuration struct {
	ServerCfg  ServerConf `json:"server_cfg" toml:"server_cfg"`
	UseCORS    bool       `json:"use_cors" toml:"use_cors"`
	DocsFolder string     `json:"docs_folder" toml:"docs_folder"`
}

// ServerConf contains the server configuration details.
type ServerConf struct {
	Host           string `json:"host" toml:"host"`
	Port           int    `json:"port" toml:"port"`
	MainPath       string `json:"main_path" toml:"main_path"`
	StartupMessage bool   `json:"startup_message" toml:"startup_message"`
}

// PrepareConfiguration initializes the configuration from a file specified by the command line flag.
func PrepareConfiguration() (*Configuration, error) {
	confName := flag.String("conf", "", "Config file path")
	flag.Parse()

	// Explicitly call PrepareFileConfiguration due in future we can use different
	// configuration sources (e.g. environment variables)
	mainCfg, err := PrepareFileConfiguration(*confName)
	if err != nil {
		return nil, errors.Wrap(err, "Can't read configuration file")
	}
	return mainCfg, nil
}

// PrepareFileConfiguration reads the configuration from a specified file and unmarshals it into a Configuration struct.
func PrepareFileConfiguration(fname string) (*Configuration, error) {
	configFile, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	cfg := &Configuration{}
	ext := strings.ToLower(filepath.Ext(fname))

	switch ext {
	case ".json":
		err = json.Unmarshal(configFile, cfg)
	case ".toml":
		err = toml.Unmarshal(configFile, cfg)
	default:
		return nil, errors.New("unsupported config file format, use .json or .toml")
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to parse config file")
	}

	return cfg, nil
}

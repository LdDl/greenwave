package configuration

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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

	// Read configuration from the specified file if provided
	if confName != nil && *confName != "" {
		mainCfg, err := PrepareFileConfiguration(*confName)
		if err != nil {
			return nil, err
		}
		return mainCfg, nil
	}

	// Read configuration from environment variables
	if _, err := os.Stat(".env"); err == nil {
		log.Info().Str("scope", "configuration").Msg("Found '.env' file. Trying to loading environment variables")
		err := godotenv.Load()
		if err != nil {
			return nil, err
		}
	}

	serverHost := os.Getenv("SERVER_HOST")
	serverPortStr := os.Getenv("SERVER_PORT")
	serverPort, err := strconv.Atoi(serverPortStr)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid SERVER_PORT environment variable: '%s'", serverPortStr)
	}
	serverMainPath := os.Getenv("SERVER_MAIN_PATH")
	serverStartupMessageStr := os.Getenv("SERVER_STARTUP_MESSAGE")
	serverStartupMessage, err := strconv.ParseBool(serverStartupMessageStr)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid SERVER_STARTUP_MESSAGE environment variable: '%s'", serverStartupMessageStr)
	}

	corsEnableStr := os.Getenv("USE_CORS")
	corsEnable, err := strconv.ParseBool(corsEnableStr)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid USE_CORS environment variable: '%s'", corsEnableStr)
	}
	docsPathStr := os.Getenv("DOCS_PATH")

	mainCfg := &Configuration{
		ServerCfg: ServerConf{
			Host:           serverHost,
			Port:           serverPort,
			MainPath:       serverMainPath,
			StartupMessage: serverStartupMessage,
		},
		UseCORS:    corsEnable,
		DocsFolder: docsPathStr,
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

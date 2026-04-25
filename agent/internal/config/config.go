package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"databasus-agent/internal/logger"
)

var log = logger.GetLogger()

const configFileName = "databasus.json"

type Config struct {
	DatabasusHost          string `json:"databasusHost"`
	DbID                   string `json:"dbId"`
	Token                  string `json:"token"`
	PgHost                 string `json:"pgHost"`
	PgPort                 int    `json:"pgPort"`
	PgUser                 string `json:"pgUser"`
	PgPassword             string `json:"pgPassword"`
	PgType                 string `json:"pgType"`
	PgHostBinDir           string `json:"pgHostBinDir"`
	PgDockerContainerName  string `json:"pgDockerContainerName"`
	PgWalDir               string `json:"pgWalDir"`
	IsDeleteWalAfterUpload *bool  `json:"deleteWalAfterUpload"`
	InsecureSkipVerify     bool   `json:"insecureSkipVerify,omitempty"`

	flags parsedFlags
}

// LoadFromJSONAndArgs reads databasus.json into the struct
// and overrides JSON values with any explicitly provided CLI flags.
func (c *Config) LoadFromJSONAndArgs(fs *flag.FlagSet, args []string) {
	c.loadFromJSON()
	c.applyDefaults()
	c.initSources()

	c.flags.databasusHost = fs.String(
		"databasus-host",
		"",
		"Databasus server URL (e.g. http://your-server:4005)",
	)
	c.flags.dbID = fs.String("db-id", "", "Database ID")
	c.flags.token = fs.String("token", "", "Agent token")
	c.flags.pgHost = fs.String("pg-host", "", "PostgreSQL host")
	c.flags.pgPort = fs.Int("pg-port", 0, "PostgreSQL port")
	c.flags.pgUser = fs.String("pg-user", "", "PostgreSQL user")
	c.flags.pgPassword = fs.String("pg-password", "", "PostgreSQL password")
	c.flags.pgType = fs.String("pg-type", "", "PostgreSQL type: host or docker")
	c.flags.pgHostBinDir = fs.String("pg-host-bin-dir", "", "Path to PG bin directory (host mode)")
	c.flags.pgDockerContainerName = fs.String("pg-docker-container-name", "", "Docker container name (docker mode)")
	c.flags.pgWalDir = fs.String("pg-wal-dir", "", "Path to WAL queue directory")
	c.flags.insecureSkipVerify = fs.Bool(
		"insecure-skip-verify",
		false,
		"Skip TLS certificate verification for HTTPS to Databasus (self-signed or dev only; reduces security)",
	)

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	c.applyFlags(fs)
	log.Info("========= Loading config ============")
	c.logConfigSources()
	log.Info("========= Config has been loaded ====")
}

// SaveToJSON writes the current struct to databasus.json.
func (c *Config) SaveToJSON() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFileName, data, 0o644)
}

func (c *Config) LoadFromJSON() {
	c.loadFromJSON()
	c.applyDefaults()
}

func (c *Config) loadFromJSON() {
	data, err := os.ReadFile(configFileName)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("No databasus.json found, will create on save")
			return
		}

		log.Warn("Failed to read databasus.json", "error", err)

		return
	}

	if err := json.Unmarshal(data, c); err != nil {
		log.Warn("Failed to parse databasus.json", "error", err)

		return
	}

	log.Info("Configuration loaded from " + configFileName)
}

func (c *Config) applyDefaults() {
	if c.PgPort == 0 {
		c.PgPort = 5432
	}

	if c.PgType == "" {
		c.PgType = "host"
	}

	if c.IsDeleteWalAfterUpload == nil {
		c.IsDeleteWalAfterUpload = new(bool)
		*c.IsDeleteWalAfterUpload = true
	}
}

func (c *Config) initSources() {
	c.flags.sources = map[string]string{
		"databasus-host":           "not configured",
		"db-id":                    "not configured",
		"token":                    "not configured",
		"pg-host":                  "not configured",
		"pg-port":                  "not configured",
		"pg-user":                  "not configured",
		"pg-password":              "not configured",
		"pg-type":                  "not configured",
		"pg-host-bin-dir":          "not configured",
		"pg-docker-container-name": "not configured",
		"pg-wal-dir":              "not configured",
		"delete-wal-after-upload": "not configured",
		"insecure-skip-verify":    "default (false)",
	}

	if c.DatabasusHost != "" {
		c.flags.sources["databasus-host"] = configFileName
	}

	if c.DbID != "" {
		c.flags.sources["db-id"] = configFileName
	}

	if c.Token != "" {
		c.flags.sources["token"] = configFileName
	}

	if c.PgHost != "" {
		c.flags.sources["pg-host"] = configFileName
	}

	// PgPort always has a value after applyDefaults
	c.flags.sources["pg-port"] = configFileName

	if c.PgUser != "" {
		c.flags.sources["pg-user"] = configFileName
	}

	if c.PgPassword != "" {
		c.flags.sources["pg-password"] = configFileName
	}

	// PgType always has a value after applyDefaults
	c.flags.sources["pg-type"] = configFileName

	if c.PgHostBinDir != "" {
		c.flags.sources["pg-host-bin-dir"] = configFileName
	}

	if c.PgDockerContainerName != "" {
		c.flags.sources["pg-docker-container-name"] = configFileName
	}

	if c.PgWalDir != "" {
		c.flags.sources["pg-wal-dir"] = configFileName
	}

	// IsDeleteWalAfterUpload always has a value after applyDefaults
	c.flags.sources["delete-wal-after-upload"] = configFileName

	if c.InsecureSkipVerify {
		c.flags.sources["insecure-skip-verify"] = configFileName
	}
}

func (c *Config) applyFlags(fs *flag.FlagSet) {
	if c.flags.databasusHost != nil && *c.flags.databasusHost != "" {
		c.DatabasusHost = *c.flags.databasusHost
		c.flags.sources["databasus-host"] = "command line args"
	}

	if c.flags.dbID != nil && *c.flags.dbID != "" {
		c.DbID = *c.flags.dbID
		c.flags.sources["db-id"] = "command line args"
	}

	if c.flags.token != nil && *c.flags.token != "" {
		c.Token = *c.flags.token
		c.flags.sources["token"] = "command line args"
	}

	if c.flags.pgHost != nil && *c.flags.pgHost != "" {
		c.PgHost = *c.flags.pgHost
		c.flags.sources["pg-host"] = "command line args"
	}

	if c.flags.pgPort != nil && *c.flags.pgPort != 0 {
		c.PgPort = *c.flags.pgPort
		c.flags.sources["pg-port"] = "command line args"
	}

	if c.flags.pgUser != nil && *c.flags.pgUser != "" {
		c.PgUser = *c.flags.pgUser
		c.flags.sources["pg-user"] = "command line args"
	}

	if c.flags.pgPassword != nil && *c.flags.pgPassword != "" {
		c.PgPassword = *c.flags.pgPassword
		c.flags.sources["pg-password"] = "command line args"
	}

	if c.flags.pgType != nil && *c.flags.pgType != "" {
		c.PgType = *c.flags.pgType
		c.flags.sources["pg-type"] = "command line args"
	}

	if c.flags.pgHostBinDir != nil && *c.flags.pgHostBinDir != "" {
		c.PgHostBinDir = *c.flags.pgHostBinDir
		c.flags.sources["pg-host-bin-dir"] = "command line args"
	}

	if c.flags.pgDockerContainerName != nil && *c.flags.pgDockerContainerName != "" {
		c.PgDockerContainerName = *c.flags.pgDockerContainerName
		c.flags.sources["pg-docker-container-name"] = "command line args"
	}

	if c.flags.pgWalDir != nil && *c.flags.pgWalDir != "" {
		c.PgWalDir = *c.flags.pgWalDir
		c.flags.sources["pg-wal-dir"] = "command line args"
	}

	fs.Visit(func(f *flag.Flag) {
		if f.Name != "insecure-skip-verify" {
			return
		}

		val, err := strconv.ParseBool(f.Value.String())
		if err != nil {
			return
		}

		c.InsecureSkipVerify = val
		c.flags.sources["insecure-skip-verify"] = "command line args"
	})
}

func (c *Config) logConfigSources() {
	log.Info("databasus-host", "value", c.DatabasusHost, "source", c.flags.sources["databasus-host"])
	log.Info("db-id", "value", c.DbID, "source", c.flags.sources["db-id"])
	log.Info("token", "value", maskSensitive(c.Token), "source", c.flags.sources["token"])
	log.Info("pg-host", "value", c.PgHost, "source", c.flags.sources["pg-host"])
	log.Info("pg-port", "value", c.PgPort, "source", c.flags.sources["pg-port"])
	log.Info("pg-user", "value", c.PgUser, "source", c.flags.sources["pg-user"])
	log.Info("pg-password", "value", maskSensitive(c.PgPassword), "source", c.flags.sources["pg-password"])
	log.Info("pg-type", "value", c.PgType, "source", c.flags.sources["pg-type"])
	log.Info("pg-host-bin-dir", "value", c.PgHostBinDir, "source", c.flags.sources["pg-host-bin-dir"])
	log.Info(
		"pg-docker-container-name",
		"value",
		c.PgDockerContainerName,
		"source",
		c.flags.sources["pg-docker-container-name"],
	)
	log.Info("pg-wal-dir", "value", c.PgWalDir, "source", c.flags.sources["pg-wal-dir"])
	log.Info(
		"delete-wal-after-upload",
		"value",
		fmt.Sprintf("%v", *c.IsDeleteWalAfterUpload),
		"source",
		c.flags.sources["delete-wal-after-upload"],
	)
	log.Info(
		"insecure-skip-verify",
		"value",
		fmt.Sprintf("%v", c.InsecureSkipVerify),
		"source",
		c.flags.sources["insecure-skip-verify"],
	)
}

func maskSensitive(value string) string {
	if value == "" {
		return "(not set)"
	}

	visibleLen := max(len(value)/4, 1)

	return value[:visibleLen] + "***"
}

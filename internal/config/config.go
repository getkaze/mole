package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub    GitHubConfig    `yaml:"github"`
	LLM      LLMConfig       `yaml:"llm"`
	MySQL    MySQLConfig     `yaml:"mysql"`
	Valkey   ValkeyConfig    `yaml:"valkey"`
	Server   ServerConfig    `yaml:"server"`
	Worker   WorkerConfig    `yaml:"worker"`
	Log      LogConfig       `yaml:"log"`
	Dashboard DashboardConfig `yaml:"dashboard"`
}

type DashboardConfig struct {
	GitHubClientID     string `yaml:"github_client_id"`
	GitHubClientSecret string `yaml:"github_client_secret"`
	SessionSecret      string `yaml:"session_secret"`
	BaseURL            string `yaml:"base_url"`
}

func (c *DashboardConfig) Enabled() bool {
	return c.GitHubClientID != "" && c.GitHubClientSecret != "" && c.SessionSecret != ""
}

type GitHubConfig struct {
	AppID          int64  `yaml:"app_id"`
	PrivateKeyPath string `yaml:"private_key_path"`
	WebhookSecret  string `yaml:"webhook_secret"`
}

type LLMConfig struct {
	APIKey         string `yaml:"api_key"`
	ReviewModel    string `yaml:"review_model"`
	DeepReviewModel string `yaml:"deep_review_model"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type ValkeyConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type WorkerConfig struct {
	Count int `yaml:"count"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

func (c *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
		c.User, c.Password, c.Host, c.Port, c.Database)
}

func (c *ValkeyConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func Load(path string) (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	cfg.applyDefaults()
	cfg.applyEnvOverrides()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.MySQL.Port == 0 {
		c.MySQL.Port = 3306
	}
	if c.Valkey.Port == 0 {
		c.Valkey.Port = 6379
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Worker.Count == 0 {
		c.Worker.Count = 3
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.LLM.ReviewModel == "" {
		c.LLM.ReviewModel = "claude-sonnet-4-6"
	}
	if c.LLM.DeepReviewModel == "" {
		c.LLM.DeepReviewModel = "claude-opus-4-6"
	}
}

func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("MOLE_GITHUB_APP_ID"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			c.GitHub.AppID = id
		}
	}
	if v := os.Getenv("MOLE_GITHUB_PRIVATE_KEY_PATH"); v != "" {
		c.GitHub.PrivateKeyPath = v
	}
	if v := os.Getenv("MOLE_GITHUB_WEBHOOK_SECRET"); v != "" {
		c.GitHub.WebhookSecret = v
	}
	if v := os.Getenv("MOLE_LLM_API_KEY"); v != "" {
		c.LLM.APIKey = v
	}
	if v := os.Getenv("MOLE_LLM_REVIEW_MODEL"); v != "" {
		c.LLM.ReviewModel = v
	}
	if v := os.Getenv("MOLE_LLM_DEEP_REVIEW_MODEL"); v != "" {
		c.LLM.DeepReviewModel = v
	}
	if v := os.Getenv("MOLE_MYSQL_HOST"); v != "" {
		c.MySQL.Host = v
	}
	if v := os.Getenv("MOLE_MYSQL_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.MySQL.Port = p
		}
	}
	if v := os.Getenv("MOLE_MYSQL_DATABASE"); v != "" {
		c.MySQL.Database = v
	}
	if v := os.Getenv("MOLE_MYSQL_USER"); v != "" {
		c.MySQL.User = v
	}
	if v := os.Getenv("MOLE_MYSQL_PASSWORD"); v != "" {
		c.MySQL.Password = v
	}
	if v := os.Getenv("MOLE_VALKEY_HOST"); v != "" {
		c.Valkey.Host = v
	}
	if v := os.Getenv("MOLE_VALKEY_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Valkey.Port = p
		}
	}
	if v := os.Getenv("MOLE_SERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Server.Port = p
		}
	}
	if v := os.Getenv("MOLE_WORKER_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Worker.Count = n
		}
	}
	if v := os.Getenv("MOLE_LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
	if v := os.Getenv("MOLE_DASHBOARD_GITHUB_CLIENT_ID"); v != "" {
		c.Dashboard.GitHubClientID = v
	}
	if v := os.Getenv("MOLE_DASHBOARD_GITHUB_CLIENT_SECRET"); v != "" {
		c.Dashboard.GitHubClientSecret = v
	}
	if v := os.Getenv("MOLE_DASHBOARD_SESSION_SECRET"); v != "" {
		c.Dashboard.SessionSecret = v
	}
	if v := os.Getenv("MOLE_DASHBOARD_BASE_URL"); v != "" {
		c.Dashboard.BaseURL = v
	}
}

func (c *Config) validate() error {
	var missing []string

	if c.GitHub.AppID == 0 {
		missing = append(missing, "github.app_id")
	}
	if c.GitHub.PrivateKeyPath == "" {
		missing = append(missing, "github.private_key_path")
	}
	if c.GitHub.WebhookSecret == "" {
		missing = append(missing, "github.webhook_secret")
	}
	if c.LLM.APIKey == "" {
		missing = append(missing, "llm.api_key")
	}
	if c.MySQL.Host == "" {
		missing = append(missing, "mysql.host")
	}
	if c.MySQL.Database == "" {
		missing = append(missing, "mysql.database")
	}
	if c.MySQL.User == "" {
		missing = append(missing, "mysql.user")
	}
	if c.Valkey.Host == "" {
		missing = append(missing, "valkey.host")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required config fields: %s", strings.Join(missing, ", "))
	}

	return nil
}

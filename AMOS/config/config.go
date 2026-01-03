package config

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for AMOS
type Config struct {
	// Kubernetes configuration
	Kubernetes KubernetesConfig `yaml:"kubernetes"`

	// ADK/Gemini configuration
	ADK ADKConfig `yaml:"adk"`

	// Email/SMTP configuration
	Email EmailConfig `yaml:"email"`

	// Remediation configuration
	Remediation RemediationConfig `yaml:"remediation"`

	// Monitoring configuration
	Monitoring MonitoringConfig `yaml:"monitoring"`
}

// KubernetesConfig holds K8s cluster connection settings
type KubernetesConfig struct {
	// InCluster indicates whether to use in-cluster config
	InCluster bool `yaml:"inCluster"`
	// KubeConfigPath is the path to kubeconfig file (for external usage)
	KubeConfigPath string `yaml:"kubeConfigPath"`
	// Namespaces to monitor (empty = all namespaces)
	Namespaces []string `yaml:"namespaces"`
}

// ADKConfig holds Google ADK settings
type ADKConfig struct {
	// APIKey for Gemini (can also use GOOGLE_API_KEY env var)
	APIKey string `yaml:"apiKey"`
	// Model to use (default: gemini-2.0-flash)
	Model string `yaml:"model"`
	// ProjectID for Vertex AI (optional)
	ProjectID string `yaml:"projectId"`
	// Location for Vertex AI (optional)
	Location string `yaml:"location"`
}

// EmailConfig holds SMTP settings
type EmailConfig struct {
	// SMTPHost is the SMTP server hostname
	SMTPHost string `yaml:"smtpHost"`
	// SMTPPort is the SMTP server port
	SMTPPort int `yaml:"smtpPort"`
	// Username for SMTP authentication
	Username string `yaml:"username"`
	// Password for SMTP authentication
	Password string `yaml:"password"`
	// FromAddress is the sender email address
	FromAddress string `yaml:"fromAddress"`
	// EngineerEmail is the email to notify on issues
	EngineerEmail string `yaml:"engineerEmail"`
	// UseTLS enables TLS for SMTP connection
	UseTLS bool `yaml:"useTls"`
}

// RemediationConfig holds remediation settings
type RemediationConfig struct {
	// MaxRetries is the maximum number of fix attempts (default: 2)
	MaxRetries int `yaml:"maxRetries"`
	// RetryBackoffSeconds is the wait time between retries
	RetryBackoffSeconds int `yaml:"retryBackoffSeconds"`
	// EnableRollback enables automatic rollback on failure
	EnableRollback bool `yaml:"enableRollback"`
}

// MonitoringConfig holds monitoring settings
type MonitoringConfig struct {
	// PollIntervalSeconds is the interval between monitoring checks
	PollIntervalSeconds int `yaml:"pollIntervalSeconds"`
	// HealthCheckPort is the port for health check endpoint
	HealthCheckPort int `yaml:"healthCheckPort"`
	// EnableMetrics enables Prometheus metrics endpoint
	EnableMetrics bool `yaml:"enableMetrics"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Kubernetes: KubernetesConfig{
			InCluster:      true,
			KubeConfigPath: "",
			Namespaces:     []string{}, // All namespaces
		},
		ADK: ADKConfig{
			Model: "gemini-2.0-flash",
		},
		Email: EmailConfig{
			SMTPPort: 587,
			UseTLS:   true,
		},
		Remediation: RemediationConfig{
			MaxRetries:          2,
			RetryBackoffSeconds: 30,
			EnableRollback:      true,
		},
		Monitoring: MonitoringConfig{
			PollIntervalSeconds: 30,
			HealthCheckPort:     8080,
			EnableMetrics:       true,
		},
	}
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Override with environment variables
	cfg.loadFromEnv()

	return cfg, nil
}

// LoadFromEnv loads configuration from environment variables only
func LoadFromEnv() *Config {
	cfg := DefaultConfig()
	cfg.loadFromEnv()
	return cfg
}

// loadFromEnv overrides config values from environment variables
func (c *Config) loadFromEnv() {
	// Kubernetes
	if v := os.Getenv("AMOS_IN_CLUSTER"); v == "false" {
		c.Kubernetes.InCluster = false
	}
	if v := os.Getenv("KUBECONFIG"); v != "" {
		c.Kubernetes.KubeConfigPath = v
	}

	// ADK
	if v := os.Getenv("GOOGLE_API_KEY"); v != "" {
		c.ADK.APIKey = v
	}
	if v := os.Getenv("AMOS_ADK_MODEL"); v != "" {
		c.ADK.Model = v
	}
	if v := os.Getenv("GOOGLE_CLOUD_PROJECT"); v != "" {
		c.ADK.ProjectID = v
	}
	if v := os.Getenv("GOOGLE_CLOUD_LOCATION"); v != "" {
		c.ADK.Location = v
	}

	// Email
	if v := os.Getenv("AMOS_SMTP_HOST"); v != "" {
		c.Email.SMTPHost = v
	}
	if v := os.Getenv("AMOS_SMTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Email.SMTPPort = port
		}
	}
	if v := os.Getenv("AMOS_SMTP_USERNAME"); v != "" {
		c.Email.Username = v
	}
	if v := os.Getenv("AMOS_SMTP_PASSWORD"); v != "" {
		c.Email.Password = v
	}
	if v := os.Getenv("AMOS_FROM_EMAIL"); v != "" {
		c.Email.FromAddress = v
	}
	if v := os.Getenv("AMOS_ENGINEER_EMAIL"); v != "" {
		c.Email.EngineerEmail = v
	}

	// Remediation
	if v := os.Getenv("AMOS_MAX_RETRIES"); v != "" {
		if retries, err := strconv.Atoi(v); err == nil {
			c.Remediation.MaxRetries = retries
		}
	}
}

// PollInterval returns the poll interval as a Duration
func (c *Config) PollInterval() time.Duration {
	return time.Duration(c.Monitoring.PollIntervalSeconds) * time.Second
}

// RetryBackoff returns the retry backoff as a Duration
func (c *Config) RetryBackoff() time.Duration {
	return time.Duration(c.Remediation.RetryBackoffSeconds) * time.Second
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// ADK API key is required
	if c.ADK.APIKey == "" {
		return ErrMissingAPIKey
	}

	// Email configuration is required for notifications
	if c.Email.SMTPHost == "" {
		return ErrMissingSMTPHost
	}
	if c.Email.EngineerEmail == "" {
		return ErrMissingEngineerEmail
	}

	return nil
}

// Custom errors
var (
	ErrMissingAPIKey        = configError("GOOGLE_API_KEY is required")
	ErrMissingSMTPHost      = configError("SMTP host is required")
	ErrMissingEngineerEmail = configError("Engineer email is required")
)

type configError string

func (e configError) Error() string {
	return string(e)
}

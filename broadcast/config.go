package broadcast

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

const (
	defaultIGTVMinDuration = 2
	defaultLogLevel        = "info"
)

var (
	encoderCommand = "ffmpeg"
	encoderArgs    = []string{
		"-analyzeduration", "20M",
		"-probesize", "20M",
		"-c", "copy",
		"-bufsize", "4096k",
		"-max_muxing_queue_size", "1024",
		"-loglevel", "error",
	}
)

type Encoder struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

type IGTV struct {
	Enabled     bool   `yaml:"enabled"`
	MinDuration int    `yaml:"min_duration"`
	ShareToFeed bool   `yaml:"share_to_feed"`
	Description string `yaml:"description"`
}

type Config struct {
	InputURL string              `yaml:"input_url"`
	Accounts map[string]*Account `yaml:"accounts"`
	BindIP   string              `yaml:"bind_ip"`
	BindPort int                 `yaml:"bind_port"`
	Encoder  Encoder             `yaml:"encoder"`
	Title    string              `yaml:"title"`
	IGTV     IGTV                `yaml:"igtv"`
	Notify   bool                `yaml:"notify"`
	LogLevel string              `yaml:"log_level"`
	path     string
}

type Account struct {
	Password string `yaml:"password"`
	Token    string `yaml:"token"`
}

func LoadConfig(configPath string) (*Config, error) {
	f, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(f, &config); err != nil {
		return nil, err
	}

	if config.Encoder.Command == "" {
		config.Encoder.Command = encoderCommand
	}

	if config.Encoder.Args == nil {
		config.Encoder.Args = encoderArgs
	}

	if config.IGTV.MinDuration < defaultIGTVMinDuration {
		config.IGTV.MinDuration = defaultIGTVMinDuration
	}

	if config.LogLevel == "" {
		config.LogLevel = defaultLogLevel
	}

	config.path = configPath

	return &config, nil
}

func (c *Config) SaveConfig() error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	f, err := os.Create(c.path)
	if err != nil {
		return err
	}

	_, err = f.WriteString(string(b))
	if err != nil {
		f.Close()
		return err
	}

	return f.Close()
}

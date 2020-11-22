package broadcast

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

const (
	defaultIGTVMinDuration = 2
	defaultLogLevel        = "info"
	defaultHeight          = 1280
	defaultWidth           = 720
	defaultPollInterval    = 2
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
	Height  int      `yaml:"height"`
	Width   int      `yaml:"width"`
}

type IGTV struct {
	Enabled     bool   `yaml:"enabled"`
	MinDuration int    `yaml:"min_duration"`
	ShareToFeed bool   `yaml:"share_to_feed"`
	Description string `yaml:"description"`
}

type Logging struct {
	Enabled        bool   `yaml:"enabled"`
	ViewerLogFile  string `yaml:"viewer_log_file"`
	CommentLogFile string `yaml:"comment_log_file"`
}

type Config struct {
	InputURL     string              `yaml:"input_url"`
	Accounts     map[string]*Account `yaml:"accounts"`
	BindIP       string              `yaml:"bind_ip"`
	BindPort     int                 `yaml:"bind_port"`
	Encoder      Encoder             `yaml:"encoder"`
	Title        string              `yaml:"title"`
	IGTV         IGTV                `yaml:"igtv"`
	Notify       bool                `yaml:"notify"`
	LogLevel     string              `yaml:"log_level"`
	PollInterval int                 `yaml:"poll_interval"`
	Logging      Logging             `yaml:"logging"`
	path         string
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

	if config.Encoder.Height == 0 {
		config.Encoder.Height = defaultHeight
	}

	if config.Encoder.Width == 0 {
		config.Encoder.Width = defaultWidth
	}

	if config.IGTV.MinDuration < defaultIGTVMinDuration {
		config.IGTV.MinDuration = defaultIGTVMinDuration
	}

	if config.PollInterval == 0 {
		config.PollInterval = defaultPollInterval
	}

	if config.LogLevel == "" {
		config.LogLevel = defaultLogLevel
	}

	if config.Logging.ViewerLogFile == "" {
		config.Logging.ViewerLogFile = "/var/log/broadcastd/viewers.csv"
	}

	if config.Logging.CommentLogFile == "" {
		config.Logging.CommentLogFile = "/var/log/broadcastd/comments.csv"
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

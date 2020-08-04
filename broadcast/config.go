package broadcast

import (
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
	"os"
)

var (
	encoderCommand = "ffmpeg"
	encoderArgs    = []string{
		"-analyzeduration", "20M",
		"-probesize", "20M",
		"-c", "copy",
		"-bufsize", "4096k",
		"-max_muxing_queue_size", "1024",
	}
)

type Encoder struct {
	Command string   `mapstructure:"command" yaml:"command"`
	Args    []string `mapstructure:"args" yaml:"args"`
}

type Config struct {
	InputURL string              `mapstructure:"input_url" yaml:"input_url"`
	Accounts map[string]*Account `mapstructure:"accounts" yaml:"accounts"`
	BindIP   string              `mapstructure:"bind_ip" yaml:"bind_ip"`
	BindPort int                 `mapstructure:"bind_port" yaml:"bind_port"`
	Encoder  Encoder             `mapstructure:"encoder" yaml:"encoder"`
	Message  string              `mapstructure:"message" yaml:"message"`
}

type Account struct {
	Password string `mapstructure:"password"`
	Token    string `mapstructure:"token"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/broadcastd/")
	viper.AddConfigPath("$HOME/.broadcastd")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config *Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	if config.Encoder.Command == "" {
		config.Encoder.Command = encoderCommand
	}

	if config.Encoder.Args == nil {
		config.Encoder.Args = encoderArgs
	}

	return config, nil
}

func (c *Config) SaveConfig() error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	f, err := os.Create(viper.ConfigFileUsed())
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

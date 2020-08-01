package config

import (
	"github.com/spf13/viper"
)

var (
	encoderPath = "ffmpeg"
	encoderArgs = []string{
		"-analyzeduration", "20M",
		"-probesize", "20M",
		"-c", "copy",
		"-bufsize", "4096k",
		"-max_muxing_queue_size", "1024",
	}
)

type Encoder struct {
	Path string   `mapstructure:"path"`
	Args []string `mapstructure:"args"`
}

type Config struct {
	InputURL string             `mapstructure:"input_url"`
	Accounts map[string]Account `mapstructure:"accounts"`
	BindIP   string             `mapstructure:"bind_ip"`
	BindPort int                `mapstructure:"bind_port"`
	Encoder  Encoder            `mapstructure:"encoder"`
}

type Account struct {
	Token string `mapstructure:"token"`
}

func Load() (*Config, error) {
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

	if config.Encoder.Path == "" {
		config.Encoder.Path = encoderPath
	}

	if config.Encoder.Args == nil {
		config.Encoder.Args = encoderArgs
	}

	return config, nil
}

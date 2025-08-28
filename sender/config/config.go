package config

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	GRPC  GRPCConfig  `yaml:"grpc"`
	Redis RedisConfig `yaml:"redis"`
}

type GRPCConfig struct {
	Port int `yaml:"port"`
}

type RedisConfig struct {
	Address string `yaml:"address"`
}

func parseFlags() string {
	cfgPath := flag.String("config", "", "path to config")
	flag.Parse()
	return *cfgPath
}

func MustLoad() (*Config, error) {
	cfgPath := parseFlags()

	if cfgPath == "" {
		return nil, errors.New("config path is empty")
	}

	if _, err := os.Stat(cfgPath); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("config is not exists: %v", err)
	}

	var config Config
	if err := cleanenv.ReadConfig(cfgPath, &config); err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	return &config, nil
}

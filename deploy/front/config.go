package main

import (
	"encoding/json"
	"os"

	"media-box-ng/business/entity"
	"media-box-ng/pkg/logger"
)

const (
	defaultConfigPath = "media-box-front.json"
)

type Config struct {
	Broker   *BrokerConfig              `json:"broker"`
	Logger   *LoggerConfig              `json:"logger"`
	Image    *ImageConfig               `json:"image"`
	Channels map[string]*entity.Channel `json:"channels"`
}

type LoggerConfig struct {
	Level             string `json:"level"`
	TimeFieldFormat   string `json:"time_field_format"`
	PrettyPrint       bool   `json:"pretty_print"`
	DisableSampling   bool   `json:"disable_sampling"`
	RedirectStdLogger bool   `json:"redirect_std_logger"`
	ErrorStack        bool   `json:"error_stack"`
	ShowCaller        bool   `json:"show_caller"`
}

type BrokerConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	StateTopic string `json:"state_topic"`
	ClientID   string `json:"client_id"`
	UserName   string `json:"user_name"`
	Password   string `json:"password"`
}

type ImageConfig struct {
	OutputPath string `json:"output_path"`
}

var (
	cfg = &Config{}
)

func init() {
	log := logger.NewDefaultZerolog()

	path, ok := os.LookupEnv("MEDIA_BOX_FRONT_CONFIG")
	if !ok {
		path = defaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		log.Fatal().Msg(err.Error())
	}
}

package main

import (
	"os"
	"os/signal"
	"syscall"

	"media-box-ng/adapter/broker"
	"media-box-ng/adapter/image"
	"media-box-ng/business/usecase"
	"media-box-ng/pkg/logger"
)

var (
	log *logger.Zerolog

	brokerClient *broker.Client
	imageClient  *image.Client

	guiUseCase *usecase.GUIUseCase
)

func main() {
	defer shutdown()

	log = logger.NewZerolog(logger.ZeroConfig{
		Level:             cfg.Logger.Level,
		TimeFieldFormat:   cfg.Logger.TimeFieldFormat,
		PrettyPrint:       cfg.Logger.PrettyPrint,
		DisableSampling:   cfg.Logger.DisableSampling,
		RedirectStdLogger: cfg.Logger.RedirectStdLogger,
		ErrorStack:        cfg.Logger.ErrorStack,
		ShowCaller:        cfg.Logger.ShowCaller,
	})

	initAdapters()
	initUseCases()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	<-quit
}

func initAdapters() {
	var err error
	brokerClient, err = broker.NewBrokerClient(&broker.Config{
		Host:       cfg.Broker.Host,
		Port:       cfg.Broker.Port,
		StateTopic: cfg.Broker.StateTopic,
		ClientID:   cfg.Broker.ClientID,
		UserName:   cfg.Broker.UserName,
		Password:   cfg.Broker.Password,
	}, log)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	imageClient = image.NewImageClient(&image.Config{
		OutputPath: cfg.Image.OutputPath,
	}, log)
}

func initUseCases() {
	usecase.SetStateTopic(cfg.Broker.StateTopic)

	var err error
	guiUseCase, err = usecase.NewGUIUseCase(brokerClient, imageClient, log)
	if err != nil {
		log.Fatal().Msgf(err.Error())
	}
}

func shutdown() {
	brokerClient.Close()
}

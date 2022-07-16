package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/soellman/pidfile"

	"media-box-ng/adapter/broker"
	"media-box-ng/adapter/shoutcast"
	"media-box-ng/adapter/uds"
	"media-box-ng/business/usecase"
	"media-box-ng/pkg/logger"
)

var (
	log *logger.Zerolog

	brokerClient *broker.Client
	udsServer    *uds.Server
	mp3player    *shoutcast.MP3Shoutcast

	stateUseCase *usecase.StateUseCase
)

const (
	pidFile = "/tmp/media-box.pid"
)

// GOOS=linux GOARCH=arm go build

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

	if err := pidfile.Write(pidFile); err != nil {
		log.Fatal().Msgf("failed to create pid file: %v", err)
	}

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

	udsServer, err = uds.NewUDSServer(&uds.ServerConfig{
		SocketPath:     cfg.UDS.ServerSocket,
		CommandTimeout: 2,
	}, log)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	mp3player = shoutcast.NewMP3Shoutcast(&shoutcast.Config{}, log)
}

func initUseCases() {
	stateUseCase = usecase.NewStateUseCase(&usecase.StateConfig{
		Channels: cfg.Channels,
		Mute:     cfg.VolumeControl.Mute,
		Unmute:   cfg.VolumeControl.Unmute,
	}, brokerClient, mp3player, log)

	uds.SetStateUseCase(stateUseCase)
}

func shutdown() {
	if r := recover(); r != nil {
		fmt.Println(r)
	}
	_ = pidfile.Remove(pidFile)
	brokerClient.Close()
	udsServer.Close()
}

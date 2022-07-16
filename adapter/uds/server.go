package uds

import (
	"net"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"media-box-ng/pkg/logger"
)

type ServerConfig struct {
	SocketPath     string
	CommandTimeout int64
}

type Server struct {
	cfg      *ServerConfig
	log      *logger.Zerolog
	listener net.Listener

	lastCommandName      string
	lastCommandTimeStamp int64
}

func NewUDSServer(cfg *ServerConfig, log *logger.Zerolog) (*Server, error) {
	s := &Server{
		cfg: cfg,
		log: log,
	}

	_ = os.Remove(cfg.SocketPath)

	var err error
	s.listener, err = net.Listen("unix", cfg.SocketPath)
	if err != nil {
		return nil, err
	}

	go s.handler()

	return s, nil
}

func (s *Server) handler() {
	defer func() {
		if err := s.listener.Close(); err != nil {
			log.Error().Msgf("close error: %v", err)
		}
	}()

	for {
		fd, err := s.listener.Accept()
		if err == net.ErrClosed {
			break
		} else if err != nil {
			s.log.Error().Msgf("accept error: %v", err)
			continue
		}
		go s.reader(fd)
	}
}

func (s *Server) reader(c net.Conn) {
	for {
		buf := make([]byte, 512)
		nr, err := c.Read(buf[:])
		if err != nil {
			return
		}

		cmd := strings.TrimSpace(string(buf[0:nr]))

		if time.Now().Unix()-s.lastCommandTimeStamp < s.cfg.CommandTimeout && s.lastCommandName == cmd {
			s.log.Debug().Msgf("chatter!")
			continue
		}

		s.log.Debug().Msgf("command: %s", cmd)

		s.lastCommandName = cmd
		s.lastCommandTimeStamp = time.Now().Unix()

		switch cmd {
		case "POWER":
			stateUseCase.Power()
		case "MUTE":
			stateUseCase.Mute()
		case "PAUSE":
			stateUseCase.Pause()
		case "NEXT":
			stateUseCase.NextChannel()
		case "PREV":
			stateUseCase.PrevChannel()
		case "DEFAULT_STATION":
			stateUseCase.DefaultChannel()
		default:
			s.log.Error().Msgf("unknown command: %s", cmd)
		}
	}
}

func (s *Server) Close() {
	if s.listener != nil {
		_ = s.listener.Close()
		_ = os.Remove(s.cfg.SocketPath)
	}
}

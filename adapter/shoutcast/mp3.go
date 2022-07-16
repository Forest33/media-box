package shoutcast

import (
	"io"
	"sync"

	"github.com/hajimehoshi/oto"
	"github.com/romantomjak/shoutcast"

	"media-box-ng/pkg/minimp3"

	"media-box-ng/business/entity"
	"media-box-ng/pkg/logger"
)

type MP3Shoutcast struct {
	cfg                 *Config
	log                 *logger.Zerolog
	stream              *shoutcast.Stream
	decoder             *minimp3.Decoder
	player              *oto.Player
	wgPlaying           sync.WaitGroup
	isPlay              bool
	chPause             chan struct{}
	changeTrackCallback entity.ChangeTrackCallback
	disconnectCallback  entity.DisconnectCallback
}

func NewMP3Shoutcast(cfg *Config, log *logger.Zerolog) *MP3Shoutcast {
	return &MP3Shoutcast{
		cfg: cfg,
		log: log,
	}
}

func (s *MP3Shoutcast) Play(url string) error {
	var err error
	s.stream, err = shoutcast.Open(url)
	if err != nil {
		return err
	}

	s.stream.MetadataCallbackFunc = s.metadataCallback

	if s.decoder, err = minimp3.NewDecoder(s.stream); err != nil {
		return err
	}
	<-s.decoder.Started()

	s.log.Debug().Msgf("convert audio sample quality: %dKbps rate: %d, channels: %d", s.decoder.Kbps, s.decoder.SampleRate, s.decoder.Channels)

	var playerContext *oto.Context
	if playerContext, err = oto.NewContext(s.decoder.SampleRate, s.decoder.Channels, 2, 4096); err != nil {
		return err
	}

	s.player = playerContext.NewPlayer()

	go func() {
		var playErr error

		defer func() {
			s.log.Debug().Msgf("player finished error: %v", playErr)

			s.isPlay = false
			s.wgPlaying.Done()

			if s.stream != nil {
				if err := s.stream.Close(); err != nil {
					s.log.Error().Msgf("failed to close stream: %v", err)
				}
			}

			if s.decoder != nil {
				s.decoder.Close()
			}

			if s.player != nil {
				if err := s.player.Close(); err != nil {
					s.log.Error().Msgf("failed to close player: %v", err)
				}
			}
			
			if playerContext != nil {
				if err := playerContext.Close(); err != nil {
					s.log.Error().Msgf("failed to close player context: %v", err)
				}
			}

			if playErr != nil && s.disconnectCallback != nil {
				s.disconnectCallback()
			}
		}()

		s.isPlay = true
		s.wgPlaying = sync.WaitGroup{}
		s.wgPlaying.Add(1)

		for s.isPlay {
			var data = make([]byte, 512)

			_, playErr = s.decoder.Read(data)
			if err == io.EOF {
				break
			} else if playErr != nil {
				s.log.Error().Msgf("failed to read data from decoder: %v", err)
				break
			}

			s.isPaused()

			if _, playErr = s.player.Write(data); playErr != nil {
				s.log.Error().Msgf("failed to write decoded data: %v", err)
			}
		}
	}()

	return nil
}

func (s *MP3Shoutcast) isPaused() {
	if s.chPause != nil {
		<-s.chPause
	}
}

func (s *MP3Shoutcast) Stop() {
	if !s.isPlay {
		return
	}
	s.isPlay = false
	s.wgPlaying.Wait()
}

func (s *MP3Shoutcast) Pause() error {
	if s.chPause != nil {
		s.chPause <- struct{}{}
		close(s.chPause)
		s.chPause = nil
	} else {
		s.chPause = make(chan struct{})
	}
	return nil
}

func (s *MP3Shoutcast) SetChangeTrackCallback(cb entity.ChangeTrackCallback) {
	s.changeTrackCallback = cb
}

func (s *MP3Shoutcast) SetCloseConnectionCallback(cb entity.DisconnectCallback) {
	s.disconnectCallback = cb
}

func (s *MP3Shoutcast) metadataCallback(m *shoutcast.Metadata) {
	s.log.Debug().Msgf("now listening to: %s", m.StreamTitle)
	if s.changeTrackCallback != nil {
		s.changeTrackCallback(m.StreamTitle)
	}
}

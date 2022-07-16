package usecase

import (
	"encoding/json"
	"os/exec"
	"strings"
	"sync"
	"time"

	"media-box-ng/business/entity"
	"media-box-ng/pkg/logger"
)

type StateUseCase struct {
	cfg           *StateConfig
	broker        Broker
	player        Player
	log           *logger.Zerolog
	state         *entity.State
	mu            sync.RWMutex
	onceTrackList sync.Once
	curChannelIdx int
	pauseBegin    int64
	pauseDelay    int64
	pauseExists   bool
	trackList     []*track
}

type StateConfig struct {
	Channels []*entity.Channel
	Mute     string
	Unmute   string
}

type track struct {
	ts    int64
	title string
}

type Player interface {
	Play(url string) error
	Stop()
	Pause() error
	SetChangeTrackCallback(cb entity.ChangeTrackCallback)
	SetCloseConnectionCallback(cb entity.DisconnectCallback)
}

func NewStateUseCase(cfg *StateConfig, broker Broker, player Player, log *logger.Zerolog) *StateUseCase {
	uc := &StateUseCase{
		cfg:           cfg,
		broker:        broker,
		player:        player,
		log:           log,
		state:         &entity.State{},
		trackList:     make([]*track, 0, 10),
		curChannelIdx: -1,
	}

	uc.player.SetChangeTrackCallback(uc.changeTrackCallback)
	uc.player.SetCloseConnectionCallback(uc.disconnectConnectionCallback)

	return uc
}

func (uc *StateUseCase) Power() {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if uc.curChannelIdx == -1 {
		uc.curChannelIdx = 0
	}

	if !uc.state.Power {
		if err := uc.play(); err != nil {
			uc.log.Error().Msgf("failed to change power state: %v", err)
			return
		}
	} else {
		uc.pauseExists = false
		uc.pauseDelay = 0
		uc.clearTrackList()
		uc.player.Stop()
	}

	uc.state.Power = !uc.state.Power
	uc.publish()
}

func (uc *StateUseCase) play() error {
	uc.player.Stop()

	err := uc.player.Play(uc.cfg.Channels[uc.curChannelIdx].URL)
	if err != nil {
		return err
	}

	uc.state.Channel = uc.cfg.Channels[uc.curChannelIdx]

	return nil
}

func (uc *StateUseCase) Pause() {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	if err := uc.player.Pause(); err != nil {
		uc.log.Error().Msgf("failed to mute: %v", err)
		return
	}

	if !uc.state.Pause {
		uc.trackListNotifier()
		uc.pauseBegin = getCurrentTimestamp()
		uc.state.Pause = true
		uc.pauseExists = true
	} else {
		uc.recalculateTrackList(getCurrentTimestamp() - uc.pauseBegin)
		uc.pauseDelay += getCurrentTimestamp() - uc.pauseBegin
		uc.pauseBegin = 0
		uc.state.Pause = false
	}

	uc.publish()
}

func (uc *StateUseCase) Mute() {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	cmdArgs := strings.Split(uc.cfg.Mute, " ")
	if uc.state.Mute {
		cmdArgs = strings.Split(uc.cfg.Unmute, " ")
	}
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	_, err := cmd.Output()
	if err != nil {
		uc.log.Error().Msgf("failed to mute/unmute: %v", err)
		return
	}

	uc.state.Mute = !uc.state.Mute
	uc.publish()
}

func (uc *StateUseCase) NextChannel() {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	uc.curChannelIdx++
	if uc.curChannelIdx >= len(uc.cfg.Channels) {
		uc.curChannelIdx = 0
	}

	if err := uc.play(); err != nil {
		uc.log.Error().Msgf("failed to change channel: %v (%d)", err, uc.curChannelIdx)
		return
	}
}

func (uc *StateUseCase) PrevChannel() {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	uc.curChannelIdx--
	if uc.curChannelIdx < 0 {
		uc.curChannelIdx = len(uc.cfg.Channels) - 1
	}

	if err := uc.play(); err != nil {
		uc.log.Error().Msgf("failed to change channel: %v (%d)", err, uc.curChannelIdx)
		return
	}
}

func (uc *StateUseCase) DefaultChannel() {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	uc.curChannelIdx = 0
	if err := uc.play(); err != nil {
		uc.log.Error().Msgf("failed to change channel: %v (%d)", err, uc.curChannelIdx)
		return
	}
}

func (uc *StateUseCase) changeTrackCallback(track string) {
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	
	if !uc.state.Pause && !uc.pauseExists {
		uc.state.Channel = uc.cfg.Channels[uc.curChannelIdx]
		uc.state.Track = track
		if !uc.state.Mute {
			uc.publish()
		}
	} else {
		uc.addToTrackList(track)
	}
}

func (uc *StateUseCase) disconnectConnectionCallback() {
	for {
		time.Sleep(time.Second)
		if err := uc.play(); err != nil {
			uc.log.Error().Msgf("failed to replay: %v", err)
		} else {
			break
		}
	}
}

func (uc *StateUseCase) recalculateTrackList(delay int64) {
	for _, t := range uc.trackList {
		t.ts += delay
		uc.log.Debug().Msgf("recalculate %ds - %s (%d)", t.ts-getCurrentTimestamp(), t.title, delay)
	}
}

func (uc *StateUseCase) addToTrackList(title string) {
	uc.log.Debug().Msgf("addToTrackList %s delay=%d", title, uc.pauseDelay)
	uc.trackList = append(uc.trackList, &track{
		ts:    getCurrentTimestamp() + uc.pauseDelay,
		title: title,
	})
}

func (uc *StateUseCase) clearTrackList() {
	uc.trackList = uc.trackList[:0]
}

func (uc *StateUseCase) trackListNotifier() {
	loop := func() {
		uc.log.Debug().Msg("trackListNotifier started")
		for {
			uc.mu.Lock()
			curTs := getCurrentTimestamp()
			trackList := uc.trackList[:0]

			if !uc.state.Pause {
				for _, t := range uc.trackList {
					if t.ts <= curTs {
						uc.state.Channel = uc.cfg.Channels[uc.curChannelIdx]
						uc.state.Track = t.title
						uc.publish()
					} else {
						trackList = append(trackList, t)
					}
				}
				uc.trackList = trackList
			}

			uc.mu.Unlock()
			time.Sleep(time.Second)
		}
	}
	go uc.onceTrackList.Do(loop)
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}

func (uc *StateUseCase) publish() {
	data, err := json.Marshal(uc.state)
	if err != nil {
		uc.log.Error().Msg(err.Error())
		return
	}
	uc.broker.PublishState(data)
}

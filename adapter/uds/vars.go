package uds

type StateUseCase interface {
	Power()
	Pause()
	Mute()
	NextChannel()
	PrevChannel()
	DefaultChannel()
}

var (
	stateUseCase StateUseCase
)

func SetStateUseCase(uc StateUseCase) {
	stateUseCase = uc
}

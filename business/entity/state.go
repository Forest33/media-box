package entity

type State struct {
	Power   bool     `json:"power"`
	Mute    bool     `json:"mute"`
	Pause   bool     `json:"pause"`
	Track   string   `json:"track"`
	Channel *Channel `json:"channel"`
}

type Channel struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Img   string `json:"img"`
}

type ChangeTrackCallback func(track string)
type DisconnectCallback func()

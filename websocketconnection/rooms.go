package websocketconnection

import "github.com/google/uuid"

type Room struct {
	ID        string `json:"id"`
	Clients   []*Client `json:"clients"`
	Creator   string `json:"creator"`
	GameState string `json:"game_state"` // fen string
}

const DefaultFen string = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

func NewRoom(creator string) *Room {
	return &Room{
		ID:        uuid.NewString(),
		Creator:   creator,
		Clients: make([]*Client, 0),
		GameState: DefaultFen,
	}
}

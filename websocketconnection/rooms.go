package websocketconnection

import "github.com/google/uuid"

type Room struct {
	ID        string
	Player1   *Client
	Player2   *Client
	GameState string // fen string
}

func NewRoom(state string, creator *Client) *Room {
	return &Room{
		ID:      uuid.NewString(),
		Player1: creator,
		GameState: state,
	}
}

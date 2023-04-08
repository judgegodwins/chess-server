package websocketconnection

import (
	"github.com/google/uuid"
)

type Room struct {
	ID        string    `json:"id"`
	Clients   []*Client `json:"clients"`
	Creator   string    `json:"creator"`
	GameState string    `json:"game_state"` // fen string
}

const DefaultFen string = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

func NewRoom(creator string) *Room {
	return &Room{
		ID:        uuid.NewString(),
		Creator:   creator,
		Clients:   make([]*Client, 0),
		GameState: DefaultFen,
	}
}

func (r *Room) Broadcast(e Event, except *Client) {
	for _, client := range r.Clients {
		if except != nil && client == except {
			continue
		}
		client.PushToEgress(e)
	}
}

func (r *Room) Join(c *Client) {
	r.Clients = append(r.Clients, c)
}

func (r *Room) GetCreator() (c *Client) {
	for _, client := range r.Clients {
		if client.ID == r.Creator {
			c = client
			return
		}
	}
	return
}
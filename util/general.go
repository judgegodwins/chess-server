package util

import "fmt"

const (
	RoomIDKey              = "id"
	RoomPlayer1Key         = "player1"
	RoomPlayer1UsernameKey = "player1_username"
	RoomPlayer2UsernameKey = "player2_username"
	RoomPlayer2Key         = "player2"
	RoomGameStateKey       = "game_state"
	RoomGameStartedKey     = "active"
)

const DefaultFEN string = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

type GameStartedEnum int

const (
	GameStartedFalse GameStartedEnum = iota // 0
	GameStartedTrue // 1
)

func (n GameStartedEnum) String() string {
	return []string{"no", "yes"}[n]
}

func (w GameStartedEnum) EnumIndex() int {
	return int(w)
}

func GetRoomKey(room string) string {
	return fmt.Sprintf("room:%v", room)
}

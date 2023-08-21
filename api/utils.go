package api

const (
	ErrorMessage500 = "Something went wrong!"
)

const (
	roomIDKey      = "id"
	roomPlayer1Key = "player1"
	roomPlayer2Key = "player2"
)

func errorResponse(msg string) map[string]string {
	return map[string]string{
		"status":  "error",
		"message": msg,
	}
}

func successResponse[T interface{}](msg string, data T) map[string]interface{} {
	return map[string]interface{}{
		"status":  "success",
		"message": msg,
		"data":    data,
	}
}

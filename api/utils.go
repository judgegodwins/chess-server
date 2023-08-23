package api

import (
	"github.com/gin-gonic/gin"
	"github.com/judgegodwins/chess-server/tokens"
)

const (
	ErrorMessage500 = "Something went wrong!"
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

func GetPayload(ctx *gin.Context) (*tokens.Payload, bool) {
	v, ok := ctx.Get(string(authContextKey))

	if !ok {
		return nil, ok
	}

	payload, ok := v.(*tokens.Payload)

	return payload, ok
}

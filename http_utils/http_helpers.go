package http_utils

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
)


func SendResponse(w http.ResponseWriter, code int, payload any) {
	data, err := json.Marshal(payload)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(code)
	w.Write(data)
}

func ValidateStruct(w http.ResponseWriter, v *validator.Validate, s interface{}) ValidationErrorResponse {
	if err := v.Struct(s); err != nil {
		response := ValidationErrorResponse{
			BaseResponse: BaseResponse{
				Success: false,
				Message: "invalid body, validation failed",
			},
			Errors: lo.Map(err.(validator.ValidationErrors), func(item validator.FieldError, index int) string {
				return item.Error()
			}),
		}

		return response
	}

	return ValidationErrorResponse{}
}
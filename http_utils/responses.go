package http_utils

type BaseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type DataResponse struct {
	BaseResponse
	Data interface{} `json:"data"`
}

type ValidationErrorResponse struct {
	BaseResponse
	Errors []string `json:"errors"`
}

func NewBaseResponse(success bool, msg string) BaseResponse {
	return BaseResponse{
		Success: success,
		Message: msg,
	}
}
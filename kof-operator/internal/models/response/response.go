package response

type BasicResponse struct {
	Success      bool   `json:"success,omitempty"`
	ErrorMessage string `json:"err_msg,omitempty"`
}

func NewBasicResponse(success bool, errMessage string) *BasicResponse {
	return &BasicResponse{
		Success:      success,
		ErrorMessage: errMessage,
	}
}

package server

type ErrorResponse struct {
	status  int
	Message string `json:"message" doc:"Mô tả lỗi"`
	Code    string `json:"code" doc:"Code lỗi nội bộ" example:"INTERNAL_FAILURE"`
	Details any    `json:"details,omitempty" doc:"Chi tiết bổ sung, chỉ có trên một số Code nhất định"`
}

func (e *ErrorResponse) Error() string  { return e.Message }
func (e *ErrorResponse) GetStatus() int { return e.status }

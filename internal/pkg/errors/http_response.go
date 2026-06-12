package errors

// Body matches internal/pkg/response.ErrorBody for middleware JSON responses.
type Body struct {
	Code      int            `json:"code"`
	Reason    string         `json:"reason"`
	Message   string         `json:"message"`
	ErrorCode string         `json:"error_code,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// ToBody converts the business error to an HTTP error response body.
func (e *BizError) ToBody() Body {
	if e == nil {
		return Body{
			Code: 500, Reason: "InternalError",
			Message: "internal server error", ErrorCode: "50001",
		}
	}
	return Body{
		Code:      e.HTTPStatus(),
		Reason:    e.Reason,
		Message:   e.Message,
		ErrorCode: e.ErrorCode,
		Metadata:  e.Metadata,
	}
}

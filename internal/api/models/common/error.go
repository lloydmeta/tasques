package common

type Headers map[string]string

// Body models errors as JSON in the API
type Body struct {
	Message string `json:"message" binding:"required" example:"Something went wrong :("`
}

type ApiError struct {
	StatusCode int
	Body       Body
}

func (a *ApiError) Error() string {
	return a.Body.Message
}

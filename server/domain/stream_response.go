package domain

type StreamResponse struct {
	Name    string
	Message string
	Error   error
}

func NewStreamResponse(name, message string) StreamResponse {
	return StreamResponse{
		Name:    name,
		Message: message,
	}
}

func NewStreamError(err error) StreamResponse {
	return StreamResponse{
		Error: err,
	}
}

func (r StreamResponse) IsError() bool {
	return r.Error != nil
}

func (r StreamResponse) IsValid() bool {
	if r.IsError() {
		return r.Error != nil
	}
	return r.Name != "" && r.Message != ""
}

func (r StreamResponse) String() string {
	if r.IsError() {
		return "error: " + r.Error.Error()
	}
	return r.Name + ": " + r.Message
}

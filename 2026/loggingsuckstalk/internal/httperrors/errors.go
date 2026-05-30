package httperrors

type (
	// ValidationError is an error to return when a value is not valid.
	ValidationError struct {
		Title   string
		Message string
	}

	// InternalServerError is an error to return when something goes wrong in the server.
	InternalServerError struct {
		Title   string
		Message string
	}
)

func (e ValidationError) Error() string {
	return e.Message
}

func (e InternalServerError) Error() string {
	return e.Message
}

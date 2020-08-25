package lifecycle

// ExecutionIDNotFound custom error
type ExecutionIDNotFound struct {
	Err error
}

func (e ExecutionIDNotFound) Error() string {
	return e.Err.Error()
}

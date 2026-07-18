package lifecycle

import "errors"

// ExecutionIDNotFoundError custom error.
type ExecutionIDNotFoundError struct {
	Err error
}

func (e ExecutionIDNotFoundError) Error() string {
	return e.Err.Error()
}

func (ExecutionIDNotFoundError) Is(err error) bool {
	e2 := &ExecutionIDNotFoundError{}
	return errors.As(err, &e2)
}

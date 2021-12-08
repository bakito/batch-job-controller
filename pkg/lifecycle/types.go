package lifecycle

import "errors"

// ExecutionIDNotFound custom error
type ExecutionIDNotFound struct {
	Err error
}

func (e ExecutionIDNotFound) Error() string {
	return e.Err.Error()
}

func (e ExecutionIDNotFound) Is(err error) bool {
	e2 := &ExecutionIDNotFound{}
	return errors.As(err, &e2)
}

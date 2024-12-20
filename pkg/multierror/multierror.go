package multierror

import (
	"strings"
	"sync"
)

type Error struct {
	errors []error
	mu     sync.Mutex
}

var _ error = (*Error)(nil)

func (e *Error) Add(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.errors == nil {
		e.errors = make([]error, 0)
	}

	e.errors = append(e.errors, err)
}

func (e *Error) None() bool {
	return !e.Any()
}

func (e *Error) Any() bool {
	return len(e.errors) != 0
}

func (e *Error) ErrorOrNil() *Error {
	if len(e.errors) == 0 {
		return nil
	}

	return e
}

func (e *Error) Unwrap() []error {
	return e.errors
}

func (e *Error) Error() string {
	messages := make([]string, len(e.errors))
	for i, err := range e.errors {
		messages[i] = err.Error()
	}

	return strings.Join(messages, "\n")
}

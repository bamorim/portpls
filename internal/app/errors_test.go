package app

import (
	"errors"
	"testing"
)

func TestCodeError(t *testing.T) {
	t.Run("Error returns wrapped message", func(t *testing.T) {
		err := NewCodeError(1, errors.New("test error"))
		if err.Error() != "test error" {
			t.Errorf("Error() = %q, want %q", err.Error(), "test error")
		}
	})

	t.Run("Error returns empty for nil wrapped", func(t *testing.T) {
		err := CodeError{Code: 1, Err: nil}
		if err.Error() != "" {
			t.Errorf("Error() = %q, want empty string", err.Error())
		}
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		inner := errors.New("inner error")
		err := NewCodeError(1, inner).(CodeError)
		if err.Unwrap() != inner {
			t.Error("Unwrap() should return inner error")
		}
	})

	t.Run("errors.Is works with wrapped error", func(t *testing.T) {
		err := NewCodeError(1, ErrNoFreePorts)
		if !errors.Is(err, ErrNoFreePorts) {
			t.Error("errors.Is should find ErrNoFreePorts")
		}
	})

	t.Run("Code is preserved", func(t *testing.T) {
		err := NewCodeError(42, errors.New("test")).(CodeError)
		if err.Code != 42 {
			t.Errorf("Code = %d, want 42", err.Code)
		}
	})
}

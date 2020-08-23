package http

import (
	"unicode"

	"github.com/go-playground/validator/v10"
	corev1 "k8s.io/api/core/v1"
)

// Event to be sent as k8s event
type Event struct {
	Waring  bool     `json:"warning"`
	Reason  string   `json:"reason" validate:"required,first_char_must_be_uppercase"`
	Message string   `json:"message,omitEmpty" validate:"required"`
	Args    []string `json:"args,omitEmpty"`
}

func (e *Event) args() []interface{} {
	var args []interface{}
	for _, a := range e.Args {
		args = append(args, a)
	}
	return args
}

// Validate the event
func (e *Event) Validate() error {
	validate := validator.New()
	_ = validate.RegisterValidation("first_char_must_be_uppercase", firstIsUpper)
	return validate.Struct(e)
}

func (e *Event) Type() string {
	if e.Waring {
		return corev1.EventTypeWarning
	}
	return corev1.EventTypeNormal
}

func firstIsUpper(fl validator.FieldLevel) bool {
	reason := fl.Field()
	return unicode.IsUpper(rune(reason.String()[0]))
}

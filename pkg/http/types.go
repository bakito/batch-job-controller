package http

import (
	"github.com/go-playground/validator/v10"
	"unicode"
)

// Event to be sent as k8s event
type Event struct {
	Eventtype  string   `json:"eventtype" validate:"oneof=Normal Warning"`
	Reason     string   `json:"reason" validate:"required,first_is_upper"`
	Message    string   `json:"message,omitEmpty" validate:"one_message_required=MessageFmt"`
	MessageFmt string   `json:"messageFmt,omitEmpty"`
	Args       []string `json:"args,omitEmpty"`
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
	_ = validate.RegisterValidation("first_is_upper", firstIsUpper)
	_ = validate.RegisterValidation("one_message_required", oneMessageRequired)
	return validate.Struct(e)
}

func firstIsUpper(fl validator.FieldLevel) bool {
	reason := fl.Field()
	return unicode.IsUpper(rune(reason.String()[0]))
}

func oneMessageRequired(fl validator.FieldLevel) bool {
	message := fl.Field()
	messageFmt, _, _, _ := fl.GetStructFieldOK2()
	return message.String() != "" || messageFmt.String() != ""
}

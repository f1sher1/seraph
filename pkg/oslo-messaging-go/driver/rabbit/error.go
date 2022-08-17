package rabbit

import (
	"regexp"
	"seraph/pkg/oslo-messaging-go/message"
	"strconv"

	"github.com/streadway/amqp"
)

var (
	errorRegex, _ = regexp.Compile(`^Exception \((\d+)\) Reason: (.*)`)
)

func NewError(code int, reason string, server bool) *message.Error {
	canRecover := isSoftExceptionCode(code)
	return message.NewError(code, reason, canRecover, server)
}

func parseError(err error) *message.Error {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	if errorRegex.MatchString(errStr) {
		matched := errorRegex.FindStringSubmatch(errStr)
		code, convErr := strconv.Atoi(matched[1])
		if convErr != nil {
			code = amqp.InternalError
		}
		return message.NewError(code, matched[2], isSoftExceptionCode(code), true)
	}
	code := amqp.InternalError
	return message.NewError(code, err.Error(), isSoftExceptionCode(code), true)
}

func isSoftExceptionCode(code int) bool {
	switch code {
	case 311:
		return true
	case 312:
		return true
	case 313:
		return true
	case 403:
		return true
	case 404:
		return true
	case 405:
		return true
	case 406:
		return true

	}
	return false
}

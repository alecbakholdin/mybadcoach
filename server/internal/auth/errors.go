package auth

import (
	"errors"

	"connectrpc.com/connect"

	errorsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/errors/v1"
)

// Stable machine-readable codes attached to Connect errors as
// ErrorDetail.code. The contract is documented in proto/mybadcoach/errors/v1.
const (
	CodeEmailTaken      = "email_taken"
	CodeInvalidLogin    = "invalid_credentials"
	CodeInvalidArgument = "invalid_argument"
	CodeSessionExpired  = "session_expired"
	CodeCSRFRejected    = "csrf_rejected"
)

// newConnectError builds a Connect error carrying the machine-readable
// ErrorDetail, and the field violations when there are any.
func newConnectError(
	code connect.Code,
	machineCode string,
	message string,
	violations []*errorsv1.FieldViolation,
) *connect.Error {
	err := connect.NewError(code, errors.New(message))
	detail, detailErr := connect.NewErrorDetail(&errorsv1.ErrorDetail{
		Code:            machineCode,
		FieldViolations: violations,
	})
	if detailErr == nil {
		err.AddDetail(detail)
	}
	return err
}

func invalidCredentialsError() *connect.Error {
	return newConnectError(
		connect.CodeUnauthenticated,
		CodeInvalidLogin,
		"invalid email or password",
		nil,
	)
}

func sessionExpiredError() *connect.Error {
	return newConnectError(
		connect.CodeUnauthenticated,
		CodeSessionExpired,
		"session expired",
		nil,
	)
}

func csrfRejectedError() *connect.Error {
	return newConnectError(
		connect.CodePermissionDenied,
		CodeCSRFRejected,
		"request rejected by CSRF protection",
		nil,
	)
}

func internalError() *connect.Error {
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}

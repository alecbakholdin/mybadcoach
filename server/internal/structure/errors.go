// Package structure implements the structure CRUD Connect services: teams and
// memberships, seasons and season rosters, events and event rosters, players,
// and matches. Addressing is by nested slug, membership gates reads and
// structural mutations, and destructive operations marked owner-only require
// the owner role.
package structure

import (
	"errors"

	"connectrpc.com/connect"

	errorsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/errors/v1"
)

// Stable machine-readable codes attached to Connect errors as
// ErrorDetail.code. The contract is documented in proto/mybadcoach/errors/v1.
const (
	codeNotFound         = "not_found"
	codeUserNotFound     = "user_not_found"
	codePermissionDenied = "permission_denied"
	codeLastOwner        = "last_owner"
	codeInvalidArgument  = "invalid_argument"
	codeAlreadyExists    = "already_exists"
	codeSessionExpired   = "session_expired"
	codeIDConflict       = "id_conflict"
)

// newError builds a Connect error carrying the machine-readable ErrorDetail,
// and the field violations when there are any.
func newError(
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

func fieldViolation(field, description string) *errorsv1.FieldViolation {
	return &errorsv1.FieldViolation{Field: field, Description: description}
}

func invalidArgument(violations ...*errorsv1.FieldViolation) *connect.Error {
	return newError(
		connect.CodeInvalidArgument,
		codeInvalidArgument,
		"invalid argument",
		violations,
	)
}

func notFound(resource string) *connect.Error {
	return newError(connect.CodeNotFound, codeNotFound, resource+" not found", nil)
}

func userNotFound() *connect.Error {
	return newError(connect.CodeNotFound, codeUserNotFound, "user not found", nil)
}

func permissionDenied() *connect.Error {
	return newError(
		connect.CodePermissionDenied,
		codePermissionDenied,
		"permission denied",
		nil,
	)
}

func lastOwnerError() *connect.Error {
	return newError(
		connect.CodePermissionDenied,
		codeLastOwner,
		"a team must keep at least one owner",
		nil,
	)
}

func alreadyExistsError(message string) *connect.Error {
	return newError(connect.CodeAlreadyExists, codeAlreadyExists, message, nil)
}

// idConflictError rejects an event id whose stored content differs from the
// submitted event. The id is a client-chosen idempotency key, so re-using it
// with different content is a client bug rather than a retry.
func idConflictError(field string) *connect.Error {
	return newError(
		connect.CodeInvalidArgument,
		codeIDConflict,
		"match event id already used with different content",
		[]*errorsv1.FieldViolation{
			fieldViolation(field, "id already used with different content"),
		},
	)
}

func sessionExpiredError() *connect.Error {
	return newError(
		connect.CodeUnauthenticated,
		codeSessionExpired,
		"session expired",
		nil,
	)
}

func internalError() *connect.Error {
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}

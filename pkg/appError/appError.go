package appError

import "net/http"

type AppError interface {
	error
	HTTPStatus() int
	Code() int
}

// our custom error
type appErr struct {
	message    string
	httpStatus int
	code       int
}

func (e appErr) Error() string {
	return e.message
}

func (e appErr) HTTPStatus() int {
	return e.httpStatus
}

func (e appErr) Code() int {
	return e.code
}

// below is default errors with default codes
// the error code is equal to the http status
func BadRequest(text string) AppError {
	return appErr{
		message:    text,
		httpStatus: http.StatusBadRequest,
		code:       400,
	}
}

func Internal() AppError {
	return appErr{
		message:    "internal server error",
		httpStatus: http.StatusInternalServerError,
		code:       500,
	}
}

func NotFound() AppError {
	return appErr{
		message:    "not found",
		httpStatus: http.StatusNotFound,
		code:       404,
	}
}

func Unauthorized() AppError {
	return appErr{
		message:    "not authorized",
		httpStatus: http.StatusUnauthorized,
		code:       401,
	}
}

func MethodNotAllowed() AppError {
	return appErr{
		message:    "method not allowed",
		httpStatus: http.StatusMethodNotAllowed,
		code:       405,
	}
}

func Forbidden() AppError {
	return appErr{
		message:    "Forbidden",
		httpStatus: http.StatusForbidden,
		code:       403,
	}
}

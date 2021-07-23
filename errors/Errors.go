package errors

import "fmt"

var (
	// ErrDBConnectingFailed base error when theres a DB connecting error
	ErrDBConnectingFailed = fmt.Errorf("fail to make database connection")

	//ErrUserContextKeyMissing base error if required user context is missing
	ErrUserContextKeyMissing = fmt.Errorf("user identification not in context")

	// ErrStringDataTooLong base error when required data value is too long for db column to insert
	ErrStringDataTooLong = fmt.Errorf("string data too long")
)

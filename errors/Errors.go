package errors

import "fmt"

var (
	ErrDBConnectingFailed = fmt.Errorf("fail to make database connection")
	ErrUserContextKeyMissing = fmt.Errorf("user identification not in context")

	ErrStringDataTooLong = fmt.Errorf("currency code too long, should not be more than 3 digit")
)

package contextkeys

// ContextKeys is a context key
type ContextKeys string

const (
	// XRequestID is the context key when you want to get the current request id
	XRequestID ContextKeys = "X-REQUEST-ID"

	// UserIDContextKey is the context key to obtain the current user id using the API
	UserIDContextKey ContextKeys = "USER_IDENTIFICATION"
)

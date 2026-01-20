package api

const (
	// Pagination defaults
	DefaultLimit  = 50
	DefaultOffset = 0

	// Query Parameters
	ParamStatus      = "status"
	ParamLimit       = "limit"
	ParamOffset      = "offset"
	ParamDeleteFiles = "deleteFiles"
	ParamQuery       = "q"
	ParamID          = "id"
	ParamRemote      = "remote"

	// Headers
	HeaderContentType = "Content-Type"
	MimeJSON          = "application/json"
)

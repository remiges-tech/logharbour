package wsc

const (
	INVALID_PRIORITY = "invalid_priority"
	D                = "D"
	C                = "C"
	A                = "A"
)

const (
	MsgId_InternalErr     = 1001
	MsgId_Invalid_Request = 1006
)

const (
	ErrCode_Internal       = "internal_err"
	ErrCode_InvalidRequest = "invalid_request"
	ErrCode_InvalidJson    = "invalid_json"
	ErrCode_DatabaseError  = "database_error"
	App                    = "app"
)

var (
	APP, PRI, DAYS, SEARCHAFTERTIMESTAMP, SEARCHAFTERDOCID, CLASS, INSTANCE string = "App", "Pri", "Days", "SearchAfterTimestamp", "SearchAfterDocId", "Class", "Instance"
	Priority                                                                       = []string{"Debug2", "Debug1", "Debug0", "Info", "Warn", "Err", "Crit", "Sec"}
)

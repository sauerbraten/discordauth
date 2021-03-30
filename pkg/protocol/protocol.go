package protocol

// standard master server protocol constants
// exported constants can be sent to the master server
const (
	RegServ = "regserv"
	SuccReg = "succreg"
	FailReg = "failreg"

	AddBan    = "addgban"
	ClearBans = "cleargbans"

	ReqAuth  = "reqauth"
	ChalAuth = "chalauth"
	ConfAuth = "confauth"
	SuccAuth = "succauth"
	FailAuth = "failauth"
)

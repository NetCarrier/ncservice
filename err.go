package ncservice

type Err struct {
	Code int
	Msg  string
}

func (e Err) Error() string {
	return e.Msg
}

var ErrNotFound = Err{404, "not found"}
var ErrPermissionDenied = Err{403, "permission denied"}
var ErrUser = Err{400, "user error"}

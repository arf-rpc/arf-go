package status

import "fmt"

type Status int

func (s Status) Error() string {
	v, ok := statusText[s]
	if !ok {
		return fmt.Sprintf("unknown status: %d", s)
	}
	return v
}

const (
	OK                 Status = 0
	Cancelled          Status = 1
	Unknown            Status = 2
	InvalidArgument    Status = 3
	DeadlineExceeded   Status = 4
	NotFound           Status = 5
	AlreadyExists      Status = 6
	PermissionDenied   Status = 7
	ResourceExhausted  Status = 8
	FailedPrecondition Status = 9
	Aborted            Status = 10
	OutOfRange         Status = 11
	Unimplemented      Status = 12
	InternalError      Status = 13
	Unavailable        Status = 14
	DataLoss           Status = 15
	Unauthenticated    Status = 16
)

var statusText = map[Status]string{
	OK:                 "OK",
	Cancelled:          "Cancelled",
	Unknown:            "Unknown",
	InvalidArgument:    "Invalid Argument",
	DeadlineExceeded:   "Deadline Exceeded",
	NotFound:           "Not Found",
	AlreadyExists:      "Already Exists",
	PermissionDenied:   "Permission Denied",
	ResourceExhausted:  "Resource Exhausted",
	FailedPrecondition: "Failed Precondition",
	Aborted:            "Aborted",
	OutOfRange:         "Out of Range",
	Unimplemented:      "Unimplemented",
	InternalError:      "Internal Error",
	Unavailable:        "Unavailable",
	DataLoss:           "Data Loss",
	Unauthenticated:    "Unauthenticated",
}

type BadStatus struct {
	Code    Status
	Message string
}

func (b *BadStatus) Error() string {
	return fmt.Sprintf("BadStatus: %d (%s): %s", int(b.Code), b.Code.Error(), b.Message)
}

func Error(code Status, msg string) error {
	return &BadStatus{code, msg}
}

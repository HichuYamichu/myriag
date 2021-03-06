package errors

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
)

// Error is the type that implements the error interface.
type Error struct {
	Op   Op
	Kind Kind
	Err  error
}

// pad appends str to the buffer if the buffer already has some data.
func pad(b *bytes.Buffer, str string) {
	if b.Len() == 0 {
		return
	}
	b.WriteString(str)
}

var Separator = "::"

func (e *Error) isZero() bool {
	return e.Kind == 0 && e.Err == nil
}

func (e *Error) Error() string {
	b := new(bytes.Buffer)
	if e.Op != "" {
		pad(b, ": ")
		b.WriteString(string(e.Op))
	}
	if e.Kind != 0 {
		pad(b, ": ")
		b.WriteString(e.Kind.String())
	}
	if e.Err != nil {
		if prevErr, ok := e.Err.(*Error); ok {
			if !prevErr.isZero() {
				pad(b, Separator)
				b.WriteString(e.Err.Error())
			}
		} else {
			pad(b, ": ")
			b.WriteString(e.Err.Error())
		}
	}
	if b.Len() == 0 {
		return "no error"
	}
	return b.String()
}

// Op operation
type Op string

// Kind kind of error.
type Kind uint8

// Kinds of errors.
const (
	Other            Kind = iota // Unclassified error.
	Invalid                      // Invalid operation for this type of item.
	IO                           // External I/O error such as network failure.
	Internal                     // Internal error or inconsistency.
	EvalTimeout                  // Evaluation timed out.
	LanguageNotFound             // Language not found.
)

func (k Kind) String() string {
	switch k {
	case Other:
		return "other error"
	case Invalid:
		return "invalid operation"
	case IO:
		return "I/O error"
	case Internal:
		return "internal error"
	case EvalTimeout:
		return "evaluation timed out"
	case LanguageNotFound:
		return "language not found"
	}
	return "unknown error kind"
}

// HTTPStatus transforms error kind to HTTP status code
func (k Kind) HTTPStatus() int {
	switch k {
	case Other:
		return 500
	case Invalid:
		return 400
	case IO:
		return 500
	case Internal:
		return 500
	case EvalTimeout:
		return 513
	case LanguageNotFound:
		return 404
	}
	return 500
}

// E builds an error value from its arguments.
func E(args ...interface{}) error {
	if len(args) == 0 {
		panic("call to errors.E with no arguments")
	}
	e := &Error{}
	for _, arg := range args {
		switch arg := arg.(type) {
		case Op:
			e.Op = arg
		case string:
			e.Err = Str(arg)
		case Kind:
			e.Kind = arg
		case *Error:
			copy := *arg
			e.Err = &copy
		case error:
			e.Err = arg
		default:
			_, file, line, _ := runtime.Caller(1)
			log.Printf("errors.E: bad call from %s:%d: %v", file, line, args)
			return Errorf("unknown type %T, value %v in error call", arg, arg)
		}
	}

	prev, ok := e.Err.(*Error)
	if !ok {
		return e
	}

	if prev.Kind == e.Kind {
		prev.Kind = Other
	}

	if e.Kind == Other {
		e.Kind = prev.Kind
		prev.Kind = Other
	}
	return e
}

type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}

// Errorf is equivalent to fmt.Errorf, but allows clients to import only this
// package for all error handling.
func Errorf(format string, args ...interface{}) error {
	return &errorString{fmt.Sprintf(format, args...)}
}

func Is(err error, kind Kind) bool {
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	if e.Kind != Other {
		return e.Kind == kind
	}
	if e.Err != nil {
		return Is(e.Err, kind)
	}
	return false
}

func Str(text string) error {
	return &errorString{text}
}

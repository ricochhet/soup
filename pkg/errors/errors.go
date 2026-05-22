package errors

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

func New(s any, format string, a ...any) error { return Error(Name(s), fmt.Errorf(format, a...)) }
func Newf(format string, a ...any) error       { return Error("", fmt.Errorf(format, a...)) }
func Error(format string, err error) error {
	if err == nil {
		return nil
	}

	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return err
	}

	if format == "" {
		return fmt.Errorf("%s@L%d: %w", file, line, err)
	}

	return fmt.Errorf("%s@L%d: "+format+": %w", file, line, err)
}

func Name(s any) string {
	pc := reflect.ValueOf(s).Pointer()
	full := runtime.FuncForPC(pc).Name()

	if i := strings.LastIndex(full, "/"); i >= 0 {
		return full[i+1:]
	}

	return strings.TrimSuffix(full, "-fm")
}

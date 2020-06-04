package misc

import "fmt"

type OutputNotPresentError struct {
	errorMsg string
}

func (e *OutputNotPresentError) Error() string {
	return fmt.Sprintf(e.errorMsg)
}

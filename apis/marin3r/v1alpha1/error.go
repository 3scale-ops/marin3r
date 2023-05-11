package v1alpha1

import (
	"encoding/json"
	"strings"
)

// +kubebuilder:object:generate:=false
type MultiError struct {
	Errors ErrorList `json:"errors"`
}

func (ve MultiError) Error() string {
	b, _ := json.Marshal(ve)
	return string(b)
}

// +kubebuilder:object:generate:=false
type ErrorList []error

func (el ErrorList) MarshalJSON() ([]byte, error) {
	marshalledList := []string{}
	for _, e := range el {
		jsonValue, err := json.Marshal(e.Error())
		if err != nil {
			return nil, err
		}
		marshalledList = append(marshalledList, string(jsonValue))
	}
	return []byte("[" + strings.Join(marshalledList, ",") + "]"), nil
}

func NewMultiError(e []error) MultiError {
	return MultiError{Errors: e}
}

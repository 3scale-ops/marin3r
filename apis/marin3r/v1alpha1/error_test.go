package v1alpha1

import (
	"fmt"
	"testing"
)

func TestMultiError_Error(t *testing.T) {
	type fields struct {
		List []error
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "Returns a json string with errors",
			fields: fields{List: []error{fmt.Errorf("error1"), fmt.Errorf("error2")}},
			want:   `{"errors":["error1","error2"]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ve := MultiError{
				Errors: tt.fields.List,
			}
			if got := ve.Error(); got != tt.want {
				t.Errorf("MultiError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

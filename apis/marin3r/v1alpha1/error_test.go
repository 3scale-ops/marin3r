package v1alpha1

import (
	"fmt"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
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
			want:   `{"validationErrors":["error1","error2"]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ve := ValidationError{
				Errors: tt.fields.List,
			}
			if got := ve.Error(); got != tt.want {
				t.Errorf("ValidationError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

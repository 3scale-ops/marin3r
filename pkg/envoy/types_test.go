package envoy

import "testing"

func TestAPIVersion_String(t *testing.T) {
	tests := []struct {
		name    string
		version APIVersion
		want    string
	}{
		{"Returns v3", APIv3, "v3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.String(); got != tt.want {
				t.Errorf("APIVersion.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseAPIVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    APIVersion
		wantErr bool
	}{
		{"Returns APIv3", "v3", APIv3, false},
		{"Returns error", "xx", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAPIVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAPIVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseAPIVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

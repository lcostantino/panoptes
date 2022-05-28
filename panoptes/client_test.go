//go:build windows
// +build windows

package panoptes

import "testing"

func newProvider(guid string) Provider {
	return Provider{Guid: guid}
}

func TestClient_AddProvider(t *testing.T) {

	type args struct {
		prov Provider
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"New Provider fails due to wrong GUID", args{prov: newProvider("{111-222}")}, true},
		{"New Provider added", args{prov: newProvider("{1C95126E-7EEA-49A9-A3FE-A378B03DDB4D}")}, true},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient()
			if err := c.AddProvider(tt.args.prov); (err != nil) != tt.wantErr {
				t.Errorf("Client.AddProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

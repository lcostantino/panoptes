//go:build windows
// +build windows

package panoptes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func newProvider(guid string, name string) Provider {
	return Provider{Guid: guid, Name: name}
}

func TestClient_AddProvider(t *testing.T) {

	type args struct {
		prov Provider
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantErrStr string
	}{
		{"New Provider fails due to wrong GUID", args{prov: newProvider("{111-222}", "Prov")}, true, ""},
		{"New Provider added", args{prov: newProvider("{1C95126E-7EEA-49A9-A3FE-A378B03DDB4D}", "Prov")}, false, ""},
		{"Fail to add provider due to empty name", args{prov: newProvider("{1C95126E-7EEA-49A9-A3FE-A378B03DDB4D}", "")}, true, "Empty provider name"},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient()
			if err := c.AddProvider(tt.args.prov); err != nil {
				if tt.wantErr == false {
					t.Errorf("Client.AddProvider() error = %v, wantErr %v", err, tt.wantErr)
				} else if tt.wantErrStr != "" && tt.wantErrStr != err.Error() {
					t.Errorf("Client.AddProvider() error = %v, wantErrStr: %v", err, tt.wantErrStr)
				}
			}

		})
	}
}

func TestClient_AddProviderDuplicate(t *testing.T) {

	provider := newProvider("{1C95126E-7EEA-49A9-A3FE-A378B03DDB4D}", "Prov")
	provider2 := provider

	c := NewClient()
	if err := c.AddProvider(provider); err != nil {
		t.Errorf("Client.AddProvider() error = %v", err)

	}
	if err := c.AddProvider(provider2); err == nil {
		t.Errorf("Client.AddProvider(2) - Expected error")
	} else {
		assert.Equal(t, err.Error(), "A provider with the same name is already registered")
	}
}

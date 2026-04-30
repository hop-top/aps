package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfileURI(t *testing.T) {
	p := &Profile{ID: "noor"}
	assert.Equal(t, "aps://profile/noor", p.URI())
}

func TestProfileURI_Empty(t *testing.T) {
	p := &Profile{}
	assert.Equal(t, "aps://profile/", p.URI())
}

func TestParseProfileRef(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "bare id", input: "noor", want: "noor"},
		{name: "full uri", input: "aps://profile/noor", want: "noor"},
		{name: "wrong scheme", input: "tlc://profile/noor", wantErr: true},
		{name: "wrong space", input: "aps://workspace/foo", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseProfileRef(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProfileURI_RoundTrip(t *testing.T) {
	original := &Profile{ID: "sami"}
	id, err := ParseProfileRef(original.URI())
	assert.NoError(t, err)
	assert.Equal(t, original.ID, id)
}

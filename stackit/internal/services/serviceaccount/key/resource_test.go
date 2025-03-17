package key

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func int64Ptr(i int64) *int64 {
	return &i
}

func TestComputeValidUntil(t *testing.T) {
	tests := []struct {
		name        string
		ttlDays     *int64
		expectError bool
	}{
		{
			name:        "ttlDays is nil",
			ttlDays:     nil,
			expectError: true,
		},
		{
			name:        "ttlDays is 10",
			ttlDays:     int64Ptr(10),
			expectError: false,
		},
		{
			name:        "ttlDays is 0",
			ttlDays:     int64Ptr(0),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultTime, err := computeValidUntil(tt.ttlDays)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				expectedTime := time.Now().UTC().Add(time.Duration(*tt.ttlDays) * 24 * time.Hour)
				// Assert the result is within a reasonable tolerance range
				assert.WithinDuration(t, expectedTime, resultTime, 1*time.Second)
			}
		})
	}
}

func TestMapReadResponse(t *testing.T) {
	tests := []struct {
		name        string
		resp        *serviceaccount.GetServiceAccountKeyResponse
		model       *Model
		expectError bool
	}{
		{
			name: "Valid input",
			resp: &serviceaccount.GetServiceAccountKeyResponse{
				Id:           strPtr("key-123"),
				KeyAlgorithm: strPtr("RSA_2048"),
				KeyOrigin:    strPtr("USER_PROVIDED"),
				KeyType:      strPtr("USER_MANAGED"),
				PublicKey:    strPtr("test-public-key"),
				Credentials: &serviceaccount.GetServiceAccountKeyResponseCredentials{
					Aud: strPtr("audience"),
					Kid: strPtr("kid"),
					Iss: strPtr("issuer"),
					Sub: strPtr("subject"),
				},
				CreatedAt:  timePtr(time.Now().Add(-time.Hour)),
				ValidUntil: timePtr(time.Now().Add(time.Hour * 24 * 365)),
			},
			model:       &Model{},
			expectError: false,
		},
		{
			name:        "Nil response and model",
			resp:        nil,
			model:       nil,
			expectError: true,
		},
		{
			name: "Missing service account key id",
			resp: &serviceaccount.GetServiceAccountKeyResponse{
				KeyAlgorithm: strPtr("RSA_2048"),
				KeyOrigin:    strPtr("USER_PROVIDED"),
				KeyType:      strPtr("USER_MANAGED"),
				PublicKey:    strPtr("test-public-key"),
				Credentials: &serviceaccount.GetServiceAccountKeyResponseCredentials{
					Aud: strPtr("audience"),
					Kid: strPtr("kid"),
					Iss: strPtr("issuer"),
					Sub: strPtr("subject"),
				},
				CreatedAt:  timePtr(time.Now().Add(-time.Hour)),
				ValidUntil: timePtr(time.Now().Add(time.Hour * 24 * 365)),
			},
			model:       &Model{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapReadResponse(tt.resp, tt.model)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				assert.Equal(t, *tt.resp.KeyAlgorithm, tt.model.KeyAlgorithm.ValueString())
				assert.Equal(t, *tt.resp.KeyOrigin, tt.model.KeyOrigin.ValueString())
				assert.Equal(t, *tt.resp.KeyType, tt.model.KeyType.ValueString())
				assert.Equal(t, *tt.resp.PublicKey, tt.model.PublicKey.ValueString())

				if tt.resp.Credentials != nil {
					assert.Equal(t, *tt.resp.Credentials.Aud, tt.model.Audience.ValueString())
					assert.Equal(t, *tt.resp.Credentials.Kid, tt.model.Kid.ValueString())
					assert.Equal(t, *tt.resp.Credentials.Iss, tt.model.Issuer.ValueString())
					assert.Equal(t, *tt.resp.Credentials.Sub, tt.model.Subject.ValueString())
				}
			}
		})
	}
}

func TestMapCreateResponse(t *testing.T) {
	tests := []struct {
		name        string
		resp        *serviceaccount.CreateServiceAccountKeyResponse
		model       *Model
		expectError bool
	}{
		{
			name: "Valid input",
			resp: &serviceaccount.CreateServiceAccountKeyResponse{
				Id:           strPtr("key-123"),
				KeyAlgorithm: strPtr("RSA_2048"),
				KeyOrigin:    strPtr("USER_PROVIDED"),
				KeyType:      strPtr("USER_MANAGED"),
				PublicKey:    strPtr("test-public-key"),
				Credentials: &serviceaccount.CreateServiceAccountKeyResponseCredentials{
					PrivateKey: strPtr("private-key-value"),
					Aud:        strPtr("audience"),
					Kid:        strPtr("kid"),
					Iss:        strPtr("issuer"),
					Sub:        strPtr("subject"),
				},
				CreatedAt:  timePtr(time.Now().Add(-time.Hour)),
				ValidUntil: timePtr(time.Now().Add(time.Hour * 24 * 365)),
			},
			model:       &Model{},
			expectError: false,
		},
		{
			name:        "Nil response",
			resp:        nil,
			model:       &Model{},
			expectError: true,
		},
		{
			name: "Missing private key",
			resp: &serviceaccount.CreateServiceAccountKeyResponse{
				Id:           strPtr("key-123"),
				KeyAlgorithm: strPtr("RSA_2048"),
				KeyOrigin:    strPtr("USER_PROVIDED"),
				KeyType:      strPtr("USER_MANAGED"),
				PublicKey:    strPtr("test-public-key"),
				Credentials: &serviceaccount.CreateServiceAccountKeyResponseCredentials{
					Aud: strPtr("audience"),
					Kid: strPtr("kid"),
					Iss: strPtr("issuer"),
					Sub: strPtr("subject"),
				},
				CreatedAt:  timePtr(time.Now().Add(-time.Hour)),
				ValidUntil: timePtr(time.Now().Add(time.Hour * 24 * 365)),
			},
			model:       &Model{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapCreateResponse(tt.resp, tt.model)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, tt.model.RawResponse)

				// Verify the raw JSON representation has some content
				var saKeyResp serviceaccount.CreateServiceAccountKeyResponse
				err := json.Unmarshal([]byte(tt.model.RawResponse.ValueString()), &saKeyResp)
				assert.NoError(t, err)
				assert.NotEmpty(t, saKeyResp)
			}
		})
	}
}

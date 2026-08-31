package auth

import ("testing"
		"time"
		"github.com/google/uuid")


func TestValidateJWT(t *testing.T) {
	validID := uuid.New()
	secret := "super-secret-key"
	
	tests := []struct {
		name         string
		userID       uuid.UUID
		signSecret   string
		valSecret    string
		expiresIn    time.Duration
		wantErr      bool
	}{
			{
				name: "Valid Token",
				userID: validID,
				signSecret: secret,
				valSecret: secret,
				expiresIn: time.Hour,
				wantErr: false,
			},
			{	
				name: "Expired Token",
				userID: validID,
				signSecret: secret,
				valSecret: secret,
				expiresIn: -time.Hour,
				wantErr: true,
			},
			{
				name: "Wrong secret Key",
				userID: validID,
				signSecret: secret,
				valSecret: "omae wa mou shinderiu",
				expiresIn: time.Hour,
				wantErr: true,
			},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString, err := MakeJWT(tt.userID, tt.signSecret, tt.expiresIn)
			if err != nil {
				t.Fatalf("failed to make JWT: %v", err)
			}
		
		extractedID, err := validateJWT(tokenString, tt.valSecret)
		
		// checking error behavior
		if tt.wantErr {
			if err == nil {
				t.Errorf("expected error, got nil")
			}
			return
		}

		// checking success behaivor
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if extractedID != tt.userID {
			t.Errorf("got userID: %v, want %v", extractedID, tt.userID)
		}
	})
	}
}
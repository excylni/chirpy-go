package auth

import ("testing"
		"net/http")

func TestGetBearerToken(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer super-secret-token")

	token, err := GetBearerToken(headers)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if token != "super-secret-token" {
		t.Errorf("got %s, want super-secret-token", token)
	}
}
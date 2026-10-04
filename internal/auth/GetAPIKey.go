package auth

import (
		"net/http"
		"strings"
		"fmt"
	)

func GetAPIKey (headers http.Header) (string,error) {
	apiKey := headers.Get("Authorization")
	if apiKey == "" {
		return "", fmt.Errorf("missing API Key")
	}

		const prefix = "ApiKey "
	if !strings.HasPrefix(apiKey, prefix) {
		return "", fmt.Errorf("invalid API Key")
	}

		apiString := strings.TrimSpace(strings.TrimPrefix(apiKey, prefix))
	if apiString == "" {
		return "", fmt.Errorf("empty API Key")
	}
	return apiString, nil
}
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/excylni/chirpy-go/internal/database"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func TestHandleUpgradeUser(t *testing.T) {
	// 1. DB_URL laden
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://localhost:5432/chirpy?sslmode=disable"
	}

	// 2. DB öffnen & Queries initialisieren (Verhindert nil-pointer panic)
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	cfg := &apiConfig{
		dataBaseQueries: dbQueries,
	}

	ctx := t.Context()

	// 3. Dynamische E-Mail generieren (Verhindert duplicate key error 23505)
	testEmail := fmt.Sprintf("test_upgrade_%s@example.com", uuid.New().String())

	testUser, err := dbQueries.CreateUser(ctx, database.CreateUserParams{
		Email:          testEmail,
		HashedPassword: "hashedpassword123",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	validUserID := testUser.ID
	unknownUserID := uuid.New()

	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Valid Upgrade Request",
			body: map[string]interface{}{
				"event": "user.upgraded",
				"data": map[string]interface{}{
					"user_id": validUserID.String(),
				},
			},
			expectedStatus: http.StatusNoContent, // 204
		},
		{
			name: "Ignored Event Type",
			body: map[string]interface{}{
				"event": "user.deleted",
				"data": map[string]interface{}{
					"user_id": validUserID.String(),
				},
			},
			expectedStatus: http.StatusNoContent, // 204
		},
		{
			name: "User Not Found in DB",
			body: map[string]interface{}{
				"event": "user.upgraded",
				"data": map[string]interface{}{
					"user_id": unknownUserID.String(),
				},
			},
			expectedStatus: http.StatusNotFound, // 404
		},
		{
			name: "Invalid JSON Body",
			body: map[string]interface{}{
				"event": "user.upgraded",
				"data":  "invalid-data-structure",
			},
			expectedStatus: http.StatusBadRequest, // 400
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, err := json.Marshal(tt.body)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/polka/webhooks", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			handler := http.HandlerFunc(cfg.handleUpgradeUser)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					rr.Code, tt.expectedStatus)
			}
		})
	}
}
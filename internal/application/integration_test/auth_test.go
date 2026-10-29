//go:build integration

package integration_test

import (
	"net/http"
	"testing"
)

func TestLogin_Success(t *testing.T) {
	status, body := doJSON(t, http.MethodPost, "/auth/v1/login", "", map[string]string{
		"username": testUsername,
		"password": testPassword,
	})
	if status != http.StatusOK {
		t.Fatalf("status %d, body %s", status, body)
	}

	response := decodeJSON[struct {
		Token string `json:"token"`
	}](t, body)
	if response.Token == "" {
		t.Fatal("expected a token")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	status, body := doJSON(t, http.MethodPost, "/auth/v1/login", "", map[string]string{
		"username": testUsername,
		"password": "wrong-password",
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("status %d, body %s", status, body)
	}

	got := decodeJSON[apiError](t, body)
	if got.Code != http.StatusUnauthorized {
		t.Fatalf("error code %d", got.Code)
	}
}

func TestCreateCompany_UnauthorizedWithoutToken(t *testing.T) {
	status, body := doJSON(t, http.MethodPost, "/company/v1/", "", map[string]any{
		"name":                "NoAuth",
		"type":                "Corporations",
		"amount_of_employees": 1,
		"registered":          true,
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("status %d, body %s", status, body)
	}
}

func TestCreateCompany_UnauthorizedWithMalformedHeader(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, httpServer.URL+"/company/v1/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Token abc")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

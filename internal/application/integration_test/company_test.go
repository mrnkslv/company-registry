//go:build integration

package integration_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestCompanyLifecycle(t *testing.T) {
	truncateCompanies(t)

	token := login(t)
	description := "Test company"

	status, body := doJSON(t, http.MethodPost, "/company/v1/", token, map[string]any{
		"name":                "Acme",
		"description":         description,
		"type":                "Corporations",
		"amount_of_employees": 10,
		"registered":          true,
	})
	if status != http.StatusCreated {
		t.Fatalf("create: status %d, body %s", status, body)
	}

	created := decodeJSON[companyResponse](t, body)
	if created.ID == "" {
		t.Fatal("create: empty id")
	}
	if created.Name != "Acme" || created.Type != "Corporations" || created.AmountOfEmployees != 10 || !created.Registered {
		t.Fatalf("create: unexpected company %+v", created)
	}
	if created.Description == nil || *created.Description != description {
		t.Fatalf("create: unexpected description %+v", created.Description)
	}

	status, body = doJSON(t, http.MethodGet, "/company/v1/"+created.ID, "", nil)
	if status != http.StatusOK {
		t.Fatalf("get: status %d, body %s", status, body)
	}

	got := decodeJSON[companyResponse](t, body)
	if got.ID != created.ID || got.Name != "Acme" {
		t.Fatalf("get: unexpected company %+v", got)
	}

	employees := int32(25)
	registered := false
	status, body = doJSON(t, http.MethodPatch, "/company/v1/"+created.ID, token, map[string]any{
		"amount_of_employees": employees,
		"registered":          registered,
	})
	if status != http.StatusOK {
		t.Fatalf("patch: status %d, body %s", status, body)
	}

	patched := decodeJSON[companyResponse](t, body)
	if patched.Name != "Acme" {
		t.Fatalf("patch: name changed to %q", patched.Name)
	}
	if patched.AmountOfEmployees != employees || patched.Registered {
		t.Fatalf("patch: unexpected fields %+v", patched)
	}

	status, body = doJSON(t, http.MethodDelete, "/company/v1/"+created.ID, token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete: status %d, body %s", status, body)
	}

	status, body = doJSON(t, http.MethodGet, "/company/v1/"+created.ID, "", nil)
	if status != http.StatusNotFound {
		t.Fatalf("get after delete: status %d, body %s", status, body)
	}
}

func TestCreateCompany_DuplicateName(t *testing.T) {
	truncateCompanies(t)

	token := login(t)
	payload := map[string]any{
		"name":                "DupName",
		"type":                "NonProfit",
		"amount_of_employees": 3,
		"registered":          true,
	}

	status, body := doJSON(t, http.MethodPost, "/company/v1/", token, payload)
	if status != http.StatusCreated {
		t.Fatalf("first create: status %d, body %s", status, body)
	}

	status, body = doJSON(t, http.MethodPost, "/company/v1/", token, payload)
	if status != http.StatusConflict {
		t.Fatalf("second create: status %d, body %s", status, body)
	}
}

func TestCreateCompany_ValidationError(t *testing.T) {
	token := login(t)

	status, body := doJSON(t, http.MethodPost, "/company/v1/", token, map[string]any{
		"name":                "TooLongCompanyName",
		"type":                "Corporations",
		"amount_of_employees": 1,
		"registered":          true,
	})
	if status != http.StatusBadRequest {
		t.Fatalf("long name: status %d, body %s", status, body)
	}

	status, body = doJSON(t, http.MethodPost, "/company/v1/", token, map[string]any{
		"name":                "BadType",
		"type":                "LLC",
		"amount_of_employees": 1,
		"registered":          true,
	})
	if status != http.StatusBadRequest {
		t.Fatalf("bad type: status %d, body %s", status, body)
	}
}

func TestGetCompany_NotFoundAndInvalidID(t *testing.T) {
	status, body := doJSON(t, http.MethodGet, "/company/v1/"+uuid.NewString(), "", nil)
	if status != http.StatusNotFound {
		t.Fatalf("unknown id: status %d, body %s", status, body)
	}

	status, body = doJSON(t, http.MethodGet, "/company/v1/not-a-uuid", "", nil)
	if status != http.StatusBadRequest {
		t.Fatalf("invalid id: status %d, body %s", status, body)
	}
}

func TestPatchAndDelete_NotFound(t *testing.T) {
	token := login(t)
	missingID := uuid.NewString()

	status, body := doJSON(t, http.MethodPatch, "/company/v1/"+missingID, token, map[string]any{
		"name": "Nope",
	})
	if status != http.StatusNotFound {
		t.Fatalf("patch missing: status %d, body %s", status, body)
	}

	status, body = doJSON(t, http.MethodDelete, "/company/v1/"+missingID, token, nil)
	if status != http.StatusNotFound {
		t.Fatalf("delete missing: status %d, body %s", status, body)
	}
}

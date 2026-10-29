//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Lifecycle and basic cases
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var nameSeq atomic.Int64

// uniqueName returns a company name that is unique within the test run.
// The prefix must be short enough for the result to fit into 15 characters.
func uniqueName(prefix string) string {
	return fmt.Sprintf("%s%d", prefix, nameSeq.Add(1))
}

func validCompany(name string) map[string]any {
	return map[string]any{
		"name":                name,
		"description":         "Integration test company",
		"type":                "Corporations",
		"amount_of_employees": 10,
		"registered":          true,
	}
}

// createCompany creates a company and fails the test unless the API returns 201.
func createCompany(t *testing.T, token string, payload map[string]any) companyResponse {
	t.Helper()

	status, body := doJSON(t, http.MethodPost, "/company/v1/", token, payload)
	if status != http.StatusCreated {
		t.Fatalf("create: status %d, body %s", status, body)
	}

	return decodeJSON[companyResponse](t, body)
}

func getCompany(t *testing.T, id string) companyResponse {
	t.Helper()

	status, body := doJSON(t, http.MethodGet, "/company/v1/"+id, "", nil)
	if status != http.StatusOK {
		t.Fatalf("get: status %d, body %s", status, body)
	}

	return decodeJSON[companyResponse](t, body)
}

type rawResponse struct {
	status int
	header http.Header
	body   []byte
}

// doRaw sends an arbitrary body (including malformed JSON) and returns the full response.
// It returns an error instead of failing the test so it can be used from goroutines.
func doRaw(method, path, token, body string) (rawResponse, error) {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, httpServer.URL+path, reader)
	if err != nil {
		return rawResponse{}, fmt.Errorf("http.NewRequest: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return rawResponse{}, fmt.Errorf("http.Do: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return rawResponse{}, fmt.Errorf("io.ReadAll: %w", err)
	}

	return rawResponse{status: resp.StatusCode, header: resp.Header, body: bytes.TrimSpace(raw)}, nil
}

func mustRaw(t *testing.T, method, path, token, body string) rawResponse {
	t.Helper()

	resp, err := doRaw(method, path, token, body)
	if err != nil {
		t.Fatal(err)
	}

	return resp
}

type companyRow struct {
	Name              string  `db:"name"`
	Description       *string `db:"description"`
	Type              string  `db:"type"`
	AmountOfEmployees int32   `db:"amount_of_employees"`
	Registered        bool    `db:"registered"`
}

// companyFromDB reads the row directly, bypassing the API.
func companyFromDB(t *testing.T, id string) (companyRow, bool) {
	t.Helper()

	var rows []companyRow
	err := db.Select(&rows, `
		SELECT name, description, type::text AS type, amount_of_employees, registered
		FROM companies
		WHERE id = $1`, id)
	if err != nil {
		t.Fatalf("select company: %v", err)
	}

	if len(rows) == 0 {
		return companyRow{}, false
	}

	return rows[0], true
}

func countCompanies(t *testing.T) int {
	t.Helper()

	var n int
	if err := db.Get(&n, `SELECT count(*) FROM companies`); err != nil {
		t.Fatalf("count companies: %v", err)
	}

	return n
}

func assertStatus(t *testing.T, got, want int, body []byte) {
	t.Helper()

	if got != want {
		t.Fatalf("status %d, want %d, body %s", got, want, body)
	}
}

func assertAPIError(t *testing.T, body []byte, wantCode int) {
	t.Helper()

	got := decodeJSON[apiError](t, body)
	if got.Code != wantCode || got.Type == "" || got.Message == "" {
		t.Fatalf("unexpected error body %s", body)
	}
}

// ---------------------------------------------------------------------------
// Authentication: tokens and headers
// ---------------------------------------------------------------------------

func signToken(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()

	signed, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return signed
}

func TestLogin_UnknownUser(t *testing.T) {
	status, body := doJSON(t, http.MethodPost, "/auth/v1/login", "", map[string]string{
		"username": "no-such-user",
		"password": testPassword,
	})
	assertStatus(t, status, http.StatusUnauthorized, body)
	assertAPIError(t, body, http.StatusUnauthorized)
}

func TestLogin_MalformedBody(t *testing.T) {
	for name, body := range map[string]string{
		"invalid json": `{"username":`,
		"empty body":   "",
		"wrong types":  `{"username": 1, "password": true}`,
	} {
		t.Run(name, func(t *testing.T) {
			resp := mustRaw(t, http.MethodPost, "/auth/v1/login", "", body)
			assertStatus(t, resp.status, http.StatusBadRequest, resp.body)
			assertAPIError(t, resp.body, http.StatusBadRequest)
		})
	}
}

func TestLogin_TokenWorksForProtectedEndpoint(t *testing.T) {
	token := login(t)

	created := createCompany(t, token, validCompany(uniqueName("Tok")))
	if created.ID == "" {
		t.Fatal("empty id")
	}
}

// Every mutating endpoint must reject requests without a valid token and must not touch the data.
func TestMutatingEndpoints_RejectInvalidTokens(t *testing.T) {
	truncateCompanies(t)

	valid := login(t)
	company := createCompany(t, valid, validCompany(uniqueName("Prot")))

	now := time.Now()
	key := []byte(testSigningKey)

	tokens := map[string]string{
		"no token":     "",
		"garbage":      "not-a-jwt",
		"expired":      signToken(t, jwt.SigningMethodHS256, key, jwt.MapClaims{"user_id": 1, "exp": now.Add(-time.Minute).Unix()}),
		"no exp claim": signToken(t, jwt.SigningMethodHS256, key, jwt.MapClaims{"user_id": 1}),
		"wrong key":    signToken(t, jwt.SigningMethodHS256, []byte("other-key"), jwt.MapClaims{"user_id": 1, "exp": now.Add(time.Hour).Unix()}),
		"HS512 alg":    signToken(t, jwt.SigningMethodHS512, key, jwt.MapClaims{"user_id": 1, "exp": now.Add(time.Hour).Unix()}),
		"alg none":     signToken(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, jwt.MapClaims{"user_id": 1, "exp": now.Add(time.Hour).Unix()}),
		"tampered":     valid[:len(valid)-2] + "xx",
	}

	requests := []struct {
		name, method, path, body string
	}{
		{"create", http.MethodPost, "/company/v1/", `{"name":"Hacker","type":"Corporations","amount_of_employees":1,"registered":true}`},
		{"patch", http.MethodPatch, "/company/v1/" + company.ID, `{"name":"Hacked"}`},
		{"delete", http.MethodDelete, "/company/v1/" + company.ID, ""},
	}

	for tokenName, token := range tokens {
		for _, req := range requests {
			t.Run(tokenName+"/"+req.name, func(t *testing.T) {
				resp := mustRaw(t, req.method, req.path, token, req.body)
				assertStatus(t, resp.status, http.StatusUnauthorized, resp.body)
				assertAPIError(t, resp.body, http.StatusUnauthorized)
			})
		}
	}

	if n := countCompanies(t); n != 1 {
		t.Fatalf("expected 1 company after rejected requests, got %d", n)
	}

	got := getCompany(t, company.ID)
	if got.Name != company.Name {
		t.Fatalf("company was modified by an unauthorized request: %+v", got)
	}
}

func TestAuthHeader_Formats(t *testing.T) {
	token := login(t)

	for name, header := range map[string]string{
		"empty bearer":     "Bearer ",
		"lowercase scheme": "bearer " + token,
		"basic scheme":     "Basic dXNlcjpwYXNz",
		"token only":       token,
	} {
		t.Run(name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodDelete, httpServer.URL+"/company/v1/00000000-0000-0000-0000-000000000000", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", header)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status %d, want 401", resp.StatusCode)
			}
		})
	}
}

func TestGetCompany_IsPublic(t *testing.T) {
	token := login(t)
	created := createCompany(t, token, validCompany(uniqueName("Pub")))

	resp := mustRaw(t, http.MethodGet, "/company/v1/"+created.ID, "", "")
	assertStatus(t, resp.status, http.StatusOK, resp.body)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCreateCompany_PersistsAllFields(t *testing.T) {
	token := login(t)
	name := uniqueName("Full")

	created := createCompany(t, token, map[string]any{
		"name":                name,
		"description":         "Persisted description",
		"type":                "Cooperative",
		"amount_of_employees": 42,
		"registered":          false,
	})

	if _, err := uuid.Parse(created.ID); err != nil {
		t.Fatalf("id is not a uuid: %q", created.ID)
	}

	row, ok := companyFromDB(t, created.ID)
	if !ok {
		t.Fatal("company not found in the database")
	}

	if row.Name != name || row.Type != "Cooperative" || row.AmountOfEmployees != 42 || row.Registered {
		t.Fatalf("unexpected row %+v", row)
	}
	if row.Description == nil || *row.Description != "Persisted description" {
		t.Fatalf("unexpected description %v", row.Description)
	}
}

func TestCreateCompany_AllTypes(t *testing.T) {
	token := login(t)

	for _, companyType := range []string{"Corporations", "NonProfit", "Cooperative", "Sole Proprietorship"} {
		t.Run(companyType, func(t *testing.T) {
			payload := validCompany(uniqueName("Type"))
			payload["type"] = companyType

			created := createCompany(t, token, payload)
			if created.Type != companyType {
				t.Fatalf("type %q, want %q", created.Type, companyType)
			}

			if got := getCompany(t, created.ID); got.Type != companyType {
				t.Fatalf("stored type %q, want %q", got.Type, companyType)
			}
		})
	}
}

func TestCreateCompany_DescriptionIsOptional(t *testing.T) {
	token := login(t)

	payload := validCompany(uniqueName("NoDesc"))
	delete(payload, "description")

	created := createCompany(t, token, payload)
	if created.Description != nil {
		t.Fatalf("expected no description, got %q", *created.Description)
	}

	row, _ := companyFromDB(t, created.ID)
	if row.Description != nil {
		t.Fatalf("expected NULL description in db, got %q", *row.Description)
	}
}

func TestCreateCompany_Boundaries(t *testing.T) {
	token := login(t)

	t.Run("name of exactly 15 characters", func(t *testing.T) {
		created := createCompany(t, token, validCompany("Exactly15Chars!"))
		if created.Name != "Exactly15Chars!" {
			t.Fatalf("name %q", created.Name)
		}
	})

	t.Run("15 multibyte characters", func(t *testing.T) {
		// 15 runes but 28 bytes: the limit must be counted in characters, not bytes.
		name := "ООО Ромашка Плю"
		created := createCompany(t, token, validCompany(name))
		if created.Name != name {
			t.Fatalf("name %q", created.Name)
		}
	})

	t.Run("description of exactly 3000 characters", func(t *testing.T) {
		payload := validCompany(uniqueName("Desc"))
		payload["description"] = strings.Repeat("a", 3000)
		createCompany(t, token, payload)
	})

	t.Run("one employee", func(t *testing.T) {
		payload := validCompany(uniqueName("One"))
		payload["amount_of_employees"] = 1
		createCompany(t, token, payload)
	})
}

func TestCreateCompany_ValidationTable(t *testing.T) {
	token := login(t)
	before := countCompanies(t)

	cases := []struct {
		name   string
		mutate func(p map[string]any)
	}{
		{"missing name", func(p map[string]any) { delete(p, "name") }},
		{"empty name", func(p map[string]any) { p["name"] = "" }},
		{"16 characters", func(p map[string]any) { p["name"] = strings.Repeat("x", 16) }},
		{"16 multibyte characters", func(p map[string]any) { p["name"] = strings.Repeat("ж", 16) }},
		{"description 3001", func(p map[string]any) { p["description"] = strings.Repeat("a", 3001) }},
		{"missing type", func(p map[string]any) { delete(p, "type") }},
		{"unknown type", func(p map[string]any) { p["type"] = "LLC" }},
		{"type wrong case", func(p map[string]any) { p["type"] = "corporations" }},
		{"missing employees", func(p map[string]any) { delete(p, "amount_of_employees") }},
		{"zero employees", func(p map[string]any) { p["amount_of_employees"] = 0 }},
		{"negative employees", func(p map[string]any) { p["amount_of_employees"] = -5 }},
		{"employees overflow int32", func(p map[string]any) { p["amount_of_employees"] = int64(1) << 40 }},
		{"employees as string", func(p map[string]any) { p["amount_of_employees"] = "10" }},
		{"employees fractional", func(p map[string]any) { p["amount_of_employees"] = 1.5 }},
		{"registered as string", func(p map[string]any) { p["registered"] = "yes" }},
		{"name as number", func(p map[string]any) { p["name"] = 123 }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := validCompany(uniqueName("Val"))
			tc.mutate(payload)

			status, body := doJSON(t, http.MethodPost, "/company/v1/", token, payload)
			assertStatus(t, status, http.StatusBadRequest, body)
			assertAPIError(t, body, http.StatusBadRequest)
		})
	}

	if after := countCompanies(t); after != before {
		t.Fatalf("invalid requests created %d companies", after-before)
	}
}

func TestCreateCompany_MalformedBody(t *testing.T) {
	token := login(t)

	for name, body := range map[string]string{
		"empty body":   "",
		"invalid json": `{"name": "Broken"`,
		"json array":   `[]`,
		"plain text":   `hello`,
	} {
		t.Run(name, func(t *testing.T) {
			resp := mustRaw(t, http.MethodPost, "/company/v1/", token, body)
			assertStatus(t, resp.status, http.StatusBadRequest, resp.body)
			assertAPIError(t, resp.body, http.StatusBadRequest)
		})
	}
}

func TestCreateCompany_ClientCannotChooseID(t *testing.T) {
	token := login(t)
	clientID := uuid.NewString()

	payload := validCompany(uniqueName("OwnID"))
	payload["id"] = clientID

	created := createCompany(t, token, payload)
	if created.ID == clientID {
		t.Fatal("service accepted the id sent by the client")
	}
}

func TestCreateCompany_WithoutTrailingSlash(t *testing.T) {
	token := login(t)

	status, body := doJSON(t, http.MethodPost, "/company/v1", token, validCompany(uniqueName("Slash")))
	assertStatus(t, status, http.StatusCreated, body)
}

func TestCreateCompany_ResponseHeaders(t *testing.T) {
	token := login(t)

	resp := mustRaw(t, http.MethodPost, "/company/v1/", token,
		`{"name":"`+uniqueName("Hdr")+`","type":"NonProfit","amount_of_employees":1,"registered":true}`)
	assertStatus(t, resp.status, http.StatusCreated, resp.body)

	if ct := resp.header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type %q", ct)
	}
}

func TestCreateCompany_DuplicateNameKeepsOriginal(t *testing.T) {
	token := login(t)
	name := uniqueName("Dup")

	original := createCompany(t, token, validCompany(name))

	payload := validCompany(name)
	payload["type"] = "NonProfit"
	payload["amount_of_employees"] = 999

	status, body := doJSON(t, http.MethodPost, "/company/v1/", token, payload)
	assertStatus(t, status, http.StatusConflict, body)
	assertAPIError(t, body, http.StatusConflict)

	got := getCompany(t, original.ID)
	if got.Type != "Corporations" || got.AmountOfEmployees != 10 {
		t.Fatalf("original company was changed: %+v", got)
	}
}

// The unique index must hold under concurrent requests: exactly one create wins.
func TestCreateCompany_ConcurrentSameName(t *testing.T) {
	// The service caps its pool with PG_MAX_OPEN_CONNS; do the same here, otherwise
	// hundreds of concurrent requests exceed Postgres max_connections.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	token := login(t)
	name := uniqueName("Race")
	body := `{"name":"` + name + `","type":"Corporations","amount_of_employees":1,"registered":true}`

	const workers = 20

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		statuses = map[int]int{}
		errs     []error
	)

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			resp, err := doRaw(http.MethodPost, "/company/v1/", token, body)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errs = append(errs, err)
				return
			}
			statuses[resp.status]++
		}()
	}
	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("request errors: %v", errs)
	}
	if statuses[http.StatusCreated] != 1 || statuses[http.StatusConflict] != workers-1 {
		t.Fatalf("expected 1x201 and %dx409, got %v", workers-1, statuses)
	}

	var n int
	if err := db.Get(&n, `SELECT count(*) FROM companies WHERE name = $1`, name); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("%d rows with name %q", n, name)
	}
}

// ---------------------------------------------------------------------------
// Patch and delete
// ---------------------------------------------------------------------------

func TestPatchCompany_SingleFields(t *testing.T) {
	token := login(t)

	cases := []struct {
		name   string
		patch  func() map[string]any
		verify func(t *testing.T, before, after companyResponse)
	}{
		{
			name:  "name",
			patch: func() map[string]any { return map[string]any{"name": uniqueName("New")} },
			verify: func(t *testing.T, before, after companyResponse) {
				t.Helper()
				if after.Name == before.Name {
					t.Fatal("name not changed")
				}
			},
		},
		{
			name:  "description",
			patch: func() map[string]any { return map[string]any{"description": "Updated"} },
			verify: func(t *testing.T, _, after companyResponse) {
				t.Helper()
				if after.Description == nil || *after.Description != "Updated" {
					t.Fatalf("description %v", after.Description)
				}
			},
		},
		{
			name:  "type",
			patch: func() map[string]any { return map[string]any{"type": "Sole Proprietorship"} },
			verify: func(t *testing.T, _, after companyResponse) {
				t.Helper()
				if after.Type != "Sole Proprietorship" {
					t.Fatalf("type %q", after.Type)
				}
			},
		},
		{
			name:  "amount_of_employees",
			patch: func() map[string]any { return map[string]any{"amount_of_employees": 777} },
			verify: func(t *testing.T, _, after companyResponse) {
				t.Helper()
				if after.AmountOfEmployees != 777 {
					t.Fatalf("employees %d", after.AmountOfEmployees)
				}
			},
		},
		{
			name:  "registered",
			patch: func() map[string]any { return map[string]any{"registered": false} },
			verify: func(t *testing.T, _, after companyResponse) {
				t.Helper()
				if after.Registered {
					t.Fatal("registered not changed")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := createCompany(t, token, validCompany(uniqueName("Patch")))

			status, body := doJSON(t, http.MethodPatch, "/company/v1/"+before.ID, token, tc.patch())
			assertStatus(t, status, http.StatusOK, body)

			after := getCompany(t, before.ID)
			tc.verify(t, before, after)

			// Fields that were not sent must stay the same.
			untouched := func(field string, same bool) {
				if _, sent := tc.patch()[field]; !sent && !same {
					t.Fatalf("field %s changed unexpectedly: before %+v, after %+v", field, before, after)
				}
			}
			untouched("name", before.Name == after.Name)
			untouched("type", before.Type == after.Type)
			untouched("amount_of_employees", before.AmountOfEmployees == after.AmountOfEmployees)
			untouched("registered", before.Registered == after.Registered)
			untouched("description", *before.Description == *after.Description)
		})
	}
}

func TestPatchCompany_EmptyObjectChangesNothing(t *testing.T) {
	token := login(t)
	before := createCompany(t, token, validCompany(uniqueName("Empty")))

	status, body := doJSON(t, http.MethodPatch, "/company/v1/"+before.ID, token, map[string]any{})
	assertStatus(t, status, http.StatusOK, body)

	after := getCompany(t, before.ID)
	if after.Name != before.Name || after.Type != before.Type ||
		after.AmountOfEmployees != before.AmountOfEmployees || after.Registered != before.Registered {
		t.Fatalf("company changed: before %+v, after %+v", before, after)
	}
}

func TestPatchCompany_SameNameIsNotConflict(t *testing.T) {
	token := login(t)
	company := createCompany(t, token, validCompany(uniqueName("Same")))

	status, body := doJSON(t, http.MethodPatch, "/company/v1/"+company.ID, token, map[string]any{
		"name":                company.Name,
		"amount_of_employees": 11,
	})
	assertStatus(t, status, http.StatusOK, body)
}

func TestPatchCompany_RenameToExistingName(t *testing.T) {
	token := login(t)
	first := createCompany(t, token, validCompany(uniqueName("First")))
	second := createCompany(t, token, validCompany(uniqueName("Second")))

	status, body := doJSON(t, http.MethodPatch, "/company/v1/"+second.ID, token, map[string]any{
		"name":                first.Name,
		"amount_of_employees": 500,
	})
	assertStatus(t, status, http.StatusConflict, body)
	assertAPIError(t, body, http.StatusConflict)

	// The whole patch is rolled back, including fields that were valid.
	got := getCompany(t, second.ID)
	if got.Name != second.Name || got.AmountOfEmployees != second.AmountOfEmployees {
		t.Fatalf("company changed after failed patch: %+v", got)
	}
}

func TestPatchCompany_InvalidValuesAreRolledBack(t *testing.T) {
	token := login(t)

	cases := map[string]map[string]any{
		"name too long":      {"name": strings.Repeat("x", 16), "registered": false},
		"empty name":         {"name": "", "registered": false},
		"description 3001":   {"description": strings.Repeat("a", 3001), "registered": false},
		"unknown type":       {"type": "LLC", "registered": false},
		"zero employees":     {"amount_of_employees": 0, "registered": false},
		"negative employees": {"amount_of_employees": -1, "registered": false},
	}

	for name, patch := range cases {
		t.Run(name, func(t *testing.T) {
			before := createCompany(t, token, validCompany(uniqueName("Inv")))

			status, body := doJSON(t, http.MethodPatch, "/company/v1/"+before.ID, token, patch)
			assertStatus(t, status, http.StatusBadRequest, body)
			assertAPIError(t, body, http.StatusBadRequest)

			row, _ := companyFromDB(t, before.ID)
			if !row.Registered || row.Name != before.Name || row.AmountOfEmployees != before.AmountOfEmployees {
				t.Fatalf("row changed after invalid patch: %+v", row)
			}
		})
	}
}

func TestPatchCompany_BadRequests(t *testing.T) {
	token := login(t)
	company := createCompany(t, token, validCompany(uniqueName("BadReq")))

	cases := []struct {
		name, path, body string
	}{
		{"invalid uuid", "/company/v1/not-a-uuid", `{"name":"X"}`},
		{"empty body", "/company/v1/" + company.ID, ""},
		{"invalid json", "/company/v1/" + company.ID, `{"name":`},
		{"wrong field type", "/company/v1/" + company.ID, `{"amount_of_employees":"many"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := mustRaw(t, http.MethodPatch, tc.path, token, tc.body)
			assertStatus(t, resp.status, http.StatusBadRequest, resp.body)
		})
	}
}

// PATCH reads the row with SELECT ... FOR UPDATE, so concurrent patches of different
// fields of the same company must not overwrite each other (no lost updates).
// Each company receives exactly one patch per field, all fired at the same moment:
// without the row lock a later commit writes back stale values read before an earlier one.
func TestPatchCompany_ConcurrentPatchesDoNotLoseUpdates(t *testing.T) {
	// The service caps its pool with PG_MAX_OPEN_CONNS; do the same here, otherwise
	// hundreds of concurrent requests exceed Postgres max_connections.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	token := login(t)

	patches := []string{
		`{"registered": true}`,
		`{"amount_of_employees": 500}`,
		`{"description": "concurrent"}`,
		`{"type": "NonProfit"}`,
	}

	const companies = 50

	ids := make([]string, 0, companies)
	for range companies {
		payload := validCompany(uniqueName("Conc"))
		payload["registered"] = false
		payload["amount_of_employees"] = 1
		ids = append(ids, createCompany(t, token, payload).ID)
	}

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		errs  []string
		start = make(chan struct{})
	)

	for _, id := range ids {
		for _, patch := range patches {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start // release all goroutines at once to maximize overlap

				resp, err := doRaw(http.MethodPatch, "/company/v1/"+id, token, patch)
				if err == nil && resp.status != http.StatusOK {
					err = fmt.Errorf("status %d, body %s", resp.status, resp.body)
				}
				if err != nil {
					mu.Lock()
					errs = append(errs, err.Error())
					mu.Unlock()
				}
			}()
		}
	}
	close(start)
	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("patch errors: %v", errs)
	}

	for _, id := range ids {
		got := getCompany(t, id)
		if !got.Registered || got.AmountOfEmployees != 500 || got.Type != "NonProfit" ||
			got.Description == nil || *got.Description != "concurrent" {
			t.Fatalf("lost update for %s, final state %+v", id, got)
		}
	}
}

func TestDeleteCompany_RemovesRow(t *testing.T) {
	token := login(t)
	company := createCompany(t, token, validCompany(uniqueName("Del")))

	resp := mustRaw(t, http.MethodDelete, "/company/v1/"+company.ID, token, "")
	assertStatus(t, resp.status, http.StatusNoContent, resp.body)

	if len(resp.body) != 0 {
		t.Fatalf("204 must have an empty body, got %s", resp.body)
	}

	if _, ok := companyFromDB(t, company.ID); ok {
		t.Fatal("row still exists after delete")
	}
}

func TestDeleteCompany_Twice(t *testing.T) {
	token := login(t)
	company := createCompany(t, token, validCompany(uniqueName("Twice")))

	status, body := doJSON(t, http.MethodDelete, "/company/v1/"+company.ID, token, nil)
	assertStatus(t, status, http.StatusNoContent, body)

	status, body = doJSON(t, http.MethodDelete, "/company/v1/"+company.ID, token, nil)
	assertStatus(t, status, http.StatusNotFound, body)
	assertAPIError(t, body, http.StatusNotFound)
}

func TestDeleteCompany_InvalidID(t *testing.T) {
	token := login(t)

	status, body := doJSON(t, http.MethodDelete, "/company/v1/not-a-uuid", token, nil)
	assertStatus(t, status, http.StatusBadRequest, body)
}

func TestDeleteCompany_OnlyTargetIsRemoved(t *testing.T) {
	token := login(t)
	keep := createCompany(t, token, validCompany(uniqueName("Keep")))
	remove := createCompany(t, token, validCompany(uniqueName("Remove")))

	status, body := doJSON(t, http.MethodDelete, "/company/v1/"+remove.ID, token, nil)
	assertStatus(t, status, http.StatusNoContent, body)

	getCompany(t, keep.ID)
}

func TestDeleteCompany_NameCanBeReused(t *testing.T) {
	token := login(t)
	name := uniqueName("Reuse")

	first := createCompany(t, token, validCompany(name))

	status, body := doJSON(t, http.MethodDelete, "/company/v1/"+first.ID, token, nil)
	assertStatus(t, status, http.StatusNoContent, body)

	second := createCompany(t, token, validCompany(name))
	if second.ID == first.ID {
		t.Fatal("new company got the id of the deleted one")
	}
}

func TestPatchCompany_AfterDelete(t *testing.T) {
	token := login(t)
	company := createCompany(t, token, validCompany(uniqueName("Gone")))

	status, body := doJSON(t, http.MethodDelete, "/company/v1/"+company.ID, token, nil)
	assertStatus(t, status, http.StatusNoContent, body)

	status, body = doJSON(t, http.MethodPatch, "/company/v1/"+company.ID, token, map[string]any{"registered": false})
	assertStatus(t, status, http.StatusNotFound, body)
}

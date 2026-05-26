package validate_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"iq-home/backend/pkg/validate"
)

// ─── Struct validation ────────────────────────────────────────────────────────

type checkoutInput struct {
	Name          string `validate:"required,min=2,max=100"`
	Phone         string `validate:"required,min=7,max=20"`
	Address       string `validate:"required,min=5,max=300"`
	PaymentMethod string `validate:"required,oneof=card cash other"`
}

func TestStruct_Valid(t *testing.T) {
	in := checkoutInput{
		Name:          "Алибек Сейтов",
		Phone:         "+77001234567",
		Address:       "ул. Абая 1, Алматы",
		PaymentMethod: "card",
	}
	if err := validate.Struct(in); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestStruct_Required(t *testing.T) {
	err := validate.Struct(checkoutInput{})
	if err == nil {
		t.Fatal("expected error for empty struct")
	}
	for _, field := range []string{"name", "phone", "address", "paymentmethod"} {
		if !strings.Contains(strings.ToLower(err.Error()), field) {
			t.Errorf("expected error to mention %q, got: %s", field, err.Error())
		}
	}
}

func TestStruct_InvalidEmail(t *testing.T) {
	type emailInput struct {
		Email string `validate:"required,email"`
	}
	err := validate.Struct(emailInput{Email: "not-an-email"})
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
	if !strings.Contains(err.Error(), "valid email") {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestStruct_MinLength(t *testing.T) {
	in := checkoutInput{
		Name:          "А", // too short
		Phone:         "+77001234567",
		Address:       "ул. Абая 1, Алматы",
		PaymentMethod: "card",
	}
	err := validate.Struct(in)
	if err == nil {
		t.Fatal("expected error for short name")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("expected name error, got: %s", err.Error())
	}
}

func TestStruct_InvalidOneof(t *testing.T) {
	in := checkoutInput{
		Name:          "Алибек",
		Phone:         "+77001234567",
		Address:       "ул. Абая 1, Алматы",
		PaymentMethod: "bitcoin", // not in oneof
	}
	err := validate.Struct(in)
	if err == nil {
		t.Fatal("expected error for invalid payment method")
	}
	if !strings.Contains(err.Error(), "paymentmethod") {
		t.Errorf("expected paymentmethod error, got: %s", err.Error())
	}
}

// ─── DecodeAndValidate ────────────────────────────────────────────────────────

func TestDecodeAndValidate_Valid(t *testing.T) {
	type body struct {
		Name string `json:"name" validate:"required,min=2"`
	}
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Алибек"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	var dst body
	ok := validate.DecodeAndValidate(w, r, &dst)
	if !ok {
		t.Fatalf("expected ok=true, got false; body: %s", w.Body.String())
	}
	if dst.Name != "Алибек" {
		t.Errorf("expected name=Алибек, got %q", dst.Name)
	}
}

func TestDecodeAndValidate_InvalidJSON(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{invalid`))
	w := httptest.NewRecorder()

	var dst struct{ Name string }
	ok := validate.DecodeAndValidate(w, r, &dst)
	if ok {
		t.Fatal("expected ok=false for invalid JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDecodeAndValidate_ValidationFails(t *testing.T) {
	type body struct {
		Email string `json:"email" validate:"required,email"`
	}
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"bad"}`))
	w := httptest.NewRecorder()

	var dst body
	ok := validate.DecodeAndValidate(w, r, &dst)
	if ok {
		t.Fatal("expected ok=false for invalid email")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "email") {
		t.Errorf("expected email in error body, got: %s", w.Body.String())
	}
}

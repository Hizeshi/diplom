package compatibility

import (
	"context"
	"testing"

	"iq-home/backend/internal/domain/compatibility"
)

// ── stubs ────────────────────────────────────────────────────────────────────

type stubRepo struct {
	items []compatibility.CartItem
}

func (r *stubRepo) GetCartForCompatibility(_ context.Context, _ string) ([]compatibility.CartItem, error) {
	return r.items, nil
}

type stubLLM struct {
	response string
}

func (l *stubLLM) Complete(_ context.Context, _ CompleteOptions) (string, error) {
	return l.response, nil
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestCheckCart_EmptyCart(t *testing.T) {
	svc := New(&stubRepo{}, &stubLLM{}, "gpt-5.4-mini")
	result, err := svc.CheckCart(context.Background(), "user1")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Compatible {
		t.Fatal("empty cart should be compatible")
	}
	if result.ItemCount != 0 {
		t.Fatalf("want 0, got %d", result.ItemCount)
	}
}

func TestCheckCart_Compatible(t *testing.T) {
	items := []compatibility.CartItem{
		{ProductID: 1, Name: "Выключатель FD", ProductType: "Выключатель", SeriesName: "FD"},
		{ProductID: 2, Name: "Рамка FD 1 пост", ProductType: "Рамка", SeriesName: "FD"},
	}
	llm := &stubLLM{response: `{"compatible": true, "issues": []}`}
	svc := New(&stubRepo{items: items}, llm, "gpt-5.4-mini")

	result, err := svc.CheckCart(context.Background(), "user1")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Compatible {
		t.Fatal("expected compatible")
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected no issues, got %v", result.Issues)
	}
	if result.ItemCount != 2 {
		t.Fatalf("want 2, got %d", result.ItemCount)
	}
}

func TestCheckCart_IncompatibleSeries(t *testing.T) {
	items := []compatibility.CartItem{
		{ProductID: 10, Name: "Выключатель FD", ProductType: "Выключатель", SeriesName: "FD"},
		{ProductID: 20, Name: "Рамка G-серия", ProductType: "Рамка", SeriesName: "G-серия"},
	}
	llm := &stubLLM{response: `{
		"compatible": false,
		"issues": [{
			"type": "incompatible",
			"product_ids": [10, 20],
			"message": "Рамка G-серия не подходит к механизму FD."
		}]
	}`}
	svc := New(&stubRepo{items: items}, llm, "gpt-5.4-mini")

	result, err := svc.CheckCart(context.Background(), "user1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Compatible {
		t.Fatal("expected incompatible")
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Type != "incompatible" {
		t.Fatalf("expected type incompatible, got %q", result.Issues[0].Type)
	}
}

func TestCheckCart_DimmerWarning(t *testing.T) {
	items := []compatibility.CartItem{
		{ProductID: 30, Name: "Диммер LED 30-400W", ProductType: "Диммер", ConfiguratorType: "dimmer", SeriesName: "G-серия"},
	}
	llm := &stubLLM{response: `{
		"compatible": true,
		"issues": [{
			"type": "warning",
			"product_ids": [30],
			"message": "Уточните тип ламп и проводки перед установкой диммера."
		}]
	}`}
	svc := New(&stubRepo{items: items}, llm, "gpt-5.4-mini")

	result, err := svc.CheckCart(context.Background(), "user1")
	if err != nil {
		t.Fatal(err)
	}
	// compatible=true даже при наличии warning
	if !result.Compatible {
		t.Fatal("warning should not make cart incompatible")
	}
	if len(result.Issues) != 1 || result.Issues[0].Type != "warning" {
		t.Fatalf("expected 1 warning, got %v", result.Issues)
	}
}

func TestBuildCartMessage_ContainsProductInfo(t *testing.T) {
	items := []compatibility.CartItem{
		{ProductID: 92, Name: "Выключатель", ProductType: "Выключатель", SeriesName: "FD", Quantity: 2},
	}
	msg := buildCartMessage(items)
	if msg == "" {
		t.Fatal("empty message")
	}
	for _, want := range []string{"92", "Выключатель", "FD", "2"} {
		if !contains(msg, want) {
			t.Errorf("message missing %q", want)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := range s {
		if i+len(sub) <= len(s) && s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

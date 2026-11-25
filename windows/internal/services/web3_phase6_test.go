package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateDWebServicesCID(t *testing.T) {
	var gotAuth string
	var gotBody map[string]string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	if err := UpdateDWebServicesCID(context.Background(), ts.URL, "planetable.eth", "secret-key", "bafy123", ts.Client()); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret-key" || gotBody["domain"] != "planetable.eth" || gotBody["cid"] != "bafy123" {
		t.Fatalf("unexpected dWebServices request: auth=%q body=%+v", gotAuth, gotBody)
	}
}

func TestBuildTipPaymentURI(t *testing.T) {
	uri, err := BuildTipPaymentURI("planetable.eth", "0.005")
	if err != nil {
		t.Fatal(err)
	}
	want := "ethereum:planetable.eth@1?value=5000000000000000"
	if uri != want {
		t.Fatalf("got %q, want %q", uri, want)
	}
	if _, err := BuildTipPaymentURI("not-an-address", "0.01"); err == nil {
		t.Fatal("expected error for invalid recipient")
	}
	if _, err := BuildTipPaymentURI("planetable.eth", "-1"); err == nil {
		t.Fatal("expected error for negative amount")
	}
}

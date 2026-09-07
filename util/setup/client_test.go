package setup

import "testing"

func TestIntentPathIncludesSelectedOrganization(t *testing.T) {
	client := &Client{OrgID: "org customer/admin"}

	got := client.intentPath("70a718b5-853e-4c1c-8c9d-941bc768235d")
	want := "/api/v2/setups/70a718b5-853e-4c1c-8c9d-941bc768235d?organizationId=org+customer%2Fadmin"
	if got != want {
		t.Fatalf("intentPath() = %q, want %q", got, want)
	}
}

func TestIntentPathOmitsEmptyOrganization(t *testing.T) {
	client := &Client{}

	got := client.intentPath("70a718b5-853e-4c1c-8c9d-941bc768235d")
	want := "/api/v2/setups/70a718b5-853e-4c1c-8c9d-941bc768235d"
	if got != want {
		t.Fatalf("intentPath() = %q, want %q", got, want)
	}
}

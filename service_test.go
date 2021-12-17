package main

import (
	"testing"
)

func TestState_String(t *testing.T) {
	assertCorrectMessage := func(t testing.TB, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	}

	t.Run("Check values", func(t *testing.T) {
		got := []State{
			Unknown,
			NotAvailable,
			Available,
		}

		want := []string{
			"Unknown",
			"NotAvailable",
			"Available",
		}

		for i, s := range got {
			if _, err := setState(&appState, s); err != nil {
				t.Error(err)
			}

			if appState.String() != want[i] {
				assertCorrectMessage(t, appState.String(), want[i])
			}
		}
	})
}

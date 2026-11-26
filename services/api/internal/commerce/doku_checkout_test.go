package commerce

import "testing"

func TestDOKUSignatureDeterministicAndBodySensitive(t *testing.T) {
	a := dokuSignature("client", "request", "2026-11-26T00:00:00Z", "/checkout/v1/payment", dokuDigest([]byte(`{"a":1}`)), "secret")
	b := dokuSignature("client", "request", "2026-11-26T00:00:00Z", "/checkout/v1/payment", dokuDigest([]byte(`{"a":2}`)), "secret")
	if a == b {
		t.Fatal("signature must bind body digest")
	}
	if a != dokuSignature("client", "request", "2026-11-26T00:00:00Z", "/checkout/v1/payment", dokuDigest([]byte(`{"a":1}`)), "secret") {
		t.Fatal("signature not deterministic")
	}
}

func TestDOKUAmount(t *testing.T) {
	for _, tc := range []struct {
		in   any
		want int64
	}{{float64(125000), 125000}, {"125000.00", 125000}} {
		got, err := dokuAmount(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("%v: %d %v", tc.in, got, err)
		}
	}
}

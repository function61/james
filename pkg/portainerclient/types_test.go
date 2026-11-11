package portainerclient

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIDJSONCompatibility(t *testing.T) {
	for _, value := range []any{&Stack{}, &Endpoint{}} {
		if err := json.Unmarshal([]byte(`{"Id":42}`), value); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(encoded), `"Id":42`) {
			t.Errorf("%T JSON = %s, want Id 42", value, encoded)
		}
	}
}

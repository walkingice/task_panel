package config

import "testing"

func TestTOMLDecoderUnmarshalsProcess(t *testing.T) {
	var configuration Configuration
	data := []byte("[[process]]\nname = 'web'\nstart = 'serve-web'\n")

	if err := (TOMLDecoder{}).Unmarshal(data, &configuration); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(configuration.Processes) != 1 {
		t.Fatalf("process count = %d, want 1", len(configuration.Processes))
	}
	if got := configuration.Processes[0].Name; got != "web" {
		t.Fatalf("process name = %q, want %q", got, "web")
	}
}

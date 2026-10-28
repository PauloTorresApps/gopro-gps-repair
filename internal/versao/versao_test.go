package versao

import "testing"

func TestAtual(t *testing.T) {
	old := Version
	defer func() { Version = old }()

	Version = "v1.2.3"
	if got := Atual(); got != "1.2.3" {
		t.Fatalf("Atual() = %q, quero 1.2.3", got)
	}
	Version = ""
	if got := Atual(); got == "" {
		t.Fatal("Atual() vazia sem -ldflags")
	}
}

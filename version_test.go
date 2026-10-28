package goprogpsrepair

import (
	"regexp"
	"testing"
)

func TestVersion(t *testing.T) {
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(Version) {
		t.Fatalf("VERSION deve ser x.y.z, contém %q", Version)
	}
}

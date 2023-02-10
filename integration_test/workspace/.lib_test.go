package lib_test

import (
	"testing"

	"github.com/project/lib"
)

func TestLib(t *testing.T) {
	if lib.Abc(123) != 123 {
		t.Fail()
	}
}

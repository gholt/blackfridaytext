package blackfridaytext

import (
	"testing"
)

func TestBasic(t *testing.T) {
	in := "Basic Test"
	out := string(MarkdownToTextNoMetadata([]byte(in), 0, false))
	exp := "Basic Test\n"
	if out != exp {
		t.Errorf("%#v != %#v", out, exp)
	}
}

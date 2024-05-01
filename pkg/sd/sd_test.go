package sd

import (
	"testing"
)

func Test_genEsdPathWithArray(t *testing.T) {
	path := []string{"a", "b", "c"}
	expect := "/esd/a/b/c"

	actual := genEsdPath(path...)
	if expect != actual {
		t.Fatalf("expected %s, got %s", expect, actual)
	}
}

func Test_genEsdPathWithSingle(t *testing.T) {
	path := "a"
	expect := "/esd/a"

	actual := genEsdPath(path)
	if expect != actual {
		t.Fatalf("expected %s, got %s", expect, actual)
	}
}

func Test_genEsdPathWithEmpty(t *testing.T) {
	expect := "/esd"

	actual := genEsdPath()
	if expect != actual {
		t.Fatalf("expected %s, got %s", expect, actual)
	}
}

func Test_genEsdPathWithEmptyString(t *testing.T) {
	expect := "/esd"

	actual := genEsdPath("")
	if expect != actual {
		t.Fatalf("expected %s, got %s", expect, actual)
	}
}

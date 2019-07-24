package common

import "testing"

func TestRandomInt(t *testing.T) {
	t.Log(RandomInt(1, 20))
}

func TestRandomString(t *testing.T) {
	t.Log(RandomString(10))
}

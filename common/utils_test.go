package common

import "testing"

func TestSliceLength(t *testing.T) {
	array := make([]bool, 10)
	t.Log(len(array))
}

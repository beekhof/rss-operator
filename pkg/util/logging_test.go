package util

import "testing"

func TestLogging(t *testing.T) {

	l := GetLogger("test")
	l.Debug("one")
	l.V(1).Debug("two")
	l.V(2).Debug("three")
}

package test

import (
	"myproject/mathutils" // Replace 'yourmodule' with the actual module name from go.mod
	"testing"
)

// TestAdd tests the Add function
func TestAdd(t *testing.T) {
	result := mathutils.Add(1, 2)
	expected := 3
	if result != expected {
		t.Errorf("Add(1, 2) = %d; want %d", result, expected)
	}
}

// TestSub tests the Sub function
func TestSub(t *testing.T) {
	result := mathutils.Sub(5, 3)
	expected := 2
	if result != expected {
		t.Errorf("Sub(5, 3) = %d; want %d", result, expected)
	}
}

package tools

import (
	"reflect"
	"testing"
)

func TestParsePageRange(t *testing.T) {
	got, err := ParsePageRange("1-3, 5, 3", 6)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{1, 2, 3, 5}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	got, err = ParsePageRange("4-", 6)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{4, 5, 6}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestParsePageRangeRejectsOutOfBounds(t *testing.T) {
	if _, err := ParsePageRange("0,2", 3); err == nil {
		t.Fatal("expected error")
	}
	if _, err := ParsePageRange("2-5", 3); err == nil {
		t.Fatal("expected error")
	}
}

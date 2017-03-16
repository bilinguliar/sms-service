package main

import "testing"

func TestNewUDH(t *testing.T) {
	var (
		total   byte
		current byte
	)
	total = 5
	current = 3

	header := NewUDH(total, current)

	actual := header.ToHexStr()
	expected := "050003000503"

	if actual != expected {
		t.Errorf("Header is: %q, expected: %q", actual, expected)
	}
}

func TestNewUDHIsSet(t *testing.T) {
	cases := []struct {
		header UDH
		want   bool
	}{
		{
			header: UDH{},
			want:   false,
		},
		{
			header: NewUDH(2, 2),
			want:   true,
		},
		{
			header: NewUDH(2, 1),
			want:   true,
		},
	}

	for _, c := range cases {
		actual := c.header.IsSet()
		if actual != c.want {
			t.Errorf("Header isSet returns: %v, expected: %v", actual, c.want)
		}
	}
}

func TestNewUDHPanicsWithZeroTotal(t *testing.T) {
	expected := true
	defer testUDHPanic(expected, t)
	NewUDH(0, 1)
}

func TestNewUDHPanicsWithZeroCurrent(t *testing.T) {
	expected := true
	defer testUDHPanic(expected, t)
	NewUDH(1, 0)
}

func TestNewUDHPanicsWithBothZeros(t *testing.T) {
	expected := true
	defer testUDHPanic(expected, t)
	NewUDH(0, 0)
}

func TestNewUDHPanicsWithCurrentMoreThanTotal(t *testing.T) {
	expected := true
	defer testUDHPanic(expected, t)
	NewUDH(5, 6)
}

func TestNewUDHPanicsWithCurrentMoreThanNine(t *testing.T) {
	expected := true
	defer testUDHPanic(expected, t)
	NewUDH(10, 10)
}

func TestNewUDHPanicsWithTotalMoreThanNine(t *testing.T) {
	expected := true
	defer testUDHPanic(expected, t)
	NewUDH(10, 1)
}

func TestNewUDHNotPanicsWithValidInput(t *testing.T) {
	expected := false
	defer testUDHPanic(expected, t)

	cases := []struct {
		total   byte
		current byte
	}{
		{
			total:   1,
			current: 1,
		},
		{
			total:   5,
			current: 4,
		},

		{
			total:   9,
			current: 9,
		},
	}

	for _, c := range cases {
		NewUDH(c.total, c.current)
	}
}

func testUDHPanic(expected bool, t *testing.T) {
	err := recover()
	if actual, ok := err.(error); ok {
		if !expected {
			t.Error("Panic happened, but we did not expect that.")
		}
		expectedText := "header values violation"
		if actual.Error() != expectedText {
			t.Errorf("Actual error: %q, expected error: %q", actual.Error(), expectedText)
		}
	}

	if expected && err == nil {
		t.Error("Panic expected but never happened.")
	}
}

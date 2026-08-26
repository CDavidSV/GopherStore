package util

import (
	"reflect"
	"testing"
)

func TestParsePositiveInt(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  int
		ok    bool
	}{
		{name: "positive", input: []byte("42"), want: 42, ok: true},
		{name: "zero", input: []byte("0"), want: 0, ok: true},
		{name: "negative", input: []byte("-1"), ok: false},
		{name: "invalid", input: []byte("abc"), ok: false},
		{name: "empty", input: nil, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParsePositiveInt(tt.input)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("ParsePositiveInt(%q) = (%d, %t), want (%d, %t)", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  int
		ok    bool
	}{
		{name: "positive", input: []byte("42"), want: 42, ok: true},
		{name: "negative", input: []byte("-42"), want: -42, ok: true},
		{name: "zero", input: []byte("0"), want: 0, ok: true},
		{name: "invalid", input: []byte("12x"), ok: false},
		{name: "empty", input: nil, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseInt(tt.input)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("ParseInt(%q) = (%d, %t), want (%d, %t)", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestReverseSlice(t *testing.T) {
	tests := []struct {
		name  string
		input [][]int
		want  [][]int
	}{
		{name: "empty", input: [][]int{}, want: [][]int{}},
		{name: "single", input: [][]int{{1}}, want: [][]int{{1}}},
		{name: "multiple", input: [][]int{{1}, {2}, {3}, {4}}, want: [][]int{{4}, {3}, {2}, {1}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make([][]int, len(tt.input))
			copy(input, tt.input)
			ReverseSlice(input)
			if !reflect.DeepEqual(input, tt.want) {
				t.Fatalf("ReverseSlice() = %v, want %v", input, tt.want)
			}
		})
	}
}

func TestSliceList(t *testing.T) {
	list := []string{"zero", "one", "two", "three", "four"}
	tests := []struct {
		name       string
		input      []string
		start, end int
		want       []string
	}{
		{name: "normal range", input: list, start: 1, end: 3, want: []string{"one", "two", "three"}},
		{name: "negative indexes", input: list, start: -3, end: -1, want: []string{"two", "three", "four"}},
		{name: "clamp indexes", input: list, start: -20, end: 20, want: list},
		{name: "start after end", input: list, start: 3, end: 1, want: []string{}},
		{name: "empty list", input: []string{}, start: 0, end: 1, want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SliceList(tt.input, tt.start, tt.end)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SliceList(%v, %d, %d) = %v, want %v", tt.input, tt.start, tt.end, got, tt.want)
			}
		})
	}
}

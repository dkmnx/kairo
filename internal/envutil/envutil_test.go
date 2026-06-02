package envutil

import (
	"reflect"
	"testing"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		name string
		in   [][]string
		want []string
	}{
		{"empty", nil, []string{}},
		{"single", [][]string{{"A=1"}}, []string{"A=1"}},
		{"dedup later wins", [][]string{{"A=1"}, {"A=2", "B=3"}}, []string{"A=2", "B=3"}},
		{"later position wins", [][]string{{"A=1", "B=2"}, {"C=3"}, {"A=99"}}, []string{"B=2", "C=3", "A=99"}},
		{"malformed skipped", [][]string{{"=bad", "novalue", "ok=1"}}, []string{"ok=1"}},
		{"empty value allowed", [][]string{{"A="}}, []string{"A="}},
		{"legacy three-slice behavior", [][]string{{"KEY1=first"}, {"KEY2=value2"}, {"KEY1=last", "KEY3=value3"}}, []string{"KEY2=value2", "KEY1=last", "KEY3=value3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Merge(tt.in...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Merge(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

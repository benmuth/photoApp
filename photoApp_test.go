package main

import (
	"testing"
)

func TestPerm(t *testing.T) {
	examples := []struct {
		name    string
		albumID int64
		userID  int64
		want    bool
	}{
		{
			name:    "perm",
			albumID: 1,
			userID:  1,
			want:    true,
		},
		{
			name:    "noPerm",
			albumID: 1,
			userID:  2,
			want:    false,
		},
	}
	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			got := checkPerm(ex.albumID, ex.userID)
			if got != ex.want {
				t.Fatalf("got %v, want %v\n", got, ex.want)
			}
		})
	}
}

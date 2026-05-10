package domain

import "testing"

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name string
		pass string
		want error
	}{
		{"empty", "", ErrInvalidPassword},
		{"short", "short", ErrInvalidPassword},
		{"ok_min", "12345678", nil},
		{"too_long", string(make([]byte, PasswordMaxLen+1)), ErrInvalidPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.pass)
			if tt.want != nil {
				if err != tt.want {
					t.Fatalf("want %v, got %v", tt.want, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

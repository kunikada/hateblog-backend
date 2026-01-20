package archive

import "fmt"

const allowedMinUsersText = "5, 10, 50, 100, 500, 1000"

// IsAllowedMinUsers returns true when the value is an allowed archive threshold.
func IsAllowedMinUsers(value int) bool {
	switch value {
	case 5, 10, 50, 100, 500, 1000:
		return true
	default:
		return false
	}
}

// ValidateMinUsers validates the archive threshold for min_users.
func ValidateMinUsers(value int) error {
	if !IsAllowedMinUsers(value) {
		return fmt.Errorf("min_users must be one of %s", allowedMinUsersText)
	}
	return nil
}

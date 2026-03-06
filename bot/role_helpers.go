package bot

// roleInList checks if targetID exists in the roles slice.
func roleInList(roles []string, targetID string) bool {
	if targetID == "" {
		return false
	}
	for _, r := range roles {
		if r == targetID {
			return true
		}
	}
	return false
}

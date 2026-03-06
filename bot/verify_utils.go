package bot

import "strings"

func cleanPhoneNumber(phone string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)
	// Remove leading 1 for US numbers
	if len(digits) == 11 && digits[0] == '1' {
		digits = digits[1:]
	}
	// Require exactly 10 digits for a valid US phone number
	if len(digits) != 10 {
		return ""
	}
	return "+1" + digits
}

func nvl(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

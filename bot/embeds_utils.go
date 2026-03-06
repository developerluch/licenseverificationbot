package bot

import (
	"fmt"
	"strings"
)

func stageLabel(stage int) string {
	switch stage {
	case 1:
		return "1 \u2014 Joined"
	case 2:
		return "2 \u2014 Form Started"
	case 3:
		return "3 \u2014 Sorted"
	case 4:
		return "4 \u2014 Student"
	case 5:
		return "5 \u2014 Licensed"
	case 6:
		return "6 \u2014 Contracting"
	case 7:
		return "7 \u2014 Setup"
	case 8:
		return "8 \u2014 Active"
	default:
		return fmt.Sprintf("%d \u2014 Unknown", stage)
	}
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "_", " ")
	return strings.Title(s)
}

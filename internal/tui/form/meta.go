package form

// FocusedFlag returns the default CLI flag name for a focused row index.
func FocusedFlag(focus, fieldCount int) string {
	if focus == fieldCount {
		return "--flash-attn"
	}
	if focus == fieldCount+2 {
		return "--mlock"
	}
	return FieldDefaultFlag(focus)
}

// DescriptionTitle returns the title shown in the form details pane.
func DescriptionTitle(focus, fieldCount int) string {
	if focus < fieldCount {
		return Labels[focus]
	}
	switch focus - fieldCount {
	case 0:
		return "Flash Attention"
	case 1:
		return "CPU Only"
	case 2:
		return "MLock"
	case 3:
		return "Layer Split"
	default:
		return "Save Profile"
	}
}

// DescriptionText returns the help text for the focused form row.
func DescriptionText(focus, fieldCount int) string {
	if focus < fieldCount {
		return FieldDescription(focus)
	}
	switch focus - fieldCount {
	case 0:
		return FieldDescription(fieldCount)
	case 1:
		return FieldDescription(fieldCount + 1)
	case 2:
		return FieldDescription(fieldCount + 2)
	case 3:
		return FieldDescription(fieldCount + 3)
	default:
		return FieldDescription(fieldCount + 4)
	}
}

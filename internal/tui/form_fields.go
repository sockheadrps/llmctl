package tui

import tui_form "github.com/sockheadrps/llmctl/internal/tui/form"

// fieldDefaultFlag returns the default llama-server CLI flag for a form field
// index, or "" for fields that don't map to a single CLI flag.
func fieldDefaultFlag(idx int) string {
	return tui_form.FieldDefaultFlag(idx)
}

func buildFormFields(defaults []string) []formField {
	built := tui_form.BuildFields(defaults)
	fields := make([]formField, len(built))
	for i, field := range built {
		fields[i] = formField{label: field.Label, input: field.Input}
	}
	return fields
}

func formFieldDescription(idx int) string {
	return tui_form.FieldDescription(idx)
}

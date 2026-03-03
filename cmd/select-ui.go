package cmd

import "github.com/manifoldco/promptui"

func simpleSelectTemplates(noun string) *promptui.SelectTemplates {
	selectedLabel := "Selected"
	if noun != "" {
		selectedLabel = "Selected " + noun
	}

	return &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "> {{ . | cyan }}",
		Inactive: "  {{ . }}",
		Selected: selectedLabel + ": {{ . | cyan }}",
	}
}

func titleMetaSelectTemplates(noun string) *promptui.SelectTemplates {
	selectedLabel := "Selected"
	if noun != "" {
		selectedLabel = "Selected " + noun
	}

	return &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "> {{ .Title | cyan }}{{ if .Meta }} {{ .Meta | faint }}{{ end }}",
		Inactive: "  {{ .Title }}{{ if .Meta }} {{ .Meta | faint }}{{ end }}",
		Selected: selectedLabel + ": {{ .Title | cyan }}{{ if .Meta }} {{ .Meta }}{{ end }}",
	}
}

func selectSize(total int) int {
	if total < 2 {
		return total
	}
	if total > 12 {
		return 12
	}

	return total
}

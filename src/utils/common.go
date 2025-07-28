package utils

import (
	"fmt"
	"github.com/radiusmethod/promptui"
	"log"
	"os"
	"path/filepath"
)

func GetEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func CheckError(err error) {
	if err.Error() == "^D" {
		// https://github.com/manifoldco/promptui/issues/179
		log.Fatalf("<Del> not supported")
	} else if err.Error() == "^C" {
		os.Exit(1)
	} else {
		log.Fatal(err)
	}
}

func GetHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory: %v\n", err)
	}
	return homeDir
}

func GetCurrentProfileFile() string {
	return GetEnv("AWS_CONFIG_FILE", filepath.Join(GetHomeDir(), ".aws/config"))
}

func AppendIfNotExists(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

func Contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

func CreatePrompt(elements []string) (string, error) {
	prompt := promptui.Select{
		Label:        fmt.Sprintf(PromptColor, "Choose an element"),
		Items:        elements,
		HideHelp:     true,
		HideSelected: true,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   fmt.Sprintf("%s {{ . | cyan }}", promptui.IconSelect),
			Inactive: "  {{.}}",
			Selected: "  {{ . | cyan }}",
		},
		Searcher:          NewPromptUISearcher(elements),
		StartInSearchMode: true,
		Stdout:            &BellSkipper{},
	}

	_, result, err := prompt.Run()

	if err != nil {
		CheckError(err)
		return "", nil
	}
	return result, nil
}

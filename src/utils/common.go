package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	newprompt "github.com/manifoldco/promptui"
	"github.com/spf13/afero"
)

func TouchFile(fs afero.Fs, name string) error {
	file, err := fs.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	return file.Close()
}

func WriteFile(fs afero.Fs, profile, loc string) error {
	homeDir, err := GetHomeDir()
	if err != nil {
		return err
	}
	if err := TouchFile(fs, fmt.Sprintf("%s/.paws", homeDir)); err != nil {
		return err
	}
	s := []byte("")
	if profile != "default" {
		s = []byte(profile)
	}
	return afero.WriteFile(fs, fmt.Sprintf("%s/.paws", loc), s, 0644)
}

func GetEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func GetHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting user home directory: %w", err)
	}
	return homeDir, nil
}

func GetCurrentProfileFile() string {
	homeDir, err := GetHomeDir()
	if err != nil {
		// Fall back to empty string if we can't get home directory
		// The caller will handle the missing config file appropriately
		return GetEnv("AWS_CONFIG_FILE", "")
	}
	return GetEnv("AWS_CONFIG_FILE", filepath.Join(homeDir, ".aws/config"))
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

func CreateNewPrompt(elements []string) (string, error) {

	templates := &newprompt.SelectTemplates{
		Label:    "{{ . | italic }}:",
		Active:   fmt.Sprintf("%s {{ . | cyan }}", newprompt.IconSelect),
		Inactive: "  {{.| faint }}",
		Selected: "  {{ . | cyan }}",
	}

	searcher := func(input string, index int) bool {
		element := elements[index]
		name := strings.ReplaceAll(strings.ToLower(element), " ", "")
		input = strings.ReplaceAll(strings.ToLower(input), " ", "")

		return strings.Contains(name, input)
	}

	prompt := newprompt.Select{
		Label:             "Select an item",
		Templates:         templates,
		Items:             elements,
		Searcher:          searcher,
		StartInSearchMode: true,
	}

	_, result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil

}

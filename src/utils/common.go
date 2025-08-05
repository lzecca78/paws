package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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
	if err := TouchFile(fs, fmt.Sprintf("%s/.paws", GetHomeDir())); err != nil {
		return err
	}
	s := []byte("")
	if profile != "default" {
		s = []byte(profile)
	}
	err := afero.WriteFile(fs, fmt.Sprintf("%s/.paws", loc), s, 0644)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

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

func CreateNewPrompt(elements []string) (string, error) {

	templates := &newprompt.SelectTemplates{
		Label:    "{{ . | italic }}:",
		Active:   fmt.Sprintf("%s {{ . | cyan }}", newprompt.IconSelect),
		Inactive: "  {{.| faint }}",
		Selected: "  {{ . | cyan }}",
	}

	prompt := newprompt.Select{
		Label:     "Select an item",
		Templates: templates,
		Items:     elements,
	}

	_, result, err := prompt.Run()

	if err != nil {
		CheckError(err)
		return "", err
	}

	return result, nil

}

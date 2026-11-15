package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const LOCAL_DATA_FILE_PATH = "/Users/andrewserra/GithubProjects/kitchen/credentials.toml"

type Email = string

type LocalAccountData struct {
	ID           int       `toml:"id"`
	AccessToken  string    `toml:"access_token"`
	RefreshToken string    `toml:"refresh_token"`
	SavedAt      time.Time `toml:"saved_at"`
}

type LocalData map[Email]LocalAccountData

func loadLocalData() (LocalData, error) {
	inData, err := os.ReadFile(LOCAL_DATA_FILE_PATH)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LocalData{}, nil
		}

		return LocalData{}, err
	}

	var localData LocalData
	err = toml.Unmarshal(inData, &localData)
	if err != nil {
		return LocalData{}, err
	}

	return localData, nil
}

func saveLocalData(localData LocalData) error {
	data, err := toml.Marshal(localData)
	if err != nil {
		return err
	}
	return os.WriteFile(LOCAL_DATA_FILE_PATH, data, 0644)
}

func printSavedAccounts(localData LocalData) {
	emailWidth := 40
	idWidth := 10
	savedAtWidth := 20

	fmt.Printf("%-*s %-*s %-*s\n", emailWidth, "Email", idWidth, "ID", savedAtWidth, "Saved At")
	for k, v := range localData {
		var savedAt string
		if v.SavedAt.IsZero() {
			savedAt = "N/A"
		} else {
			savedAt = v.SavedAt.Format(time.RFC3339)
		}
		fmt.Printf("%-*s %-*d %-*s\n", emailWidth, k, idWidth, v.ID, savedAtWidth, savedAt)
	}
}

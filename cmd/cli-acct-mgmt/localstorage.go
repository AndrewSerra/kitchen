package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/pelletier/go-toml/v2"
)

const LOCAL_CREDS_FILE_PATH = "/Users/andrewserra/GithubProjects/kitchen/credentials.toml"
const LOCAL_DATACACHE_FILE_PATH = "/Users/andrewserra/GithubProjects/kitchen/datacache.toml"

const READ_WRITE_PERMISSIONS = 0644

// Data Cache Types
// ===========================
type LocalStreamData struct {
	Stream *models.Stream `toml:"stream" json:"stream"`
	// IsSynced bool           `toml:"is_synced" json:"is_synced"`
}

type LocalAccountDataCache struct {
	Streams []*LocalStreamData `toml:"streams"`
}

type LocalDataCache map[Email]LocalAccountDataCache

// Account and Credential Types
// ===========================
type Email = string

type LocalAccountData struct {
	ID           int       `toml:"id"`
	AccessToken  string    `toml:"access_token"`
	RefreshToken string    `toml:"refresh_token"`
	SavedAt      time.Time `toml:"saved_at"`
}

type LocalCredentials map[Email]LocalAccountData

// Credential Functions
// ===========================
func loadLocalCredentials() (LocalCredentials, error) {
	inData, err := os.ReadFile(LOCAL_CREDS_FILE_PATH)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LocalCredentials{}, nil
		}

		return LocalCredentials{}, err
	}

	var localData LocalCredentials
	err = toml.Unmarshal(inData, &localData)
	if err != nil {
		return LocalCredentials{}, err
	}

	return localData, nil
}

func saveLocalAccountData(localData LocalCredentials) error {
	data, err := toml.Marshal(localData)
	if err != nil {
		return err
	}
	return os.WriteFile(LOCAL_CREDS_FILE_PATH, data, READ_WRITE_PERMISSIONS)
}

func printSavedAccounts(localData LocalCredentials) {
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

func getAccessTokenForEmail(localData LocalCredentials, email Email) (string, error) {
	accountData, ok := localData[email]
	if !ok {
		return "", fmt.Errorf("account not found")
	}
	return accountData.AccessToken, nil
}

func getRefreshTokenForEmail(localData LocalCredentials, email Email) (string, error) {
	accountData, ok := localData[email]
	if !ok {
		return "", fmt.Errorf("account not found")
	}
	return accountData.RefreshToken, nil
}

// Data Cache Functions
// ===========================

func loadLocalDataCache() (LocalDataCache, error) {
	inData, err := os.ReadFile(LOCAL_DATACACHE_FILE_PATH)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LocalDataCache{}, nil
		}
		return LocalDataCache{}, err
	}

	var localData LocalDataCache
	err = json.Unmarshal(inData, &localData)
	if err != nil {
		return LocalDataCache{}, err
	}
	return localData, nil
}

func loadLocalAccountDataCache(email Email) (LocalAccountDataCache, error) {
	localData, err := loadLocalDataCache()
	if err != nil {
		return LocalAccountDataCache{}, err
	}

	accountData, ok := localData[email]
	if !ok {
		return LocalAccountDataCache{}, fmt.Errorf("account not found")
	}

	return accountData, nil
}

func saveLocalDataCache(localData LocalDataCache) error {
	data, err := json.Marshal(localData)
	if err != nil {
		return err
	}
	return os.WriteFile(LOCAL_DATACACHE_FILE_PATH, data, READ_WRITE_PERMISSIONS)
}

func saveLocalAccountDataCache(email Email, localAccountDataCache LocalAccountDataCache) error {
	localDataCache, err := loadLocalDataCache()
	if err != nil {
		return err
	}

	if _, ok := localDataCache[email]; !ok {
		return fmt.Errorf("account not found")
	}

	localDataCache[email] = localAccountDataCache
	return nil
}

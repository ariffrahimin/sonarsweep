package main

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	keychainService = "sonarsweep"
	keychainAccount = "user_token"
)

func StoreToken(token string) error {
	err := keyring.Set(keychainService, keychainAccount, token)
	if err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}
	return nil
}

func GetToken() (string, error) {
	token, err := keyring.Get(keychainService, keychainAccount)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", nil
		}
		return "", fmt.Errorf("failed to get token: %w", err)
	}
	return token, nil
}

func DeleteToken() error {
	err := keyring.Delete(keychainService, keychainAccount)
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete token: %w", err)
	}
	return nil
}
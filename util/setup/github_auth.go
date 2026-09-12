package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	githubKeyringService = "intercube-cli"
	githubKeyringAccount = "github-app-user-token"
)

type GitHubDeviceAuthorization struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type githubTokenResponse struct {
	AccessToken      string `json:"access_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func LoadGitHubToken() (string, error) {
	token, err := keyring.Get(githubKeyringService, githubKeyringAccount)
	if err == nil {
		return token, nil
	}
	path, pathErr := githubTokenFallbackPath()
	if pathErr != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	payload, fileErr := os.ReadFile(path)
	if errors.Is(fileErr, os.ErrNotExist) {
		return "", nil
	}
	if fileErr != nil {
		return "", fileErr
	}
	return strings.TrimSpace(string(payload)), nil
}

func SaveGitHubToken(token string) error {
	if err := keyring.Set(githubKeyringService, githubKeyringAccount, token); err == nil {
		return nil
	}
	path, err := githubTokenFallbackPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token), 0600)
}

func githubTokenFallbackPath() (string, error) {
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDirectory, "intercube", "github-app-token"), nil
}

func StartGitHubDeviceAuthorization(ctx context.Context, clientID string) (*GitHubDeviceAuthorization, error) {
	values := url.Values{"client_id": {clientID}}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://github.com/login/device/code",
		strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var result GitHubDeviceAuthorization
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}
	if response.StatusCode >= 300 || result.DeviceCode == "" {
		return nil, fmt.Errorf("GitHub device authorization failed with status %d", response.StatusCode)
	}
	return &result, nil
}

func PollGitHubDeviceAuthorization(
	ctx context.Context,
	clientID string,
	authorization *GitHubDeviceAuthorization,
) (string, error) {
	interval := time.Duration(max(authorization.Interval, 5)) * time.Second
	deadline := time.NewTimer(time.Duration(authorization.ExpiresIn) * time.Second)
	defer deadline.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-deadline.C:
			return "", errors.New("GitHub authorization expired")
		case <-time.After(interval):
		}

		values := url.Values{
			"client_id":   {clientID},
			"device_code": {authorization.DeviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		}
		request, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			"https://github.com/login/oauth/access_token",
			strings.NewReader(values.Encode()))
		if err != nil {
			return "", err
		}
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return "", err
		}
		var token githubTokenResponse
		decodeErr := json.NewDecoder(response.Body).Decode(&token)
		response.Body.Close()
		if decodeErr != nil {
			return "", decodeErr
		}
		if token.AccessToken != "" {
			return token.AccessToken, nil
		}
		switch token.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5 * time.Second
			continue
		case "expired_token":
			return "", errors.New("GitHub authorization expired")
		case "access_denied":
			return "", errors.New("GitHub authorization was cancelled")
		default:
			message := token.ErrorDescription
			if message == "" {
				message = token.Error
			}
			return "", fmt.Errorf("GitHub authorization failed: %s", message)
		}
	}
}

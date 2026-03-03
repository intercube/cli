package auth

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

var ErrNoSession = errors.New("no stored auth session")

type SessionStore struct {
	service string
	account string
	path    string
}

func NewSessionStore(service string) (*SessionStore, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	return &SessionStore{
		service: service,
		account: "clerk-session",
		path:    filepath.Join(configDir, "intercube", "session.json"),
	}, nil
}

func (s *SessionStore) Save(_ context.Context, session *Session) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}

	if err := keyring.Set(s.service, s.account, string(payload)); err == nil {
		return nil
	}

	return s.saveToFile(payload)
}

func (s *SessionStore) Load(_ context.Context) (*Session, error) {
	payload, err := keyring.Get(s.service, s.account)
	if err == nil {
		return decodeSession([]byte(payload))
	}

	session, fileErr := s.loadFromFile()
	if fileErr == nil {
		return session, nil
	}

	if errors.Is(fileErr, ErrNoSession) {
		return nil, ErrNoSession
	}

	return nil, fileErr
}

func (s *SessionStore) Clear(_ context.Context) error {
	if err := keyring.Delete(s.service, s.account); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}

	err := os.Remove(s.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}

func (s *SessionStore) saveToFile(payload []byte) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}

	return os.WriteFile(s.path, payload, 0600)
}

func (s *SessionStore) loadFromFile() (*Session, error) {
	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoSession
		}

		return nil, err
	}

	return decodeSession(payload)
}

func decodeSession(payload []byte) (*Session, error) {
	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return nil, err
	}

	if session.AccessToken == "" {
		return nil, ErrNoSession
	}

	return &session, nil
}

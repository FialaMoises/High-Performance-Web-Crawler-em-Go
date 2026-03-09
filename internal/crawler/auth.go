package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/FialaMoises/go-web-crawler/internal/config"
)

// Authenticator handles login and session management
type Authenticator struct {
	config     *config.Config
	client     *http.Client
	logger     *slog.Logger
	token      string
	tokenMu    sync.RWMutex
	isLoggedIn bool
	loginMu    sync.Mutex
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(cfg *config.Config, client *http.Client, logger *slog.Logger) *Authenticator {
	return &Authenticator{
		config:     cfg,
		client:     client,
		logger:     logger,
		isLoggedIn: false,
	}
}

// Login performs authentication based on the configured method
func (a *Authenticator) Login(ctx context.Context) error {
	a.loginMu.Lock()
	defer a.loginMu.Unlock()

	a.logger.Info("Attempting authentication", "method", a.config.AuthMethod, "url", a.config.LoginURL)

	var err error
	switch a.config.AuthMethod {
	case "form":
		err = a.loginForm(ctx)
	case "token":
		err = a.loginToken(ctx)
	default:
		return fmt.Errorf("unsupported auth method: %s", a.config.AuthMethod)
	}

	if err != nil {
		a.logger.Error("Authentication failed", "error", err)
		return err
	}

	a.isLoggedIn = true
	a.logger.Info("Authentication successful")
	return nil
}

// loginForm performs form-based authentication (POST with form data)
func (a *Authenticator) loginForm(ctx context.Context) error {
	// Prepare form data
	formData := url.Values{
		"username": {a.config.Username},
		"password": {a.config.Password},
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", a.config.LoginURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", a.config.UserAgent)

	// Send request
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Cookies are automatically stored in the client's cookie jar
	a.logger.Debug("Form-based login successful", "status", resp.StatusCode)
	return nil
}

// loginToken performs token-based authentication (JWT/Bearer)
func (a *Authenticator) loginToken(ctx context.Context) error {
	// Prepare JSON payload with common field names
	loginData := map[string]string{
		"username": a.config.Username,
		"password": a.config.Password,
		// Also include alternative field names for compatibility
		"userName": a.config.Username,
		"email":    a.config.Username,
	}

	// Add pageUrl field if LoginURL is set (required by some APIs like Holdprint)
	// Extract the base URL from the start URL or login URL
	if a.config.StartURL != "" {
		loginData["pageUrl"] = a.config.StartURL
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return fmt.Errorf("failed to marshal login data: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", a.config.LoginURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", a.config.UserAgent)

	// Send request
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		a.logger.Debug("Login failed", "status", resp.StatusCode, "body", string(body))
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	a.logger.Debug("Login response received", "status", resp.StatusCode, "body_length", len(body))

	var token string

	// Try to parse as JSON first (common case: {"token": "..."})
	var result struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
		AuthToken   string `json:"auth_token"`
	}

	if err := json.Unmarshal(body, &result); err == nil {
		// Successfully parsed as JSON, extract token from fields
		token = result.Token
		if token == "" {
			token = result.AccessToken
		}
		if token == "" {
			token = result.AuthToken
		}
		a.logger.Debug("Parsed token from JSON response", "token_length", len(token))
	} else {
		// Not JSON - assume the entire response body is the raw token (e.g., Holdprint API)
		token = strings.TrimSpace(string(body))
		a.logger.Debug("Using raw response body as token", "token_length", len(token))
	}

	if token == "" {
		return fmt.Errorf("no token found in login response")
	}

	// Store token
	a.tokenMu.Lock()
	a.token = token
	a.tokenMu.Unlock()

	a.logger.Debug("Token-based login successful", "token_length", len(token))
	return nil
}

// IsAuthenticated returns whether the authenticator has a valid session
func (a *Authenticator) IsAuthenticated() bool {
	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	return a.isLoggedIn
}

// GetToken returns the current authentication token (for token-based auth)
func (a *Authenticator) GetToken() string {
	a.tokenMu.RLock()
	defer a.tokenMu.RUnlock()
	return a.token
}

// AddAuthToRequest adds authentication headers/data to a request
func (a *Authenticator) AddAuthToRequest(req *http.Request) {
	if !a.config.RequiresAuth {
		return
	}

	if a.config.AuthMethod == "token" {
		token := a.GetToken()
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	// For form-based auth, cookies are automatically sent by the client
}

// HandleAuthError checks if an error is authentication-related and attempts re-login
func (a *Authenticator) HandleAuthError(ctx context.Context, statusCode int) (bool, error) {
	// Check if status code indicates auth failure
	if statusCode != http.StatusUnauthorized && statusCode != http.StatusForbidden {
		return false, nil
	}

	a.logger.Warn("Detected authentication failure, attempting re-login", "status_code", statusCode)

	// Mark as not logged in
	a.loginMu.Lock()
	a.isLoggedIn = false
	a.loginMu.Unlock()

	// Attempt re-login
	if err := a.Login(ctx); err != nil {
		return true, fmt.Errorf("re-authentication failed: %w", err)
	}

	return true, nil
}
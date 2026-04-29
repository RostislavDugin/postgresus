package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignInResponse struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Token  string `json:"token"`
}

type WorkspaceResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListWorkspacesResponse struct {
	Workspaces []WorkspaceResponse `json:"workspaces"`
}

// SignIn authenticates with the databasus API and returns a JWT token.
func SignIn(ctx context.Context, baseURL, email, password string) (string, error) {
	c := &DatabasusClient{
		httpClient: http.DefaultClient,
		baseURL:    baseURL,
	}

	body, statusCode, err := c.do(ctx, http.MethodPost, "/api/v1/users/signin", &SignInRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return "", fmt.Errorf("sign in request failed: %w", err)
	}

	if statusCode != http.StatusOK {
		return "", fmt.Errorf("sign in failed (status %d): %s", statusCode, string(body))
	}

	var resp SignInResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse sign in response: %w", err)
	}

	return resp.Token, nil
}

// ResolveWorkspace finds a workspace by name (or returns the first one if name is empty).
func (c *DatabasusClient) ResolveWorkspace(ctx context.Context, workspaceName string) (string, error) {
	body, statusCode, err := c.do(ctx, http.MethodGet, "/api/v1/workspaces", nil)
	if err != nil {
		return "", fmt.Errorf("list workspaces request failed: %w", err)
	}

	if statusCode != http.StatusOK {
		return "", fmt.Errorf("list workspaces failed (status %d): %s", statusCode, string(body))
	}

	var resp ListWorkspacesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse workspaces response: %w", err)
	}

	if len(resp.Workspaces) == 0 {
		return "", fmt.Errorf("no workspaces found for this user")
	}

	// If no name specified, use the first workspace
	if workspaceName == "" {
		return resp.Workspaces[0].ID, nil
	}

	for _, ws := range resp.Workspaces {
		if ws.Name == workspaceName {
			return ws.ID, nil
		}
	}

	return "", fmt.Errorf("workspace %q not found", workspaceName)
}

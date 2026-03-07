package extrahop

import (
	"context"
	"fmt"
)

// User represents an ExtraHop user.
type User struct {
	Username       string            `json:"username,omitempty"`
	Enabled        bool              `json:"enabled,omitempty"`
	Name           string            `json:"name,omitempty"`
	DateJoined     string            `json:"date_joined,omitempty"`
	LastUILogin    *string           `json:"last_ui_login_time,omitempty"`
	GrantedRoles   map[string]string `json:"granted_roles,omitempty"`
	EffectiveRoles map[string]string `json:"effective_roles,omitempty"`
	Password       string            `json:"password,omitempty"`
}

// UserGroup represents an ExtraHop user group.
type UserGroup struct {
	ID          int64             `json:"id,omitempty"`
	Name        string            `json:"name,omitempty"`
	DisplayName string            `json:"display_name,omitempty"`
	Enabled     bool              `json:"enabled,omitempty"`
	Source      string            `json:"source,omitempty"`
	Rights      map[string]string `json:"rights,omitempty"`
}

// UserService handles communication with user-related endpoints.
type UserService struct {
	client *Client
}

// List retrieves all users.
func (s *UserService) List(ctx context.Context) ([]*User, error) {
	var users []*User
	_, err := s.client.get(ctx, "/users", &users)
	return users, err
}

// Create creates a new user.
func (s *UserService) Create(ctx context.Context, user *User) error {
	_, err := s.client.post(ctx, "/users", user, nil)
	return err
}

// Get retrieves a specific user.
func (s *UserService) Get(ctx context.Context, username string) (*User, error) {
	var user User
	_, err := s.client.get(ctx, fmt.Sprintf("/users/%s", username), &user)
	return &user, err
}

// Update modifies a user.
func (s *UserService) Update(ctx context.Context, username string, user *User) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/users/%s", username), user)
	return err
}

// Delete deletes a user.
func (s *UserService) Delete(ctx context.Context, username string) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/users/%s", username))
	return err
}

// ListAPIKeys retrieves API keys for a user.
func (s *UserService) ListAPIKeys(ctx context.Context, username string) ([]map[string]interface{}, error) {
	var keys []map[string]interface{}
	_, err := s.client.get(ctx, fmt.Sprintf("/users/%s/apikeys", username), &keys)
	return keys, err
}

// UserGroupService handles communication with user group endpoints.
type UserGroupService struct {
	client *Client
}

// List retrieves all user groups.
func (s *UserGroupService) List(ctx context.Context) ([]*UserGroup, error) {
	var groups []*UserGroup
	_, err := s.client.get(ctx, "/usergroups", &groups)
	return groups, err
}

// Create creates a new user group.
func (s *UserGroupService) Create(ctx context.Context, group *UserGroup) error {
	_, err := s.client.post(ctx, "/usergroups", group, nil)
	return err
}

// Get retrieves a specific user group.
func (s *UserGroupService) Get(ctx context.Context, id int64) (*UserGroup, error) {
	var group UserGroup
	_, err := s.client.get(ctx, fmt.Sprintf("/usergroups/%d", id), &group)
	return &group, err
}

// Update modifies a user group.
func (s *UserGroupService) Update(ctx context.Context, id int64, group *UserGroup) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/usergroups/%d", id), group)
	return err
}

// Delete deletes a user group.
func (s *UserGroupService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/usergroups/%d", id))
	return err
}

// Refresh refreshes all user groups from LDAP/SAML.
func (s *UserGroupService) Refresh(ctx context.Context) error {
	_, err := s.client.post(ctx, "/usergroups/refresh", nil, nil)
	return err
}

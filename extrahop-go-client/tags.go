package extrahop

import (
	"context"
	"fmt"
)

// Tag represents an ExtraHop tag.
type Tag struct {
	ID      int64  `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	ModTime int64  `json:"mod_time,omitempty"`
}

// TagService handles communication with tag-related endpoints.
type TagService struct {
	client *Client
}

// List retrieves all tags.
func (s *TagService) List(ctx context.Context) ([]*Tag, error) {
	var tags []*Tag
	_, err := s.client.get(ctx, "/tags", &tags)
	return tags, err
}

// Create creates a new tag.
func (s *TagService) Create(ctx context.Context, tag *Tag) error {
	_, err := s.client.post(ctx, "/tags", tag, nil)
	return err
}

// Get retrieves a specific tag.
func (s *TagService) Get(ctx context.Context, id int64) (*Tag, error) {
	var tag Tag
	_, err := s.client.get(ctx, fmt.Sprintf("/tags/%d", id), &tag)
	return &tag, err
}

// Update modifies a tag.
func (s *TagService) Update(ctx context.Context, id int64, tag *Tag) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/tags/%d", id), tag)
	return err
}

// Delete deletes a tag.
func (s *TagService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/tags/%d", id))
	return err
}

// ListDevices retrieves devices with a specific tag.
func (s *TagService) ListDevices(ctx context.Context, id int64) ([]*Device, error) {
	var devices []*Device
	_, err := s.client.get(ctx, fmt.Sprintf("/tags/%d/devices", id), &devices)
	return devices, err
}

// AssignDevice assigns a tag to a device.
func (s *TagService) AssignDevice(ctx context.Context, tagID, deviceID int64) error {
	_, err := s.client.post(ctx, fmt.Sprintf("/tags/%d/devices/%d", tagID, deviceID), nil, nil)
	return err
}

// UnassignDevice removes a tag from a device.
func (s *TagService) UnassignDevice(ctx context.Context, tagID, deviceID int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/tags/%d/devices/%d", tagID, deviceID))
	return err
}

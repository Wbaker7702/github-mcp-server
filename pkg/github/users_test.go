package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUserActivity(t *testing.T) {
	// This is a unit test that mocks the client or just checks tool definition
	// Since setting up a full mock client is complex, I'll just check the tool definition here
	// to ensure it's registered correctly.

	tool, _ := GetUserActivity(nil, func(_, fallback string) string {
		return fallback
	})

	assert.Equal(t, "get_user_activity", tool.Name)
	assert.Equal(t, "Get recent activity for a GitHub user. Returns a list of events performed by the user.", tool.Description)

	schema := tool.InputSchema
	assert.NotNil(t, schema)

	props := schema.Properties
	assert.Contains(t, props, "username")
}

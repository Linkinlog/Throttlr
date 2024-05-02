package models_test

import (
	"testing"

	"github.com/linkinlog/throttlr/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewUser(t *testing.T) {
	u := models.NewUser().SetName("John Doe").SetEmail("jdoe@gmail.com").SetId("420-github")
	assert.Equal(t, "John Doe", u.Name)
	assert.Equal(t, "jdoe@gmail.com", u.Email)
	assert.Equal(t, u.Id, "420-github")
}

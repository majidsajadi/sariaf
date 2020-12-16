package sariaf_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/majidsajadi/sariaf"
	"github.com/stretchr/testify/assert"
)

func TestDuplicate(t *testing.T) {
	r := sariaf.New()

	assert.Nil(t, r.GET("/", http.NotFound))
	assert.True(t, errors.Is(r.GET("/", http.NotFound), sariaf.ErrRouterDuplicate))

	assert.Nil(t, r.GET("/:id", http.NotFound))
	assert.True(t, errors.Is(r.GET("/:id", http.NotFound), sariaf.ErrRouterDuplicate))
}

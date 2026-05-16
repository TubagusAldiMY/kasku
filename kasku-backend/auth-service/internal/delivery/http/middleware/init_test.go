package middleware_test

import "github.com/gin-gonic/gin"

func init() {
	// Set sekali di package load — hindari race di t.Parallel() tests.
	gin.SetMode(gin.TestMode)
}

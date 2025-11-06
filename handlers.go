package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello from Echo\n")
}

func health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func version(c echo.Context) error {
	info := BuildInfo()
	return c.JSON(http.StatusOK, info)
}

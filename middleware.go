package main

import (
	"log"
	"time"

	"github.com/labstack/echo/v4"
)

// exampleCustomMiddleware shows how to write a simple Echo middleware
func exampleCustomMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)
		dur := time.Since(start)
		log.Printf("%s %s %d %s", c.Request().Method, c.Path(), c.Response().Status, dur)
		return err
	}
}

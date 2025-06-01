package main

import (
	"github.com/gomsim/go-e2e"
	"github.com/gomsim/go-e2e/sequences"
	"github.com/gomsim/go-e2e/suites"
)

func main() {
	e2e.RunConcurrent(
		nil,

		suites.Auth,
		suites.Email,
		suites.Notifications,
		suites.Users,

		sequences.Login,
		sequences.RegisterUser,
		sequences.CreateEvent,
	)
}

// Package addr provides the AddressBook, a registry that maps environments to named
// addresses. It allows tests to resolve URLs at runtime based on the env argument
// passed to the e2e tool, making tests environment-agnostic.
package addr

import (
	"fmt"
	"os"
)

const (
	badArgument = 2
)

// The AddressBook is a nested map used to look up addresses at runtime based on the env
// argument passed to the e2e tool. The outer key is the environment, the inner key is the
// name of an address.
type AddressBook map[string]map[string]string

var addrs AddressBook

// Set registers the AddressBook to use for runtime lookups. It must be called from the
// init hook in the root of a test project.
//
// The outer key is the environment and the inner key is the name of an address.
//
// Eg.
//
//	addr.Set(addr.AddressBook{
//		"local": {
//			"auth":  "http://localhost:8080/api/v1/auth",
//			"users": "http://localhost:8081/api/v1/users",
//		},
//		"dev": {
//			"auth":  "https://auth.dev.example.com/api/v1/auth",
//			"users": "https://users.dev.example.com/api/v1/users",
//		},
//		"prod": {
//			"auth":  "https://auth.example.com/api/v1/auth",
//			"users": "https://users.example.com/api/v1/users",
//		},
//	})
func Set(book AddressBook) {
	addrs = book
}

// Lookup returns the address registered under the given name for the environment passed to
// the e2e tool. An AddressBook must have been registered with [Set] beforehand.
//
// Eg.
//
//	// Within a test
//	addr.Lookup("auth")
//
//	# On the command line
//	e2e ./mytests dev
//
// Returns the address registered under "auth" for the environment "dev".
func Lookup(name string) string {
	if os.Args[1] == "" {
		fmt.Printf("No env argument provided, pass one to use AddressBook lookups, e.g. e2e ./mytests dev\n")
		os.Exit(badArgument)
	}

	env := os.Args[1]

	addr, ok := addrs[env][name]
	if !ok {
		fmt.Printf("No address found for env %q, name %q\n", env, name)
		os.Exit(badArgument)
	}

	return addr
}

// EnvLookup works the same as [Lookup] but with the environment hard coded, ignoring whatever
// env argument was passed to the e2e tool.
//
// Eg.
//
//	// Within a test
//	addr.EnvLookup("dev", "auth")
//
// Returns the address registered under "auth" for the environment "dev", regardless of which
// environment the e2e tool was invoked with.
func EnvLookup(env, name string) string {
	if addr, ok := addrs[env][name]; !ok {
		panic(fmt.Sprintf("No address found for env %q, name %q", env, name))
	} else {
		return addr
	}
}

// Package addr provides the AddressBook, a registry that maps environments to named
// addresses. It allows tests to resolve URLs at runtime based on the env argument
// passed to the e2e tool, making tests environment-agnostic. The AddressBook itself is
// declared in the AddressBook field of the e2e.Config in the project root.
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
// name of an address. Declare it in the AddressBook field of the e2e.Config in the project
// root.
//
// Eg.
//
//	var Config = e2e.Config{
//		AddressBook: addr.AddressBook{
//			"local": {
//				"auth":  "http://localhost:8080/api/v1/auth",
//				"users": "http://localhost:8081/api/v1/users",
//			},
//			"dev": {
//				"auth":  "https://auth.dev.example.com/api/v1/auth",
//				"users": "https://users.dev.example.com/api/v1/users",
//			},
//		},
//	}
type AddressBook map[string]map[string]string

var addrs AddressBook

// Set registers the AddressBook to use for lookups. It's called by the e2e tool with the
// AddressBook from the e2e.Config in the project root, and isn't meant to be called directly.
func Set(book AddressBook) {
	addrs = book
}

// Lookup returns the address registered under the given name for the environment passed to
// the e2e tool. Lookup can only be performed if an AddressBook has been declared in the e2e.Config
// in the project root.
//
// Eg.
//
//	// Within a test
//	addr.Lookup("auth")
//
//	# On the command line
//	e2e ./mytests dev
//
// Returns the address registered under the name "auth" for the environment "dev".
func Lookup(name string) string {
	if addrs == nil {
		fmt.Println("Trying to perform lookup with no AddressBook set. Declare one in the AddressBook field of the e2e.Config in the project root\n")
		os.Exit(badArgument)
	}

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

// EnvLookup works the same as [Lookup] but with the environment hard coded in the test, ignoring
// whatever env argument was passed to the e2e tool.
//
// Eg.
//
//	// Within a test
//	addr.EnvLookup("dev", "auth")
//
// Returns the address registered under the name "auth" for the environment "dev", regardless of 
// which environment the e2e tool was invoked with.
func EnvLookup(env, name string) string {
	if addrs == nil {
		fmt.Println("Trying to perform lookup with no AddressBook set. Declare one in the AddressBook field of the e2e.Config in the project root\n")
		os.Exit(badArgument)
	}

	addr, ok := addrs[env][name]
	if !ok {
		fmt.Printf("No address found for env %q, name %q\n", env, name)
		os.Exit(badArgument)
	}

	return addr
}

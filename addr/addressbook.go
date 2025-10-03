package addr

import (
	"fmt"
	"os"
)

const (
	badArgument = 2
)

// The AddressBook is a singleton nested map used to store domains and paths. It's initiated once
// and used within test declarations to look up addresses based on the env parameter passed to `e2r`.
type AddressBook map[string]map[string]string

var addrs AddressBook

// Set registers an instance of AddressBook from which to make lookups during runtime. Set must be
// called from the init hook in the root of a test project.
//
// An AddressBook is a nested map with the outer layer representing environments and the nested
// later representing services.
//
// Eg.
//
//	AddressBook{
//		"local": {
//			"identification": "localhost:9999",
//			"authentication": "localhost:5555",
//		},
//		"dev": {
//			"identification": "dev.klick.klock",
//			"authentication": "dev.clack.cluck",
//		},
//	}
//
// .
func Set(book AddressBook) {
	addrs = book
}

// Lookup makes it possible to look up addresses durung runtime based on the env parameter passed
// to `e2r` if an AddressBook has been registered with [Set] at setup.
//
// Eg.
//
//	// Within a test
//	addr.Lookup("authentication")
//
//	# On the command line
//	e2r mytests dev
//
// This will perform a lookup for the address of "identification" for the environment "dev".
func Lookup(svc string) string {
	if os.Args[1] == "" {
		fmt.Printf("No env arg provided. Needed to run tests containing AddressBook lookups.\n")
		os.Exit(badArgument)
	}

	env := os.Args[1]

	addr, ok := addrs[env][svc]
	if !ok {
		fmt.Printf("Attempt access address for combination of env %q and svc %q that does not exist\n", env, svc)
		os.Exit(badArgument)
	}

	return addr
}

// EnvLookup makes it possible to look up addresses durung runtime if an AddressBook has been
// registered with [Set] at setup. EnvLookup works the same as Lookup but with the environment part
// being hard coded and overriding any env parameter passed to `e2r`.
//
// Eg.
//
//	// Within a test
//	addr.Lookup("dev", "authentication")
//
// This will perform a lookup for the address of "identification" for the environment "dev".
func EnvLookup(env, svc string) string {
	if addr, ok := addrs[env][svc]; !ok {
		panic(fmt.Sprintf("Attempt access address for combination of env %q and svc %q that does not exist", env, svc))
	} else {
		return addr
	}
}

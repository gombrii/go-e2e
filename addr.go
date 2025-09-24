package e2e

import (
	"fmt"
	"os"
)

const (
	badArgument = 2
)

type AddressBook map[string]map[string]string

var addrs AddressBook

func SetAddressBook(book AddressBook) {
	addrs = book
}

func Addr(svc string) string {
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

func EnvAddr(env, svc string) string {
	if addr, ok := addrs[env][svc]; !ok {
		panic(fmt.Sprintf("Attempt access address for combination of env %q and svc %q that does not exist", env, svc))
	} else {
		return addr
	}
}

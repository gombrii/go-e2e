package e2e

import (
	"fmt"
	"os"
)

var env = ""

type AddressBook map[string]map[string]string

var addrs AddressBook

func SetAddressBook(book AddressBook) {
	addrs = book
}

func Addr(svc string) string {
	if env == "" {
		env = os.Args[1]
	}

	if addr, ok := addrs[env][svc]; !ok {
		panic(fmt.Sprintf("Attempt access address for combination of env %q and svc %q that does not exist", env, svc))
	} else {
		return addr
	}
}

func EnvAddr(env, svc string) string {
	if addr, ok := addrs[env][svc]; !ok {
		panic(fmt.Sprintf("Attempt access address for combination of env %q and svc %q that does not exist", env, svc))
	} else {
		return addr
	}
}

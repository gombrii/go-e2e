package e2e

import (
	"fmt"
	"os"
)

var env = ""

type services map[string]string
type AddressBook map[string]services

var addrs AddressBook

func SetAddressBook(book AddressBook){
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

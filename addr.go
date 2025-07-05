package e2e

import (
	"encoding/json"
	"fmt"
	"os"
)

const errorExit = 1

const (
	envArg = 1
	mapArg = 2
)

var addrMap = map[string]map[string]string{}

func Addr(service string) string {
	if len(addrMap) == 0 {
		if err := json.Unmarshal([]byte(os.Args[mapArg]), &addrMap); err != nil {
			fmt.Printf("Error reading addresses: %v\n", err)
			os.Exit(errorExit)
		}
	}

	addr, ok := addrMap[os.Args[envArg]][service]
	if !ok {
		fmt.Printf("Error reading addresses: %v", fmt.Errorf("no address found for service %q and environment %q\n", service, os.Args[envArg]))
		os.Exit(errorExit)

	}

	return addr
}

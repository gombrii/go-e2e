package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var addrMap = map[string]map[string]string{}

func Addr(service string) string {
	if len(addrMap) == 0 {
		if err := json.Unmarshal([]byte(os.Args[2]), &addrMap); err != nil {
			fmt.Printf("Error reading addresses: %v", err)
			os.Exit(1)
		}
	}

	addr, ok := addrMap[strings.ToLower(os.Args[1])][strings.ToLower(service)]
	if !ok {
		fmt.Printf("Error reading addresses: %v", fmt.Errorf("no address found for service %q and environment %q", service, os.Args[1]))
		os.Exit(1)

	}

	return addr
}

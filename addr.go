package e2e

import (
	"os"
	"strings"
)

var EnvArg = ""

type AddrStore map[string]map[string]string

func Addrs() AddrStore {
	return AddrStore{}
}

func (r AddrStore) Reg(env, svc, baseAddr string) AddrStore {
	env = strings.ToLower(env)
	svc = strings.ToLower(svc)

	if _, ok := r[env]; !ok {
		r[env] = make(map[string]string)
	}

	r[env][svc] = baseAddr

	return r
}

func (r AddrStore) Get(env, svc, path string) string {
	env = strings.ToLower(env)
	svc = strings.ToLower(svc)

	return r[env][svc] + path
}

func Env() string {
	if EnvArg == "" {
		EnvArg = os.Args[1]
	}

	return EnvArg
}

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![GitHub tag](https://img.shields.io/github/v/tag/gomsim/go-e2e)](https://github.com/gomsim/go-e2e/tags)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/gomsim/go-e2e)

# Go-e2e
This is just a small library I mainly wrote over a couple of days one weekend to test my own HTTP APIs. It's only written for my own personal gain. It's not tested and it only supports my own narrow set of requrements. It has since been complemented with a CLI application making usage easier.

Go-e2e was written to be a quick and concurrent facilitator of HTTP API tests.

There are two parts to this projects, which are also located in two separate packages: e2e and e2r (main). e2e is the library used to define (and run) test cases while e2r contains the CLI appliction that is used to scan for test cases defined using e2e and initiate an execution.

## e2e
e2e is a library letting you define HTTP API tests. Tests are most easily run using the e2r command as explained later. But tests can also be run programmatically by creating and starting a runner. To do this create an empty go-module with a main fuction, create a `Runner` and call `Run` on a list of test sets you've declared yourself. 

```go
import (
	"github.com/gomsim/go-e2e"
)

func main() {
	e2e.Runner{}.Run(
		AuthSuite,
		EmailSuite,
		NotificationsSuite,
		UsersSuite,

		LoginSequence,
		RegisterUserSequence,
		CreateEventSequence,
	)
}
```

> There is an optional setup and teardown you can provide as functions in the construction of the Runner. This is good if the running of your tests for example need some environment variables set. These functions are typically called `BeforeRun` and `AfterRun` 

When you run your app you will be presented with a progress bar which when filled will give way to a result summary as well as a prompt giving you the option to see only the logs of failed tests cases or to see the logs of all performed tests (lots of text).

![Successful run](demo/image.png)

But what is a "test"?

### Tests
So the whole point of the library is its ability to run test cases. Each `Test` normally consists of at least a `Request` and an `Expect`. The request describes the details of a single HTTP call to be made. The expect describes expectations of the HTTP response. Tests which receive HTTP responses that don't meet the expectations count as failures.

```go
e2e.Test{
    Setup: e2e.Request{
        Method: "GET",
        URL:    "mydomain.com/ping",
    },
    Expect: e2e.Expect{
        Status: 200,
    },
}
```

### Suites
To make testing somewhat feasible and organized tests can be gathered in sets of type `Suite`. A suite has a name and is a set of independent named tests with no order.

```go
e2e.Suite{
	Name: "myService",
	Tests: e2e.Tests{
		"ping": {
			Request: e2e.Request{
				Method: "GET",
				URL:    "mydomain.com/ping",
			},
			Expect: e2e.Expect{
				Status: 200,
			},
		},
		"create": {
			Request: e2e.Request{
				Method: "POST",
				URL:    "mydomain.com/creatething",
			},
			Expect: e2e.Expect{
				Status: 201,
			},
		},
		"auth": {
			Request: e2e.Request{
				Method: "POST",
				URL:    "mydomain.com/login",
				Body:   `{"user": "username", "password": "password"}`,
			},
			Expect: e2e.Expect{
				Status: 200,
				Headers: e2e.Headers{
					{"Set-Cookie", "session_id=abc123xyz"},
				},
			},
		},
	},
}
```

### Sequences
Some tests require some setup. Or perhaps testing of one HTTP request requires information contained in the response to a different HTTP request. This is where type `Sequences` comes in. Sequences resemble suites in that they have a name and a collection of tests, but they differ in purpose. A sequence is unsurprisingly sequential meaning tests are run in the order they are declared. Tests within sequences work like steps. This is because tests, or steps, in a sequence are not indipendent but _interdependent_. They can take input and give output as well as perform pre test actions (tests in suites can also do this, but there is less incentive to do so). A bofore action can be two things, one of which is a manual input func (`Input`) declared within a step. It is useful when a step requires some external information in order to be performed, such as a pin code or some other information retrieved from a third source. When the tests are run the opportunity will be presented for the user to input the data as needed. The other before action is the ability for the step to run a terminal command (`Command`), such as a third party program, to for example expose a qr code, or such. Outputs from steps can be caught using a `Captor`. Captors are declared within a step to let it capture information contained within its HTTP response, such as an oid or URL, and let subsequent steps reference it to perform their own HTTP calls.

```go
e2e.Sequence{
	Name: "finger print - order flow",
	Steps: e2e.Steps{
		{
			Request: e2e.Request{
				Method:  "POST",
				URL:     "mydomain.com/fingerprint/create",
				Content: "application/json",
				Body:    `{"user": "MyUser", "phone": "010111000",}`,
			},
			Expect: e2e.Expect{
				Status: 201,
				Fields: e2e.Fields{
					"message": "OK"
				},
			},
		},
		{
			Before: e2e.Before{
				e2e.Input("finger print", "fingerprint"), // Propmpts the user for "finger print" and stores the input in a memory location called "fingerprint"
			},
			Request: e2e.Request{
				Method:  "POST",
				URL:     "mydomain.com/fingerprint/apply",
				Content: "application/json",
				Body:    `{"print": "$fingerprint"}`, // References the stored "fingerprint"
			},
			Expect: e2e.Expect{
				Status: 200,
				Fields: e2e.Fields{
					"token": "",
				},
			},
			Capture: e2e.Captors{"token"}, // Captures whatever was the value of the "token" field in the response body
		},
		{
			Request: e2e.Request{
				Method:  "POST",
				URL:     "mydomain.com/auth/token",
				Headers: e2e.Headers{{Key: "Authorization", Val: "Bearer $token"}}, // References the stored "token"
			},
			Expect: e2e.Expect{
				Status: 200,
				Fields: e2e.Fields{
					"url": "",
				},
			},
			Capture: e2e.Captors{"url"}, // Captures whatever was the value of the "url" field in the response body
		},
		{
			Request: e2e.Request{
				Method: "POST",
				URL:    "$url", // References the stored "url"
			},
			Expect: e2e.Expect{
				Status: 200,
			},
		},
	},
}
```

## e2r
e2r is a CLI app or CLI command really just eliminating the need to create a separate application to run tests manually. Perhaps it sounds unnecessary, but there is a good reason for it. When defining tests there are two types of changes that can be made between runs, changes related to the domain of what is being tested and changes that only concern _what_ is currently being tested. So an example of the former is a change to the API being tested. Maybe you add a test, or refine a test. These are things you want stored in code, with which e2e provides you the oppertunity. The latter type of change concern things like which ones of all your tests you want to run right now, or within which environment you want to run your tests. Dev? Prod?

The e2r cli command lets you define in code what the tests look like while letting you pass as arguments to the command what tests you want to run and within which environment.

### Getting started
To run the e2r command you first need to explicitly install it, even though you have already downloaded the library before.

```shell
go install github.com/gombrii/go-e2e/cmd/e2r@latest
```

You can then run it by standing in your project root, calling it and providing the path or pattern describing whatever packge of tests you want to run. The e2r command works just the same as the `go test` command in the way it interprets patterns. So if you want to run all tests in your module, simply provide it with `./...`

```shell
e2r ./...
```

e2r will look for any exported variable of type Sequence or Suite declared within any of the packages falling within the pattern provided to the cammand. e2r reads these declarations and generates and runs a temporary runnable that references these variables. The remporary runnable will be removed after being run.

### AddrStore
There is a second (optional) command that e2r currently takes, Env. This is to enable the possibility for the user to be able to be able to write tests once and run them for multiple different environments. It is not uncommon to want to test an API for multiple different environment, for example development, pre production and production. Eg:

```shell
e2r ./... dev
```

By providing the command with a second argument, the value of this argument will be available at runtime.

To gain access to the value passed as an argument the e2e library exposes a function `Env`. e2e also exposes a helper type `AddrStore`, instantiated via the exposed function `Addrs`. AddrStore is meant to be created in your project as a variable using a chained call that you can then reference in your tests. Eg.

```go
var Addrs = e2e.Addrs().
	Reg("local", "login", "localhost:9999").
	Reg("dev", "login", "dev.klick.klock")
```

Each call to `Reg` will register a new address which can be dynamically used in your tests. Reg takes three strings, the environment name for which you want this address to be active, the name of the service for which this address will be used for this environment, and lastly the actuall (base) address.

To access an address in a test declaration, instead of providing a string URL like normal you simply provide a call to the `Get` method in tour AddrStore, eg. `Addr.Get(e2e.Env(), "login", "/user")`. The first argument fetches the env argument passed to the e2r command, the second argument is the service and the third argument is whatever path you want to tack on to the address. if no path is needed simply pass `""`.

### Setup and teardown
As mentioned under the first example in [e2e](#e2e) the test runner can take as arguments a setup function and a teardown function. When running tests using e2r these can exist as well. The difference is that they'll not be provided anywhere. Instead they simply have to be declared and exported in the root package using the signatures `func BeforeRun() any` and `func AfterRun(any)` and they will both be automatically run before and after a test session respecively.

## Concurrency and performance
From my own manual testing it seems to scale pretty constantly and run whatever amount of tests in about a second, though it's only been tested on at most about 130 tests in one go.
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![GitHub tag](https://img.shields.io/github/v/tag/gombrii/go-e2e)](https://github.com/gombrii/go-e2e/tags)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/gombrii/go-e2e)

# Go-e2e
> Disclaimer: This is just a small library and an application I wrote to test my own HTTP APIs. It's only written for my own personal use and guarantees nothing. It's not tested and it only supports my own narrow set of requrements.

Go-e2e was written to be a quick and concurrent facilitator of HTTP API tests.

There are two parts to this projects, a library `e2e` and a CLI tool `e2r`. `e2e` is used to define test cases while `e2r` runs them.

## Getting started
The most minimal setup needed to run tests is a catalogue containing a `go.mod` file and one `.go` file. That setup will be used for this setup guide.

Run these commands.

```shell
mkdir mytests
cd mytests
go mod init
touch setup.go
```

You should then have this project catalogue.

```
mytests/
├── go.mod
└── setup.go
```

To create tests you will also need to depend on `github.com/gombrii/go-e2e`

```shell
# Inside catalogue mytests
go get github.com/gombrii/go-e2e@latest
```

To run tests you either have to install `e2r` or run it with `go run` using its install URL `github.com/gombrii/go-e2e/cmd/e2r`.

```shell
go install github.com/gombrii/go-e2e/cmd/e2r@latest
```

With this basic structure you can define tests that you run with `e2r`. The `setup.go` file can contain any setup as well as any number of tests. For reasons that will be described later in this guide, not least of all simply organisational, you might want to define multiple files in multiple packages.

Eg.
```
mytests/
├── go.mod
├── setup.go
├── smoketests/
│   ├── suite1.go
│   ├── suite2.go
│   └── suite3.go
└── manualtests/
    ├── suite1.go
    ├── suite2.go
    └── suite3.go
```

## e2r
You can use the `e2r` CLI tool to run tests you have defined in your project.

A test run can look something like this:

![Test run](demo/image.png)

### Usage

```
e2r <pattern> [env]
```

- `pattern` describes the location of the tests you want to run. It uses the same format as `go test`. To run all tests in the project pass `./...`. You can also run all tests in a package or all tests in a file by providing their respective paths, eg. `./smoketests` or `./smoketests/suite1.go` 
- `env` is an optional string value that if passed can be used for runtime lookups in the [`Addressbook`](#addressbook-optional) provided by the `e2e` library. This enables quick switching between testing base URLs specific to different environments.

Upon being run `e2r` will look for any exported variables of type [`Suite`](#suites) or [`Sequence`](#sequences) in the location targeted by the [`pattern`](#usage) provided and run them.

### Setup and teardown (optional)
There are two hooks that, if defined in the module root, will be run before and after each `e2r` run. These hooks can be used to perform any setup and/or teardown needed.

```go
func BeforeRun() any {
	// Any setup here
}

func AfterRun(any) {
	// Any teardown here 
}
```

For `e2r` to run them make sure to match their respective signatures exactly. Take note that they are exported. Whatever is returned by `BeforeRun` is what will be passed to `AfterRun` and can be accessed using a type assertion. If `BeforeRun` is not declared but `AfterRun` is, then `nil` will be passed.

### AddressBook (optional)
The `Addressbook` is a feature provided by `e2e` that enables runtime address lookup using a predefined addressbook in combination with the [`env`](#usage) parameter. This is to be able to make tests environment agnostic. Instead of an URL, a test will be targeted toward a service defined in the `Addressbook`. The `env` passed will then decide which instance of that service's URLs will be used.

`AddressBook` is a nested `map` which you can register with a call to `SetAddressBook` in the `init` hook in the project root.

```go
func init() {
	e2e.SetAddressBook(e2e.AddressBook{
		"local": {
			"authservice":    "https://localhost:8080/api/v1/auth",
			"userservice":    "https://localhost:8081/api/v1/users",
			"paymentservice": "https://localhost:8082/api/v1/pay",
		},
		"dev": {
			"authservice":    "https://dev.mysite-test.com/api/v1/auth",
			"userservice":    "https://dev.mysite-test.com/api/v1/users",
			"paymentservice": "https://dev.mysite-test.com/api/v1/pay",
		},
		"prod": {
			"authservice":    "https://mysite.com/api/v1/auth",
			"userservice":    "https://mysite.com/api/v1/users",
			"paymentservice": "https://mysite.com/api/v1/pay",
		},
	})
}
```

Having registred an `Addressbook` makes it possible to make lookups in tests like so `e2e.Addr("paymentservice")`. Paths can easily be appended using the plus operator.

```go
e2e.Addr("paymentservice") + "/creditcard"

// Alternatively e2e.EnvAddr can be used to override the `env` parameter
e2e.EnvAddr("dev", "paymentservice") + "/creditcard"
```

## e2e
The library needed to define tests consists of a single package `e2e`.

> Remember tests need to be declared in exported variables. The names of the variables do not matter.

### Tests
A test normally consists of at least a `Request` and an `Expect`. The `Request` defines a single HTTP request to be made. The `Expect` defines expectations of the HTTP response. Tests which receive HTTP responses that don't meet the expectations count as failures.

```go
{
    Request: e2e.Request{
        Method: "GET",
        URL:    "mydomain.com/ping",
    },
    Expect: e2e.Expect{
        Status: 200,
    },
}
```

There are many more parameters to a test.

```go
{
	Before:  e2e.Before{e2e.Input("password", "$pwd")}, // Advanced property
	Request: e2e.Request{
		Method:  "POST",
		URL:     "mydomain.com",
		CTX:     ctx,
		Headers: e2e.Headers{
			{Key: "Accept", Val: "application/json"},
		},
		Content: "application/json",
		Body:    `{"userId": "1", "pass": "$pwd"}`,
	},
	Expect: e2e.Expect{
		Status:  200,
		Body: e2e.Body{
			"userId":    1,
			"id":        1,
			"title":     "delectus aut autem",
			"completed": "false",
		},
		Headers: e2e.Headers{
			{Key: "Content-Type", Val: "application/json"}
		},
	},
	Capture: e2e.Captors{"completed"}, // Advanced property
}
```

In the `Expect` block only the parts included will be used to validate the HTTP response. If for example `Status` is left out any response status is concidered valid. For all components of the `Expect` block keys are required to match exactly while values only need to be part of the actual value.

Eg.
```go
Expect: e2e.Expect{
	Body: e2e.Body{
		"title": "delectus",
	},
},
```

In the above example the test would pass if the response body as a field "title" with a value of which "delectus" is a part. If title contained "delectus kolumplectus" the test would still pass. This is useful to be able to assert IDs that might contain some constant part and some dynamic part. However the key must match exactly for the test to pass. This makes it possible to simply test for the existance of a field without caring about the value by including `"title": ""`. The same goes for expected headers.

#### Advanced
`Before` and `Capture` are two special properties which enables actions to be performed before the execution of a test as well as response data to be captured.

`Before` takes a list of before-actions. There are two types created using the two helper functions `Input` and `Command`.

- `Input(text string, mapTo string)` will prompt the user to input a string value before the test is run. `text` is the prompt. `mapTo` is a key that can be referenced in the test using the `$`-prefix. In the example above `$pwd` is used to insert a password into the request body.
- `Command(command string, args ...string)` will run a terminal command before the test is run. Its output will be displayed to the user after which the user will be prompted to press enter to continue. Usecases include fetching some local dynamic data, displaying a QR code, or anything else might be performed.

The `Capture` property allows some data to be captured from the HTTP response in a test. This is discussed further in the [`Sequences`](#sequences) section.

### Suites
Tests can not exist on their own but must be put in a type of suite. There are two types `Suite` and `Sequence`. `Suite` is the simplest one. A `Suite` has a name and a set of independent named tests with no order.

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
A `Sequence` works similarly to a `Suite` but not exactly. Superficially the tests it contains are unnamed and are called steps. But importantly steps in a `Sequence` are run sequentially and in a common context. This means that data can be transferred from one step to the next and makes it possible to perform and test a chain of HTTP calls which build on eachother. The main mechanism to achieve this is the [captor](#advanced). A captor is a key listed in the `Capture` block of a test. If done the captor will capture the value of a field matching the catpr key in the body returned in the HTTP response in the test. The captured value can be referenced later in the `Sequence` using the `$`-prefix. This is the same mechanism used to capture and reference the input data from the [`Input`](#advanced) before-action. Captured values can be referenced in all parts of a test, even in before-actions. This means that a token returned in an HTTP response in a test can be referenced in a `Command` before-action in a later test to display a QR code, for example.

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
				Body: e2e.Body{
					"message": "OK"
				},
			},
		},
		{
			Before: e2e.Before{
				e2e.Input("finger print", "fingerprint"), // Propmpts the user for "finger print" and stores the input on the key "fingerprint"
			},
			Request: e2e.Request{
				Method:  "POST",
				URL:     "mydomain.com/fingerprint/apply",
				Content: "application/json",
				Body:    `{"print": "$fingerprint"}`, // References the captured "fingerprint"
			},
			Expect: e2e.Expect{
				Status: 200,
				Body: e2e.Body{
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
				Body: e2e.Body{
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

### Use Suite or Sequence?
Although they are similar they have some obvious and less obvious pros and cons respectively. The pros of Sequences are quite obvious in that they let tests share data between eachother. The drawback is that they run in sequence which is slower. Since tests in Suites are independent of eachother they can be run in parallell. If multiple Suites and Sequences are run in one go each Suite and Sequence will always run in parallell with eachother.

> Last tip: Since any [beofore-action](#advanced) will require user input when running the test it is a good idea to think about how tests are organized in packages and files. It can be useful to have a separate catalogue of tests that can be run as a smoke suite without needing user input. Tests that require user input can instead be used to test more intricate features of an API.

## Concurrency and performance
Since `go-e2e` is a concurrent tool tests don't scale linearly. From my own manual testing it seems to scale pretty constantly `O(1)` and run whatever amount of tests in about a second or two. `go-e2e` has been tested with at most about 370 tests.
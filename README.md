# infocenter

This application is infocenter backend service implementation. If implements POST and GET REST APIs
at URL `/infocenter/{topic}`. POST request sends a message to channel named as last segment of URL.
Message is posted as content of a POST request as plain text. GET request waits for message stream
on a specific channel specified by last segment of URL. Messages are sent in format of event stream
as specified on http://www.w3.org/TR/eventsource/.

## Installation

Installation requires two prerequisites `go` compiler (https://golang.org/) and
`dep` dependency manager (https://golang.github.io/dep/).

The application git repository should be first cloned to `$GOPATH/src/github.com/vaidasn/infocenter`

Then execute the following commands:

    $ dep ensure
    $ go install

Application should get installed into `$GOPATH/src` and executed using `$(go env GOPATH)/bin/infocenter`

## Usage

Invoke the application with option `--help` to get usage information:

    $ $(go env GOPATH)/bin/infocenter --help


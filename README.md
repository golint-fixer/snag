# Snag [![Build Status](https://travis-ci.org/Tonkpils/snag.svg?branch=wip)](https://travis-ci.org/Tonkpils/snag) [![Coverage Status](https://coveralls.io/repos/Tonkpils/snag/badge.svg?branch=coverage&service=github)](https://coveralls.io/github/Tonkpils/snag?branch=coverage)

An automatic build tool for all your needs

![](http://i.imgur.com/epcicvr.gif)

## Installation

If you have [go](http://golang.org/) installed and want to install
the latest and greatest you can run:

```go
$ go get github.com/Tonkpils/snag
```

If you do not have go installed on your machine, you can checkout
the [releases](https://github.com/Tonkpils/snag/releases) section to
download the binary for your platform.

## Usage

Snag works by reading a `.snag.yml` file which contains a set of
commands, ignored directories, and options.

As an example, the file with these contents:

```yml
script:
  - echo "hello world"
  - go test
ignore:
  - .git
verbose: true
```

will make snag run the commands `echo "hello world"` and `go test`,
ignoring changes in the `.git` directory, and returning output on success
through the `verbose` option.

Simply run:

```
snag
```

From a project with a `.snag.yml` file and develop away!

<br>

<div align="center">
    <img width="300px" src=".github/branding/logo.svg" alt="logo">
    <h3>An independent SurrealDB client for Go</h3>
</div>

<hr />

<p align="center">
    <a href="https://github.com/go-surreal/sdbc/releases"><img src="https://img.shields.io/github/v/release/go-surreal/sdbc" alt="release"></a>
    &nbsp;
    <a href="https://go.dev/doc/devel/release"><img src="https://img.shields.io/github/go-mod/go-version/go-surreal/sdbc?label=go" alt="go version"></a>
    &nbsp;
    <a href="https://goreportcard.com/report/github.com/go-surreal/sdbc"><img src="https://goreportcard.com/badge/github.com/go-surreal/sdbc" alt="go report card"></a>
    &nbsp;
    <a href="https://github.com/go-surreal/sdbc/actions/workflows/tests_codecov.yml"><img src="https://github.com/go-surreal/sdbc/actions/workflows/tests_codecov.yml/badge.svg?branch=main" alt="tests"></a>
    &nbsp;
    <a href="https://codecov.io/gh/go-surreal/sdbc"><img src="https://codecov.io/gh/go-surreal/sdbc/graph/badge.svg?token=AMR12YX5XU" alt="codecov"></a>
    &nbsp;
    <a href="https://discord.gg/surrealdb"><img src="https://img.shields.io/discord/902568124350599239?label=discord&color=5a66f6" alt="discord"></a>
    &nbsp;
    <img src="https://img.shields.io/github/contributors/go-surreal/sdbc" alt="contributors">
</p>

SDBC (**S**urreal**DB** **C**lient) is an independent Go client for the innovative [SurrealDB](https://surrealdb.com/) multi-model database system.

**DISCLAIMER**: SDBC is not the official Go client for SurrealDB but rather an independent implementation.
You can find the repository for the official implementation [here](https://github.com/surrealdb/surrealdb.go).
Currently, SDBC is designed for direct use with [SOM](https://github.com/go-surreal/som).
It's important to note that SDBC is in the early stages of development and is not yet stable or ready for production use.

## Table of Contents

- [What is SurrealDB?](#what-is-surrealdb)
- [Why SDBC instead of the official client?](#why-sdbc-instead-of-the-official-client)
- [Features](#features)
- [Getting Started](#getting-started)
  - [Installation](#installation)
  - [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)

## What is SurrealDB?

SurrealDB is a cutting-edge database system that offers a SQL-style query language with real-time queries  
and efficient related data retrieval. It supports both schema-full and schema-less data handling.
With its full graph database functionality, SurrealDB enables advanced querying and analysis by allowing 
records (or vertices) to be connected with edges, each with its own properties and metadata. 
This facilitates multi-table, multi-depth document retrieval without complex JOINs, all within the database.

*(Information extracted from the [official homepage](https://surrealdb.com))*.

## Why SDBC instead of the official client?

The official client can be found [here](https://github.com/surrealdb/surrealdb.go).

- The official Go client for SurrealDB is currently not in a really usable state.
- Inconsistencies exist in the codebase, such as the unused `url` parameter in the `New` function.
- It lacks essential features, particularly after both the 1.0 (first stable) and 2.0 releases of SurrealDB.
- The SurrealDB team has other priorities, and it seems as if they are currently not actively maintaining the Go client.
- Future versions of the official client may require CGO for direct bindings to an underlying driver, whereas SDBC will always be pure Go.
- Writing this custom client was and is an enjoyable endeavor 😉

SDBC is a practical choice until the official client becomes stable, actively maintained, and supports
all the features required by SOM. It also maintains purity in Go and avoids CGO dependencies.

It is still open whether this project will be maintained after the official client becomes stable
and usable for SOM.

## Features

- Pure Go implementation without CGO dependencies.
- Supports schema-full and schema-less data handling.
- Enables advanced querying and analysis with full graph database functionality.
- Designed for direct use with [SOM](https://github.com/go-surreal/som).

## Getting Started

### Installation

To install SDBC, run the following command:

```bash
go get github.com/go-surreal/sdbc
```

### Usage

To use SDBC, import it in your Go code:

```go
import (
	"github.com/go-surreal/sdbc"
)
```

Then, create a new client:

```go
func main() {
	client, err := sdbc.NewClient(ctx, sdbc.Config{
		Address:   "ws://localhost:8000/rpc", 
		Username:  "root", 
		Password:  "root", 
		Namespace: "test",
		Database:  "test",
	})
	
	if err != nil {
        log.Fatal(err)
    }
		
    // ...
}
```

## Contributing

We welcome contributions! If you'd like to contribute to SDBC, please read our
[Contributing Guidelines](https://github.com/go-surreal/sdbc/blob/main/CONTRIBUTING.md)
for instructions on how to get started.

## License

SDBC is licensed under the [MIT License](LICENSE).

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

### Details

#### Supported operations

This client implements the [RPC (websocket) interface](https://surrealdb.com/docs/surrealdb/integration/rpc) of SurrealDB.
The following operations are supported:

| Function                            | Description                                                                                              | Supported |
|-------------------------------------|----------------------------------------------------------------------------------------------------------|-----------|
| use [ ns, db ]                      | Specifies or unsets the namespace and/or database for the current connection                             |           |
| info                                | Returns the record of an authenticated record user                                                       |           |
| version                             | Returns version information about the database/server                                                    |           |
| signup  [ NS, DB, AC, … ]           | Signup a user using the SIGNUP query defined in a record access method                                   |           |
| signin   [NS, DB, AC, … ]           | Signin a root, NS, DB or record user against SurrealDB                                                   |           |
| authenticate [ token ]              | Authenticate a user against SurrealDB with a token                                                       |           |
| invalidate                          | Invalidate a user’s session for the current connection                                                   |           |
| let [ name, value ]                 | Define a variable on the current connection                                                              |           |
| unset [ name ]                      | Remove a variable from the current connection                                                            |           |
| live [ table, diff ]                | Initiate a live query                                                                                    |           |
| kill [ queryUuid ]                  | Kill an active live query                                                                                |           |
| query [ sql, vars ]                 | Execute a custom query with optional variables                                                           |           |
| graphql [ query, options? ]         | Execute GraphQL queries against the database                                                             |           |
| run [ func_name, version, args ]    | Execute built-in functions, custom functions, or machine learning models with optional arguments.        |           |
| select [ thing ]                    | Select either all records in a table or a single record                                                  |           |
| create [ thing, data ]              | Create a record with a random or specified ID                                                            |           |
| insert [ thing, data ]              | Insert one or multiple records in a table                                                                |           |
| insert_relation [ table, data ]     | Insert a new relation record into a specified table or infer the table from the data                     |           |
| update [ thing, data ]              | Modify either all records in a table or a single record with specified data if the record already exists |           |
| upsert [ thing, data ]              | Replace either all records in a table or a single record with specified data                             |           |
| relate [ in, relation, out, data? ] | Create graph relationships between created records                                                       |           |
| merge [ thing, data ]               | Merge specified data into either all records in a table or a single record                               |           |
| patch [ thing, patches, diff ]      | Patch either all records in a table or a single record with specified patches                            |           |
| delete [ thing ]                    | Delete either all records in a table or a single record                                                  |           |

#### Supported data types

This client supports the following [data types](https://surrealdb.com/docs/surrealql/datamodel#data-types) as defined by SurrealDB:

| Type     | Description                                                                                                                                                            | Definition                                                      | Supported | Go type                                  |
|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------|-----------|------------------------------------------|
| any      | The field will allow any data type supported by SurrealDB.                                                                                                             | (see others)                                                    | [x]       | any                                      |
| array    | An array of items. Content and length can be defined.                                                                                                                  | array, array<string>, array<int, 10>                            | [x]       | []any                                    |
| bool     | Describes whether something is truthy or not.                                                                                                                          | true, false                                                     | [x]       | bool                                     |
| bytes    | Stores a value in a byte array.                                                                                                                                        | bytes, <bytes>value                                             | [x]       | []byte                                   |
| datetime | An ISO 8601 compliant data type that stores a date with time and time zone.                                                                                            | (ISO 8601)                                                      | [x]       | time.Time                                |
| decimal  | Uses BigDecimal for storing any real number with arbitrary precision.                                                                                                  | -                                                               | [x]       | float64                                  |
| duration | Store a value representing a length of time.                                                                                                                           | 1h, 1m, 1h1m1s                                                  | [x]       | time.Duration                            |
| float    | Store a value in a 64 bit float.                                                                                                                                       | 1.5, 100.3                                                      | [x]       | float32, float64                         |
| geometry | RFC 7946 compliant data type for storing geometry in the GeoJson format.                                                                                               | [(see below)](#supported-geometry-types)                        | not yet   | [(see below)](#supported-geometry-types) |
| int      | Store a value in a 64 bit integer.                                                                                                                                     | 1, 2, 3, 4                                                      | [x]       | int                                      |
| number   | Store numbers without specifying the type. SurrealDB will store it using the minimal number of bytes.                                                                  | -                                                               | [x]       | int, float, ...                          |
| none     | ?                                                                                                                                                                      | -                                                               | [ ]       | -                                        |
| object   | Store formatted objects containing values of any supported type with no limit to object depth or nesting.                                                              | -                                                               | [x]       | struct{ ... }, `map[comparable]any`      |
| literal  | A value that may have multiple representations or formats, similar to an enum or a union type.<br>Can be composed of strings, numbers, objects, arrays, or durations.  | "a" \| "b", \[number, “abc”\], 123   \| 456 \| string \| 1y1m1d | kind of   | (any)                                    |
| option   | Makes types optional and guarantees the field to be either empty (NULL) or a value.                                                                                    | option<...>                                                     | [x]       | * (pointer)                              |
| range    | A range of possible values. Lower and upper bounds can be set, in the absence of which the range<br>becomes open-ended. A range of integers can be used in a FOR loop. | 0..10, 0..=10, ..10, 'a'..'z'                                   | [ ]       |                                          |
| record   | Store a reference to another record. The value must be a Record ID.                                                                                                    | record, record<user>, record<user \| administrator>             | [x]       | *sdbc.ID                                 |
| set      | A set of items. Similar to array, but items are automatically deduplicated.                                                                                            | set, set<string>, set<int, 10>                                  | [x]       | []any                                    |
| string   | Describes a text-like value.                                                                                                                                           | "some", "value"                                                 | [x]       | string                                   |

#### Supported geometry types

These types are not yet implemented.

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

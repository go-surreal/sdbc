<br>

<div align="center">
    <h3>SDBC - An independent SurrealDB client for Go</h3>
</div>

<hr />

<p align="center">
  <a href="https://go.dev/doc/devel/release">
    <img src="https://img.shields.io/badge/go-1.21.1-informational" alt="Go 1.21.1">
  </a>
  <a href="https://github.com/go-surreal/sdbc/actions/workflows/pull_request.yml">
    <img src="https://github.com/go-surreal/sdbc/actions/workflows/pull_request.yml/badge.svg" alt="PR">
  </a>
  <a href="https://discord.gg/surrealdb">
    <img src="https://img.shields.io/discord/902568124350599239?label=discord&color=5a66f6" alt="Discord">
  </a>
  <img src="https://img.shields.io/github/contributors/go-surreal/sdbc" alt="Contributors">
</p>

SDBC (**S**urreal**DB** **C**lient) is - as the name somewhat hints - a native Go client for the awesome [SurrealDB](https://surrealdb.com/).

DISCLAIMER: This is **NOT** the official client for Go, but an independent implementation.
You can find the repository for the official implementation [here](https://github.com/surrealdb/surrealdb-go).
Currently, SDBC is only meant for direct use with [SOM](https://github.com/go-surreal/som).
In general, it is in a very early stage of development and cannot be considered stable or ready for production use.

## What is SurrealDB?

SurrealDB is a relatively new database approach.
It provides a SQL-style query language with real-time queries and highly-efficient related data retrieval.
Both schemafull and schemaless handling of the data is possible.

With full graph database functionality, SurrealDB enables more advanced querying and analysis.
Records (or vertices) can be connected to one another with edges, each with its own record properties and metadata.
Simple extensions to traditional SQL queries allow for multi-table, multi-depth document retrieval, efficiently
in the database, without the use of complicated JOINs and without bringing the data down to the client.

*(Information extracted from the [official homepage]((https://surrealdb.com)))*

## Why is SDBC needed instead of the official client?

- The official go client for SurrealDB is not yet in a good state to work with
- There are a few inconsistencies in the codebase, e.g. the `url` param for the `New` function is completely unused
- It is missing a few important features (especially now that 1.0.0 is released)
- The SurrealDB team currently has more pressing topics to work on than the go client
- In the future the official client might be dependent on a base C/Rust driver which might require CGO. This custom client will stay 100% pure Go.
- It was (and still is) fun to write such a client myself 😉

It would definitely make sense to switch back to the official client once it is stable, actively maintained and supports all features required by SOM.
Furthermore, it should stay pure Go and not require CGO. Otherwise, the custom implementation will stay.

## Table of contents

tbd.

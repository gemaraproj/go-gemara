# go-gemara

Go SDK for parsing Gemara documents.

## Overview

This repository provides Go types and utilities for working with Gemara documents. The Go types are generated from CUE schemas published in the [Gemara CUE module](https://registry.cue.works/docs/github.com/gemaraproj/gemara@v0) (`github.com/gemaraproj/gemara@v0`) available in the [CUE Central Registry](https://registry.cue.works/).

## Regenerating Types

The `generated_types.go` file is generated from CUE schemas. To regenerate it:

1. Install CUE: `go install cuelang.org/go/cmd/cue@latest`
2. Run: `make generate`

This will fetch the latest schemas from the CUE module version `@v0` and regenerate the Go types.

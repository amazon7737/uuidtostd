package main

import "github.com/google/uuid" // want `can be replaced by the standard library package uuid`

func ID() uuid.UUID { return uuid.New() }

func main() { _ = ID() }

## Overview
The mongo package is a very simple wrapper around the labix.org/v2/mgo package. It's purpose is to allow you to do CRUD operations with very little code. It's not exhaustive and not meant to do everything for you.

## License
Mongo is licensed under the MIT license.

## Installation
To install mongo, simply run `go get github.com/sfreiberg/mongo`.

## Documentation
[GoDoc](http://godoc.org/github.com/sfreiberg/mongo)

## Example
```
package main

import (
	"github.com/sfreiberg/mongo"
	"labix.org/v2/mgo/bson"
)

type Customer struct {
	Id        bson.ObjectId `bson:"_id"`
	Firstname string
	Lastname  string
}

func init() {
	// Set server (localhost) and database (MyApp)
	err := mongo.SetServers("localhost", "MyApp")
	if err != nil {
		panic(err)
	}
}

func main() {
	customers := []interface{}{
		&Customer{
			Firstname: "George",
			Lastname:  "Jetson",
		},
		&Customer{
			Firstname: "Judy",
			Lastname:  "Jetson",
		},
	}

	err := mongo.Insert(customers...)
	if err != nil {
		panic(err)
	}
}
```

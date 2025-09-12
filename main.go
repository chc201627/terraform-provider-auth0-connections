package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Provider version
var version = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// TODO: Update this string with the published name of your provider.
		Address: "registry.terraform.io/cerifi/auth0-connections",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}

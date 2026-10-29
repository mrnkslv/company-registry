// @title           Company Registry API
// @version         1.0
// @description     CRUD companies
// @host            localhost:8080
// @BasePath        /
// @schemes         http
package main

import (
	"fmt"
	"os"

	"company-registry/internal/application"
)

var appVersion = "v0.0.0"

func main() {
	if err := application.New(appVersion).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

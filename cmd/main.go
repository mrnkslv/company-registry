// @title           Company Registry API
// @version         1.0
// @description     CRUD companies
// @host            localhost:8080
// @BasePath        /
// @schemes         http
package main

import (
	"company-registry/internal/application"
	"os"
)

var appVersion = "v0.0.0"

func main() {
	if err := application.New(appVersion).Run(); err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
}

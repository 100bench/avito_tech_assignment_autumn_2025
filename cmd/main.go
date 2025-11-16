package main

import (
	"log"

	"github.com/100bench/avito_tech_assignment_autumn_2025/app"
)

func main() {
	if err := app.RunApp(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

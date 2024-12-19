package main

import (
	"log"

	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

func main() {
	upgrades, err := checkupdates.Upgradable()
	if err != nil {
		log.Fatalf("Error checking for upgrades: %v", err)
	}

	log.Printf("Upgradable packages: %v", upgrades)
}

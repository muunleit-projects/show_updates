package main

import (
	"fmt"

	cu "github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

func main() {
	upgrades, err := cu.Upgradable()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(upgrades)
}

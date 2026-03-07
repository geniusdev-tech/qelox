package main

import (
"fmt"
"github.com/zeus/qelox/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Freeze timeout is: %v\n", cfg.Monitor.FreezeTimeoutMin)
}

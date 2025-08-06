package main

import (
	"fmt"
	"orderbook-pathfinder/internal/p2"
)

func main() {
	fmt.Println("=== Running P2: Virtual Orderbook Trading ===")
	p2.RunTestCasesFromFile("cmd/p2/testcases/specific_test_2.txt")
}

package main

import (
	"fmt"
	"orderbook-pathfinder/internal/p1"
)

func main() {
	fmt.Println("=== Running P1: Simple Trading Route Finder ===")
	p1.RunTestCasesFromFile("cmd/p1/testcases/specific_test_1.txt")
	// p1.RunTestCasesFromFile("cmd/p1/testcases/testcase1.txt")
}

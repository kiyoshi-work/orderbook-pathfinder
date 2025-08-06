package p1

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

type TradingPair struct {
	Base  string
	Quote string
	Ask   float64
	Bid   float64
}

type TradingRoute struct {
	Route []string
	Price float64
}

type Graph map[string]map[string]TradingPair

func FindOptimalTradingRoutes(baseCurrency, quoteCurrency string, pairs []TradingPair) (TradingRoute, TradingRoute) {
	graph := buildGraph(pairs)
	bestAskRoute := findBestRoute(graph, baseCurrency, quoteCurrency, true)
	bestBidRoute := findBestRoute(graph, baseCurrency, quoteCurrency, false)
	return bestAskRoute, bestBidRoute
}

func buildGraph(pairs []TradingPair) Graph {
	graph := make(Graph)

	for _, pair := range pairs {
		if graph[pair.Base] == nil {
			graph[pair.Base] = make(map[string]TradingPair)
		}
		if graph[pair.Quote] == nil {
			graph[pair.Quote] = make(map[string]TradingPair)
		}

		graph[pair.Base][pair.Quote] = pair
		reversePair := TradingPair{
			Base:  pair.Quote,
			Quote: pair.Base,
			Ask:   1.0 / pair.Bid,
			Bid:   1.0 / pair.Ask,
		}
		graph[pair.Quote][pair.Base] = reversePair
	}
	// // Visualize the trading graph in a readable format
	// fmt.Println("Trading Graph Visualization:")
	// for base, neighbors := range graph {
	// 	for quote, pair := range neighbors {
	// 		fmt.Printf("%s -> %s | Ask: %.8f, Bid: %.8f\n", base, quote, pair.Ask, pair.Bid)
	// 	}
	// }
	// fmt.Println(strings.Repeat("-", 50))

	return graph
}

func findBestRoute(graph Graph, start, end string, isAsk bool) TradingRoute {
	return bellmanFordWithLog(graph, start, end, isAsk)
}

func dijkstraWithMultiplication(graph Graph, start, end string, isAsk bool) TradingRoute {
	distances := make(map[string]float64)
	tracer := make(map[string]string)
	visited := make(map[string]bool)
	for node := range graph {
		distances[node] = math.Inf(1)
	}
	distances[start] = 1.0

	// Main Dijkstra loop
	for len(visited) < len(graph) {
		var current string
		minDist := math.Inf(1)
		for node := range graph {
			if !visited[node] && distances[node] < minDist {
				minDist = distances[node]
				current = node
			}
		}
		visited[current] = true
		if current == end || current == "" {
			break
		}
		for neighbor, pair := range graph[current] {
			if visited[neighbor] {
				continue
			}
			var weight float64
			if isAsk {
				weight = pair.Ask
			} else {
				weight = 1.0 / pair.Bid
			}
			newDist := distances[current] * weight
			if newDist < distances[neighbor] {
				distances[neighbor] = newDist
				tracer[neighbor] = current
			}
		}
	}

	// Reconstruct path
	if distances[end] == math.Inf(1) {
		return TradingRoute{
			Route: []string{},
			Price: 0,
		}
	}
	path := []string{}
	current := end
	for current != "" {
		path = append([]string{current}, path...)
		current = tracer[current]
	}
	return TradingRoute{
		Route: path,
		Price: distances[end],
	}
}

func bellmanFordWithLog(graph Graph, start, end string, isAsk bool) TradingRoute {
	distances := make(map[string]float64)
	tracer := make(map[string]string)
	for node := range graph {
		distances[node] = math.Inf(1)
	}
	distances[start] = 0

	for i := 0; i < len(graph)-1; i++ {
		for u := range graph {
			for v, pair := range graph[u] {
				var weight float64
				if isAsk {
					weight = pair.Ask
				} else {
					weight = 1.0 / pair.Bid
				}
				logWeight := math.Log(weight)
				if distances[u] != math.Inf(1) && distances[u]+logWeight < distances[v] {
					distances[v] = distances[u] + logWeight
					tracer[v] = u
				}
			}
		}
	}

	// Check for negative cycles
	for u := range graph {
		for v, pair := range graph[u] {
			var weight float64
			if isAsk {
				weight = pair.Ask
			} else {
				weight = 1.0 / pair.Bid
			}
			logWeight := math.Log(weight)

			if distances[u] != math.Inf(1) && distances[u]+logWeight < distances[v] {
				fmt.Printf("Warning: Negative cycle detected involving %s -> %s\n", u, v)
			}
		}
	}

	// Reconstruct path
	if distances[end] == math.Inf(1) {
		return TradingRoute{
			Route: []string{},
			Price: 0,
		}
	}
	path := []string{}
	current := end
	pathLength := 0
	for current != "" && pathLength < len(graph) { // Prevent infinite loop
		path = append([]string{current}, path...)
		current = tracer[current]
		pathLength++
	}

	var finalPrice float64
	if isAsk {
		finalPrice = math.Exp(distances[end])
		// NOTE: Reverse the path to get the correct route
		for i := 0; i < len(path)/2; i++ {
			path[i], path[len(path)-i-1] = path[len(path)-i-1], path[i]
		}
		return TradingRoute{
			Route: path,
			Price: finalPrice,
		}
	} else {
		finalPrice = math.Exp(-distances[end])
		return TradingRoute{
			Route: path,
			Price: finalPrice,
		}
	}
}

func formatRoute(route []string) string {
	return strings.Join(route, "->")
}

func printOutput(bestBidRoute, bestAskRoute TradingRoute) {
	// Best ask route (buying base currency)
	fmt.Println(formatRoute(bestAskRoute.Route))

	// Best ask price (buying base currency)
	fmt.Printf("%.8f\n", bestAskRoute.Price)

	// Best bid route (selling base currency)
	fmt.Println(formatRoute(bestBidRoute.Route))

	// Best bid price (selling base currency)
	fmt.Printf("%.8f\n", bestBidRoute.Price)
}

func runTestCase(input string) {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) < 2 {
		fmt.Printf("Invalid test case format: not enough lines\n")
		return
	}

	// Parse first line
	parts := strings.Fields(lines[0])
	if len(parts) < 2 {
		fmt.Printf("Invalid test case format: first line should have 2 currencies\n")
		return
	}
	baseCurrency := parts[0]
	quoteCurrency := parts[1]

	n, err := strconv.Atoi(lines[1])
	if err != nil {
		fmt.Printf("Invalid test case format: second line should be a number\n")
		return
	}

	var pairs []TradingPair
	for i := 0; i < n; i++ {
		line := lines[2+i]
		parts := strings.Fields(line)

		if len(parts) < 4 {
			fmt.Printf("Invalid trading pair format at line %d: %s\n", 2+i+1, line)
			continue
		}

		base := parts[0]
		quote := parts[1]
		ask, _ := strconv.ParseFloat(parts[2], 64)
		bid, _ := strconv.ParseFloat(parts[3], 64)

		pair := TradingPair{
			Base:  base,
			Quote: quote,
			Ask:   ask,
			Bid:   bid,
		}
		pairs = append(pairs, pair)
	}

	// Find optimal routes
	bestAskRoute, bestBidRoute := FindOptimalTradingRoutes(baseCurrency, quoteCurrency, pairs)

	// Print results
	fmt.Printf("Test Case: %s -> %s\n", baseCurrency, quoteCurrency)
	printOutput(bestBidRoute, bestAskRoute)
	fmt.Println("---")
}

func RunTestCasesFromFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentTestCase strings.Builder
	testCaseCount := 0
	passedCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "#") || strings.TrimSpace(line) == "" {
			continue
		}

		// Check if this is a new test case (two currencies separated by space)
		parts := strings.Fields(line)
		if len(parts) == 2 {
			isCurrencyPair := true
			for _, part := range parts {
				for _, char := range part {
					if char < 'A' || char > 'Z' {
						isCurrencyPair = false
						break
					}
				}
				if !isCurrencyPair {
					break
				}
			}

			if isCurrencyPair {
				// Process previous test case if exists
				if currentTestCase.Len() > 0 {
					testCaseCount++
					fmt.Printf("=== Test Case %d ===\n", testCaseCount)
					runTestCase(currentTestCase.String())
					passedCount++
				}

				// Start new test case
				currentTestCase.Reset()
				currentTestCase.WriteString(line)
				currentTestCase.WriteString("\n")
				continue
			}
		}
		currentTestCase.WriteString(line)
		currentTestCase.WriteString("\n")
	}
	if currentTestCase.Len() > 0 {
		testCaseCount++
		fmt.Printf("=== Test Case %d ===\n", testCaseCount)
		runTestCase(currentTestCase.String())
		passedCount++
	}

	fmt.Printf("\n=== SUMMARY ===\n")
	fmt.Printf("Total test cases: %d\n", testCaseCount)
	fmt.Printf("Passed: %d\n", passedCount)
	fmt.Printf("Failed: %d\n", testCaseCount-passedCount)
}

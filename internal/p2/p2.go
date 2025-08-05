package p2

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

type Level struct {
	Price  float64
	Amount float64
}

type TradingPair struct {
	Base      string
	Quote     string
	AskOrders []Level
	BidOrders []Level
}

type Graph map[string]map[string]TradingPair

type VirtualLevel struct {
	Price  float64
	Amount float64
	Route  []string
}

type VirtualTradingPair struct {
	Base      string
	Quote     string
	AskOrders []VirtualLevel
	BidOrders []VirtualLevel
}

type BestRoute []struct {
	Route  []string
	Price  float64
	Amount float64
}

type PriceVolumeCombo struct {
	prices []float64
	depth  float64
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
			Base:      pair.Quote,
			Quote:     pair.Base,
			AskOrders: invertOrders(pair.BidOrders),
			BidOrders: invertOrders(pair.AskOrders),
		}
		graph[pair.Quote][pair.Base] = reversePair
	}
	// fmt.Println("Trading Graph Visualization:")
	// for base, neighbors := range graph {
	// 	for quote, pair := range neighbors {
	// 		fmt.Printf("%s -> %s | Ask Orders: %d, Bid Orders: %d\n",
	// 			base, quote, len(pair.AskOrders), len(pair.BidOrders))
	// 		if len(pair.AskOrders) > 0 {
	// 			fmt.Printf("  Best Ask: %.8f\n", pair.AskOrders[0].Price)
	// 		}
	// 		if len(pair.BidOrders) > 0 {
	// 			fmt.Printf("  Best Bid: %.8f\n", pair.BidOrders[0].Price)
	// 		}
	// 	}
	// }
	// fmt.Println(strings.Repeat("-", 50))
	return graph
}

func invertOrders(orders []Level) []Level {
	var invertedOrders []Level
	for _, order := range orders {
		if order.Price > 0 {
			invertedOrders = append(invertedOrders, Level{
				Price:  1.0 / order.Price,
				Amount: order.Amount * order.Price,
			})
		}
	}
	return invertedOrders
}

func findAllPaths(graph Graph, start, end string, maxDepth int) [][]string {
	var allPaths [][]string
	visited := make(map[string]bool)
	currentPath := []string{start}
	dfsAllPaths(graph, start, end, visited, currentPath, &allPaths, maxDepth)
	return allPaths
}

func dfsAllPaths(graph Graph, current, end string, visited map[string]bool, currentPath []string, allPaths *[][]string, maxDepth int) {
	if len(currentPath) > maxDepth {
		return
	}
	if current == end && len(currentPath) > 1 {
		pathCopy := make([]string, len(currentPath))
		copy(pathCopy, currentPath)
		*allPaths = append(*allPaths, pathCopy)
		return
	}
	visited[current] = true
	defer func() { visited[current] = false }()
	for neighbor := range graph[current] {
		if !visited[neighbor] {
			newPath := append(currentPath, neighbor)
			dfsAllPaths(graph, neighbor, end, visited, newPath, allPaths, maxDepth)
		}
	}
}

func printBestRouteOutput(bestBidRoute, bestAskRoute BestRoute) {
	fmt.Println("=== Best Routes Found ===")
	fmt.Println("ASK Routes:")
	if len(bestAskRoute) > 0 {
		for i, route := range bestAskRoute {
			fmt.Printf("  %d. %s (Price: %.8f, Amount: %.8f %s)\n",
				i+1, formatRoute(route.Route), route.Price, route.Amount, route.Route[0])
		}
	} else {
		fmt.Println("  NO_ROUTE")
	}
	fmt.Println("BID Routes:")
	if len(bestBidRoute) > 0 {
		for i, route := range bestBidRoute {
			fmt.Printf("  %d. %s (Price: %.8f, Amount: %.8f %s)\n",
				i+1, formatRoute(route.Route), route.Price, route.Amount, route.Route[0])
		}
	} else {
		fmt.Println("  NO_ROUTE")
	}

	fmt.Println("---")
}

func formatRoute(route []string) string {
	return strings.Join(route, "->")
}

func buildVirtualOrderbook(graph Graph, baseCurrency, quoteCurrency string) VirtualTradingPair {
	virtualPair := VirtualTradingPair{
		Base:      baseCurrency,
		Quote:     quoteCurrency,
		AskOrders: []VirtualLevel{},
		BidOrders: []VirtualLevel{},
	}
	paths := findAllPaths(graph, baseCurrency, quoteCurrency, 5)
	fmt.Printf("Found %d paths for %s->%s\n: %v\n", len(paths), baseCurrency, quoteCurrency, paths)
	for _, path := range paths {
		askOrders := calculateOrdersFromPath(graph, path, true)
		bidOrders := calculateOrdersFromPath(graph, path, false)
		virtualPair.AskOrders = append(virtualPair.AskOrders, askOrders...)
		virtualPair.BidOrders = append(virtualPair.BidOrders, bidOrders...)
	}
	sortVirtualOrders(&virtualPair.AskOrders, true)
	sortVirtualOrders(&virtualPair.BidOrders, false)
	virtualPair.AskOrders = mergeVirtualOrders(virtualPair.AskOrders)
	virtualPair.BidOrders = mergeVirtualOrders(virtualPair.BidOrders)
	return virtualPair
}

func calculateOrdersFromPath(graph Graph, path []string, isAsk bool) []VirtualLevel {
	var orders []VirtualLevel
	if len(path) < 2 {
		return orders
	}
	priceVolumeCombos := getAllPriceVolumeCombinations(graph, path, isAsk)

	truePath := make([]string, len(path))
	// NOTE: for ask, the path needs to be reversed
	if isAsk {
		for i, token := range path {
			truePath[len(path)-1-i] = token
		}
	} else {
		truePath = path
	}
	for _, combo := range priceVolumeCombos {
		effectivePrice := 1.0
		for _, price := range combo.prices {
			effectivePrice *= price
		}
		if combo.depth > 0 {
			orders = append(orders, VirtualLevel{
				Price:  effectivePrice,
				Amount: combo.depth,
				Route:  truePath,
			})
		}
	}
	return orders
}

func getAllPriceVolumeCombinations(graph Graph, path []string, isAsk bool) []PriceVolumeCombo {
	var combinations []PriceVolumeCombo
	for i := 0; i < len(path)-1; i++ {
		fromToken := path[i]
		toToken := path[i+1]
		pair, exists := graph[fromToken][toToken]
		if !exists {
			return combinations
		}
		var priceVolumes []struct {
			price  float64
			volume float64
		}

		if isAsk {
			for _, order := range pair.AskOrders {
				priceVolumes = append(priceVolumes, struct {
					price  float64
					volume float64
				}{order.Price, order.Amount})
			}
		} else {
			for _, order := range pair.BidOrders {
				priceVolumes = append(priceVolumes, struct {
					price  float64
					volume float64
				}{order.Price, order.Amount})
			}
		}

		if i == 0 {
			// First pair, initialize combinations
			for _, pv := range priceVolumes {
				combinations = append(combinations, PriceVolumeCombo{
					prices: []float64{pv.price},
					depth:  pv.volume,
				})
			}
		} else {
			// Extend existing combinations
			var newCombinations []PriceVolumeCombo
			for _, combo := range combinations {
				for _, pv := range priceVolumes {
					newPrices := make([]float64, len(combo.prices))
					copy(newPrices, combo.prices)
					newPrices = append(newPrices, pv.price)

					// Calculate new max volume (minimum of existing and new)
					newdepth := math.Min(combo.depth, pv.volume)

					newCombinations = append(newCombinations, PriceVolumeCombo{
						prices: newPrices,
						depth:  newdepth,
					})
				}
			}
			combinations = newCombinations
		}
	}

	return combinations
}

func sortVirtualOrders(orders *[]VirtualLevel, ascending bool) {
	if ascending {
		// Sort asks in ascending order (lowest price first)
		for i := 0; i < len(*orders)-1; i++ {
			for j := i + 1; j < len(*orders); j++ {
				if (*orders)[i].Price > (*orders)[j].Price {
					(*orders)[i], (*orders)[j] = (*orders)[j], (*orders)[i]
				}
			}
		}
	} else {
		// Sort bids in descending order (highest price first)
		for i := 0; i < len(*orders)-1; i++ {
			for j := i + 1; j < len(*orders); j++ {
				if (*orders)[i].Price < (*orders)[j].Price {
					(*orders)[i], (*orders)[j] = (*orders)[j], (*orders)[i]
				}
			}
		}
	}
}

func mergeVirtualOrders(orders []VirtualLevel) []VirtualLevel {
	if len(orders) == 0 {
		return orders
	}
	var merged []VirtualLevel
	current := orders[0]
	for i := 1; i < len(orders); i++ {
		if math.Abs(orders[i].Price-current.Price) < 1e-8 {
			// Same price, merge quantities
			current.Amount += orders[i].Amount
			if orders[i].Price < current.Price {
				current.Route = orders[i].Route
			}
		} else {
			merged = append(merged, current)
			current = orders[i]
		}
	}
	merged = append(merged, current)
	return merged
}

func printVirtualOrderbook(virtualPair VirtualTradingPair) {
	fmt.Printf("%s %s\n", virtualPair.Base, virtualPair.Quote)
	fmt.Printf("%d\n", len(virtualPair.AskOrders))

	for _, order := range virtualPair.AskOrders {
		fmt.Printf("%.8f %.0f (%s)\n", order.Price, order.Amount, formatRoute(order.Route))
	}

	fmt.Printf("%d\n", len(virtualPair.BidOrders))

	for _, order := range virtualPair.BidOrders {
		fmt.Printf("%.8f %.0f (%s)\n", order.Price, order.Amount, formatRoute(order.Route))
	}
}

func executeOnVirtualOrderbook(virtualPair VirtualTradingPair, targetAmount float64) (BestRoute, BestRoute) {
	// Find best ask route (buying base with quote)
	bestAskRoute := findBestRouteFromVirtualOrderbook(virtualPair.AskOrders, targetAmount, true)

	// Find best bid route (selling base for quote)
	bestBidRoute := findBestRouteFromVirtualOrderbook(virtualPair.BidOrders, targetAmount, false)

	return bestAskRoute, bestBidRoute
}

func findBestRouteFromVirtualOrderbook(orders []VirtualLevel, targetAmount float64, isAsk bool) BestRoute {
	if len(orders) == 0 {
		return BestRoute{}
	}

	var bestRoute BestRoute
	bestRoute = make(BestRoute, 0)

	remainingAmount := targetAmount
	var totalCost float64
	var executedAmount float64

	// fmt.Printf("=== %s Execution (Target: %.0f) ===\n",
	// 	func() string {
	// 		if isAsk {
	// 			return "ASK"
	// 		} else {
	// 			return "BID"
	// 		}
	// 	}(), targetAmount)

	for _, order := range orders {
		if remainingAmount <= 0 {
			break
		}

		executed := math.Min(remainingAmount, order.Amount)
		executedAmount += executed

		if isAsk {
			cost := executed * order.Price
			totalCost += cost
			// fmt.Printf("Level %d: %.8f %s at %.8f ETH (Route: %s) - Cost: %.8f ETH\n",
			// 	i+1, executed, "KNC", order.Price, formatRoute(order.Route), cost)
		} else {
			received := executed * order.Price
			totalCost += received
			// fmt.Printf("Level %d: %.8f %s at %.8f ETH (Route: %s) - Received: %.8f ETH\n",
			// 	i+1, executed, "KNC", order.Price, formatRoute(order.Route), received)
		}

		remainingAmount -= executed

		// Add this route to the execution list
		bestRoute = append(bestRoute, struct {
			Route  []string
			Price  float64
			Amount float64
		}{
			Route:  order.Route,
			Price:  order.Price,
			Amount: executed,
		})
	}

	// // Calculate effective price if we executed some amount
	// effectivePrice := 0.0
	// if executedAmount > 0 && targetAmount > 0 {
	// 	effectivePrice = totalCost / targetAmount
	// }
	// fmt.Printf("Total executed: %.8f, Effective price: %.8f\n", executedAmount, effectivePrice)
	// fmt.Printf("Routes executed (%d):\n", len(bestRoute))
	// for i, route := range bestRoute {
	// 	fmt.Printf("  %d. %s (%.8f ETH, %.8f KNC)\n", i+1, formatRoute(route.Route), route.Price, route.Amount)
	// }
	// fmt.Println("---")

	return bestRoute
}

func parseOrderBook(lines []string, lineIdx *int, orderType, pairBase, pairQuote string) ([]Level, error) {
	if *lineIdx >= len(lines) {
		return nil, fmt.Errorf("missing %s orders count for pair %s/%s", orderType, pairBase, pairQuote)
	}
	count, err := strconv.Atoi(strings.TrimSpace(lines[*lineIdx]))
	if err != nil {
		return nil, fmt.Errorf("invalid %s orders count: %s", orderType, lines[*lineIdx])
	}
	*lineIdx++
	var orders []Level
	for j := 0; j < count; j++ {
		if *lineIdx >= len(lines) {
			return nil, fmt.Errorf("missing %s order %d for pair %s/%s", orderType, j+1, pairBase, pairQuote)
		}
		orderParts := strings.Fields(lines[*lineIdx])
		if len(orderParts) < 2 {
			return nil, fmt.Errorf("invalid %s order format: %s", orderType, lines[*lineIdx])
		}
		price, _ := strconv.ParseFloat(orderParts[0], 64)
		amount, _ := strconv.ParseFloat(orderParts[1], 64)
		orders = append(orders, Level{Price: price, Amount: amount})
		*lineIdx++
	}
	return orders, nil
}

func runTestCase(input string) {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) < 2 {
		fmt.Printf("Invalid test case format: not enough lines\n")
		return
	}
	parts := strings.Fields(lines[0])
	if len(parts) < 3 {
		fmt.Printf("Invalid test case format: first line should have base quote amount\n")
		return
	}
	baseCurrency := parts[0]
	quoteCurrency := parts[1]
	amount, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		fmt.Printf("Invalid amount: %s\n", parts[2])
		return
	}
	n, err := strconv.Atoi(lines[1])
	if err != nil {
		fmt.Printf("Invalid test case format: second line should be a number\n")
		return
	}
	var pairs []TradingPair
	lineIdx := 2
	for i := 0; i < n; i++ {
		if lineIdx >= len(lines) {
			fmt.Printf("Not enough lines for pair %d\n", i+1)
			return
		}
		pairParts := strings.Fields(lines[lineIdx])
		if len(pairParts) < 2 {
			fmt.Printf("Invalid pair format at line %d: %s\n", lineIdx+1, lines[lineIdx])
			return
		}
		pairBase := pairParts[0]
		pairQuote := pairParts[1]
		lineIdx++
		askOrders, err := parseOrderBook(lines, &lineIdx, "ask", pairBase, pairQuote)
		if err != nil {
			fmt.Printf("Error parsing ask orders: %v\n", err)
			return
		}
		bidOrders, err := parseOrderBook(lines, &lineIdx, "bid", pairBase, pairQuote)
		if err != nil {
			fmt.Printf("Error parsing bid orders: %v\n", err)
			return
		}
		pair := TradingPair{
			Base:      pairBase,
			Quote:     pairQuote,
			AskOrders: askOrders,
			BidOrders: bidOrders,
		}
		pairs = append(pairs, pair)
	}
	graph := buildGraph(pairs)
	fmt.Printf("Building virtual orderbook for %s/%s...\n", baseCurrency, quoteCurrency)
	virtualOrderbook := buildVirtualOrderbook(graph, baseCurrency, quoteCurrency)
	// fmt.Println("=== Virtual Orderbook ===")
	// printVirtualOrderbook(virtualOrderbook)
	// fmt.Println("---")

	// Execute on virtual orderbook to find best routes
	fmt.Printf("Executing %.0f %s on virtual orderbook...\n", amount, baseCurrency)
	bestAskRoute, bestBidRoute := executeOnVirtualOrderbook(virtualOrderbook, amount)

	// Print results
	fmt.Printf("Test Case: %s -> %s (Amount: %.0f)\n", baseCurrency, quoteCurrency, amount)
	printBestRouteOutput(bestBidRoute, bestAskRoute)
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
		parts := strings.Fields(line)
		if len(parts) == 3 {
			isCurrencyPair := true
			for i, part := range parts[:2] {
				if i < 2 {
					for _, char := range part {
						if char < 'A' || char > 'Z' {
							isCurrencyPair = false
							break
						}
					}
				}
				if !isCurrencyPair {
					break
				}
			}

			// Check if third part is a valid number
			if isCurrencyPair {
				if _, err := strconv.ParseFloat(parts[2], 64); err != nil {
					isCurrencyPair = false
				}
			}
			if isCurrencyPair {
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

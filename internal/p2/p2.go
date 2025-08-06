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
	Price       float64
	Amount      float64
	Route       []string
	LevelPrices []float64 // Giá của từng level trong mỗi pair của route
}

type VirtualTradingPair struct {
	Base      string
	Quote     string
	AskOrders []VirtualLevel
	BidOrders []VirtualLevel
}

type BestRoute []struct {
	Route       []string
	Price       float64
	Amount      float64
	LevelPrices []float64 // Giá của từng level trong mỗi pair của route
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

func invertOrders(levels []Level) []Level {
	var invertedOrders []Level
	for _, level := range levels {
		if level.Price > 0 {
			invertedOrders = append(invertedOrders, Level{
				Price:  1.0 / level.Price,
				Amount: level.Amount * level.Price,
			})
		}
	}
	return invertedOrders
}

func findAllPaths(graph Graph, start, end string, maxDepth int) [][]string {
	visited := make(map[string]bool)
	startPath := []string{start}
	return findPathsRecursive(graph, start, end, visited, startPath, maxDepth)
}

func findPathsRecursive(graph Graph, currentToken, targetToken string, visited map[string]bool, currentPath []string, maxDepth int) [][]string {
	var allPaths [][]string
	if len(currentPath) > maxDepth {
		return allPaths
	}
	if currentToken == targetToken && len(currentPath) > 1 {
		pathCopy := make([]string, len(currentPath))
		copy(pathCopy, currentPath)
		allPaths = append(allPaths, pathCopy)
		return allPaths
	}

	visited[currentToken] = true
	for nextToken := range graph[currentToken] {
		if !visited[nextToken] {
			newPath := make([]string, len(currentPath))
			copy(newPath, currentPath)
			newPath = append(newPath, nextToken)

			pathsFromNext := findPathsRecursive(graph, nextToken, targetToken, visited, newPath, maxDepth)
			allPaths = append(allPaths, pathsFromNext...)
		}
	}

	visited[currentToken] = false
	return allPaths
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
	sortVirtualLevels(&virtualPair.AskOrders, true)
	sortVirtualLevels(&virtualPair.BidOrders, false)
	virtualPair.AskOrders = mergeVirtualLevels(virtualPair.AskOrders)
	virtualPair.BidOrders = mergeVirtualLevels(virtualPair.BidOrders)
	return virtualPair
}

func calculateOrdersFromPath(graph Graph, path []string, isAsk bool) []VirtualLevel {
	var levels []VirtualLevel
	if len(path) < 2 {
		return levels
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
			levels = append(levels, VirtualLevel{
				Price:       effectivePrice,
				Amount:      combo.depth,
				Route:       truePath,
				LevelPrices: combo.prices, // Lưu giá của từng level
			})
		}
	}
	return levels
}

// RouteCandidate represents a potential trading route with tracking info
type RouteCandidate struct {
	prices       []float64
	levelIndices []int // track which level index in each hop
	finalPrice   float64
	maxVolume    float64
}

func getAllPriceVolumeCombinations(graph Graph, path []string, isAsk bool) []PriceVolumeCombo {
	if len(path) < 2 {
		return []PriceVolumeCombo{}
	}
	// Get all order levels for each hop in the path
	var allHopLevels [][]Level
	for i := 0; i < len(path)-1; i++ {
		fromToken := path[i]
		toToken := path[i+1]
		pair, exists := graph[fromToken][toToken]
		if !exists {
			return []PriceVolumeCombo{}
		}

		var hopLevels []Level

		if isAsk {
			for _, order := range pair.AskOrders {
				hopLevels = append(hopLevels, Level{order.Price, order.Amount})
			}
		} else {
			for _, order := range pair.BidOrders {
				hopLevels = append(hopLevels, Level{order.Price, order.Amount})
			}
		}
		allHopLevels = append(allHopLevels, hopLevels)
	}

	return generateCombinationsWithVolumeTracking(allHopLevels, isAsk)
}

func generateCombinationsWithVolumeTracking(allHopLevels [][]Level, isAsk bool) []PriceVolumeCombo {

	if len(allHopLevels) == 0 {
		return []PriceVolumeCombo{}
	}
	// Generate all possible route candidates first
	var candidates []RouteCandidate
	generateAllRouteCandidates(allHopLevels, 0, []float64{}, []int{}, &candidates)
	sortCandidatesByPrice(candidates, isAsk)

	remainingVolumes := make([][]float64, len(allHopLevels))
	for i, hopLevels := range allHopLevels {
		remainingVolumes[i] = make([]float64, len(hopLevels))
		for j, level := range hopLevels {
			remainingVolumes[i][j] = level.Amount
		}
	}
	// Apply greedy selection with volume tracking
	var result []PriceVolumeCombo
	for _, candidate := range candidates {
		// Calculate maximum volume we can use for this route
		maxUsableVolume := candidate.maxVolume
		for hopIdx, levelIdx := range candidate.levelIndices {
			available := remainingVolumes[hopIdx][levelIdx]
			if available < maxUsableVolume {
				maxUsableVolume = available
			}
		}

		if maxUsableVolume > 0 {
			for hopIdx, levelIdx := range candidate.levelIndices {
				remainingVolumes[hopIdx][levelIdx] -= maxUsableVolume
			}
			result = append(result, PriceVolumeCombo{
				prices: candidate.prices,
				depth:  maxUsableVolume,
			})
		}
	}
	return result
}

func generateAllRouteCandidates(allHopLevels [][]Level, hopIndex int, currentPrices []float64, currentIndices []int, candidates *[]RouteCandidate) {
	if hopIndex >= len(allHopLevels) {
		finalPrice := 1.0
		for _, price := range currentPrices {
			finalPrice *= price
		}
		maxVolume := math.Inf(1)
		for hopIdx, levelIdx := range currentIndices {
			levelVolume := allHopLevels[hopIdx][levelIdx].Amount
			if levelVolume < maxVolume {
				maxVolume = levelVolume
			}
		}
		if maxVolume != math.Inf(1) && maxVolume > 0 {
			pricesCopy := make([]float64, len(currentPrices))
			copy(pricesCopy, currentPrices)
			indicesCopy := make([]int, len(currentIndices))
			copy(indicesCopy, currentIndices)

			*candidates = append(*candidates, RouteCandidate{
				prices:       pricesCopy,
				levelIndices: indicesCopy,
				finalPrice:   finalPrice,
				maxVolume:    maxVolume,
			})
		}
		return
	}
	for levelIdx, level := range allHopLevels[hopIndex] {
		newPrices := make([]float64, len(currentPrices))
		copy(newPrices, currentPrices)
		newPrices = append(newPrices, level.Price)

		newIndices := make([]int, len(currentIndices))
		copy(newIndices, currentIndices)
		newIndices = append(newIndices, levelIdx)

		generateAllRouteCandidates(allHopLevels, hopIndex+1, newPrices, newIndices, candidates)
	}
}

func sortCandidatesByPrice(candidates []RouteCandidate, isAsk bool) {
	// NOTE: Simple bubble sort for clarity (can optimize later)
	n := len(candidates)
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			shouldSwap := false
			if isAsk {
				shouldSwap = candidates[i].finalPrice > candidates[j].finalPrice
			} else {
				shouldSwap = candidates[i].finalPrice < candidates[j].finalPrice
			}
			if shouldSwap {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
}

// NOTE: may be just need merge
func sortVirtualLevels(levels *[]VirtualLevel, isAsk bool) {
	levelList := *levels
	totalLevels := len(levelList)
	if isAsk {
		for i := 0; i < totalLevels-1; i++ {
			for j := i + 1; j < totalLevels; j++ {
				firstLevelPrice := levelList[i].Price
				secondLevelPrice := levelList[j].Price
				if firstLevelPrice > secondLevelPrice {
					levelList[i], levelList[j] = levelList[j], levelList[i]
				}
			}
		}
	} else {
		for i := 0; i < totalLevels-1; i++ {
			for j := i + 1; j < totalLevels; j++ {
				firstLevelPrice := levelList[i].Price
				secondLevelPrice := levelList[j].Price
				if firstLevelPrice < secondLevelPrice {
					levelList[i], levelList[j] = levelList[j], levelList[i]
				}
			}
		}
	}
}

func mergeVirtualLevels(levels []VirtualLevel) []VirtualLevel {
	if len(levels) == 0 {
		return levels
	}
	var merged []VirtualLevel
	current := levels[0]
	for i := 1; i < len(levels); i++ {
		if math.Abs(levels[i].Price-current.Price) < 1e-8 {
			// Same price, merge quantities
			current.Amount += levels[i].Amount
			if levels[i].Price < current.Price {
				current.Route = levels[i].Route
			}
		} else {
			merged = append(merged, current)
			current = levels[i]
		}
	}
	merged = append(merged, current)
	return merged
}

func executeOnVirtualOrderbook(virtualPair VirtualTradingPair, targetAmount float64) (BestRoute, BestRoute) {
	bestAskRoute := findBestRouteFromVirtualOrderbook(virtualPair.AskOrders, targetAmount, true)
	bestBidRoute := findBestRouteFromVirtualOrderbook(virtualPair.BidOrders, targetAmount, false)

	return bestAskRoute, bestBidRoute
}

func findBestRouteFromVirtualOrderbook(levels []VirtualLevel, targetAmount float64, isAsk bool) BestRoute {
	if len(levels) == 0 {
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

	for _, level := range levels {
		if remainingAmount <= 0 {
			break
		}

		executed := math.Min(remainingAmount, level.Amount)
		executedAmount += executed

		if isAsk {
			cost := executed * level.Price
			totalCost += cost
			// fmt.Printf("Level %d: %.8f %s at %.8f ETH (Route: %s) - Cost: %.8f ETH\n",
			// 	i+1, executed, "KNC", level.Price, formatRoute(level.Route), cost)
		} else {
			received := executed * level.Price
			totalCost += received
			// fmt.Printf("Level %d: %.8f %s at %.8f ETH (Route: %s) - Received: %.8f ETH\n",
			// 	i+1, executed, "KNC", level.Price, formatRoute(level.Route), received)
		}

		remainingAmount -= executed
		bestRoute = append(bestRoute, struct {
			Route       []string
			Price       float64
			Amount      float64
			LevelPrices []float64
		}{
			Route:       level.Route,
			Price:       level.Price,
			Amount:      executed,
			LevelPrices: level.LevelPrices,
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

func printVirtualOrderbook(virtualPair VirtualTradingPair) {
	fmt.Printf("%s %s\n", virtualPair.Base, virtualPair.Quote)
	fmt.Printf("%d\n", len(virtualPair.AskOrders))

	for _, order := range virtualPair.AskOrders {
		fmt.Printf("%.8f %.0f (%s) [Level Prices: %v]\n",
			order.Price, order.Amount, formatRoute(order.Route), order.LevelPrices)
	}

	fmt.Printf("%d\n", len(virtualPair.BidOrders))

	for _, order := range virtualPair.BidOrders {
		fmt.Printf("%.8f %.0f (%s) [Level Prices: %v]\n",
			order.Price, order.Amount, formatRoute(order.Route), order.LevelPrices)
	}
}

func printBestRouteOutput(bestBidRoute, bestAskRoute BestRoute) {
	fmt.Println("=== Best Routes Found ===")
	fmt.Println("ASK Routes:")
	if len(bestAskRoute) > 0 {
		for i, route := range bestAskRoute {
			fmt.Printf("  %d. %s (Price: %.8f, Amount: %.8f %s) [Level Prices: %v]\n",
				i+1, formatRoute(route.Route), route.Price, route.Amount, route.Route[len(route.Route)-1], route.LevelPrices)
		}
	} else {
		fmt.Println("  NO_ROUTE")
	}
	fmt.Println("BID Routes:")
	if len(bestBidRoute) > 0 {
		for i, route := range bestBidRoute {
			fmt.Printf("  %d. %s (Price: %.8f, Amount: %.8f %s) [Level Prices: %v]\n",
				i+1, formatRoute(route.Route), route.Price, route.Amount, route.Route[0], route.LevelPrices)
		}
	} else {
		fmt.Println("  NO_ROUTE")
	}

	fmt.Println("---")
}

func formatRoute(route []string) string {
	return strings.Join(route, "->")
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
	var levels []Level
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
		levels = append(levels, Level{Price: price, Amount: amount})
		*lineIdx++
	}
	return levels, nil
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
			fmt.Printf("Error parsing ask levels: %v\n", err)
			return
		}
		bidOrders, err := parseOrderBook(lines, &lineIdx, "bid", pairBase, pairQuote)
		if err != nil {
			fmt.Printf("Error parsing bid levels: %v\n", err)
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

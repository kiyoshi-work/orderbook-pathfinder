# Analysis: Problem 1 & 2

## Overview

This document analyzes two problems related to finding trading routes for cryptocurrency pairs:

- **Problem 1**: Assume infinite depth (no slippage or fees to consider)
- **Problem 2**: Work with specific trading volumes and order book depth levels

## Scope & Assumptions

### Key Questions & Decisions

- **Multi-exchange**: Can one pair have multiple orderbooks from different exchanges? → Assume no or already merged
- **Multi-hop limit**: How to handle cycle arbitrage detection? → Alert and stop
- **Orderbook handling**: Real-time vs single snapshot? → Assume offline processing
- **Implementation scope**: Core algorithm only vs complete service? → Core algorithm focus

## Technical Analysis

### Price Modeling

#### Combined Pairs
```
X(B->Q')(askX - bidX) +++ Y(Q'->Q)(askY - bidY) ==> (B->Q)(askX * askY - bidX * bidY)
```

#### Reverse Pairs
```
forward: X(B->Q)(askX - bidX) ===> reverse: X(Q->B)(1/bidX - 1/askX)
```

## Problem 1: Infinite Depth Approach

### Approach
- **Ask routes**: Minimize multiplication operator
- **Bid routes**: Maximize multiplication operator
- **Algorithm**: Weighted Single-Source Shortest Path (SSSP) Problem
  - Options: Dijkstra, A*, Bellman-Ford, Floyd-Warshall

### Data Structures

```go
type TradingPair struct {
    Base  string
    Quote string
    Ask   float64
    Bid   float64
}

type TradingRoute struct {
    Route []string // e.g., ["ETH", "USDT", "KNC"]
    Price float64
}

// Graph representation
map[string]map[string]TradingPair // ["ETH"]["USDC"]TradingPair
```

### Solution Strategy

#### For Best Ask (Min Cost)
- Find shortest path with weight > 0
- **Challenge**: Multiplication operator requires logarithm transformation
- **Issue**: Edge weight < 1 → logarithm < 0
- **Solution**: Use Bellman-Ford O(symbol×pair) instead of Dijkstra

#### For Best Bid (Max Cost)
- Find shortest path with inverted weight (1/Bid)
- Apply same algorithm as ask

## Problem 2: Finite Depth Approach

### Approach
- Optimize min ask and max bid with specific base token size
- Consider splitting size across routes for better pricing/liquidity
- **Complexity**: Min-Cost Flow problem (too complex for scope)
- **Simplified**: Keep entire size through each hop
- **Result**: Graph traversal with discrete convex function edge weights

### Example Scenario

```text
KNC ETH 300
3
KNC USDT
2
1.1 150
1.2 200
2
0.9 100
0.8 300
ETH USDT
2
360 1000
365 500
2
355 800
350 600
KNC ETH
2
0.0031 400
0.0032 100
2
0.0025 400
0.0024 100
```

**Available Routes:**
- ETH→USDT→KNC: 0.00309859 (150 volume)
- ETH→KNC: 0.0031 (400 volume)

### Data Structures

```go
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

// Graph representation
map[string]map[string]TradingPair // ["ETH"]["USDC"]TradingPair
```

### Solution Strategy

1. **Traverse all paths** and simulate execution + check depth
2. **Build virtual orderbook** for target pair by aggregating all possible routes

## Virtual Orderbook Implementation

### Step 1: Build Virtual Orderbook

1. **Find all paths** from base to quote currency (e.g., KNC→ETH)
2. **Generate all route candidates** with price combinations for each path:
   - Path 1: KNC→USDT→ETH → [1.1×1/360, 1.2×1/360, 1.1×1/365, 1.2×1/365]
   - Path 2: KNC→ETH → [0.0031, 0.0032]
3. **Sort candidates by price** (best price first)
4. **Apply greedy volume tracking**: for each candidate, calculate max usable volume considering remaining volumes, then deduct used volumes to prevent double-counting
5. **Sort and merge** orders with same price

#### Example Virtual Orderbook
```
KNC ETH
3
0.00309859 150 (KNC→USDT→ETH)
0.0031 200 (KNC→ETH)  
0.0032 100 (KNC→USDT→ETH)
2
0.0025 400 (ETH→KNC)
0.0024 100 (ETH→KNC)
```

### Step 2: Execute on Virtual Orderbook

1. **Walk Orderbook**: Start from best price level (lowest for ask, highest for bid)
2. **For each level**: Execute `min(target_amount, level_amount)` and track total cost
3. **Continue**: Move to next level until target amount is fulfilled

## Implementation Limits
- **Reduced Complexity**: Fewer levels = faster computation and less memory usage and prevents exploring extremely long routes that are rarely optimal
- **MAX_LEVELS_PER_PAIR = 5**: Limit number of order levels per trading pair to reduce complexity
- **MAX_PATH_DEPTH = 5**: Limit maximum path length to prevent exponential growth in route combinations
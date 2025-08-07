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
X(B→Q'): askX@volX, bidX@volX
Y(Q'→Q): askY@volY, bidY@volYx
==> X+Y(B→Q): (askX×askY)@min(volX,volY), (bidX×bidY)@min(volX,volY)
```

#### Reverse Pairs
```
X(B→Q): askX@volX, bidX@volX
==> X_rev(Q→B): (1/bidX)@volX, (1/askX)@volX
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
   - For each path, create all possible combinations of price levels across all hops
   - Each candidate tracks: route prices, level indices, final effective price, max volume
3. **Sort candidates by price** (best price first - lowest for ask, highest for bid)
4. **Apply greedy volume tracking**: for each candidate, calculate max usable volume considering remaining volumes, then deduct used volumes to prevent double-counting
5. **Sort and merge** orders with same effective price

#### Example

**Input:**
```
KNC USDT (ASK): 1@200, 1.4@400
ETH USDT (BID): 30@10, 20@15
=> USDT ETH (ASK): 1/30@300, 1/20@300
```

**❌ Old Logic (Incorrect - Independent Volume Calculation):**
```
Each route candidate calculates volume independently:

1. KNC(1@200) → ETH(1/30@300) = 1/30@min(200,300) = 1/30@200
2. KNC(1.4@400) → ETH(1/30@300) = 1.4/30@min(400,300) = 1.4/30@300
3. KNC(1@200) → ETH(1/20@300) = 1/20@min(200,300) = 1/20@200
4. KNC(1.4@400) → ETH(1/20@300) = 1.4/20@min(400,300) = 1.4/20@300
==> Wrong Result:
Problem: Same liquidity counted multiple times!
```
**✅ Correct Logic (Greedy Volume Tracking):**
```
Route Candidates (sorted by price):
1. 1×(1/30) = 1/30@min(200,300) = 1/30@200
2. 1.4×(1/30) = 1.4/30@min(400,300-200) = 1.4/30@100
3. 1×(1/20) = 1/20@min(200-200,300) = 1/20@0  
4. 1.4×(1/20) = 1.4/20@min(400-100,300) = 1.4/20@300
```

### Step 2: Execute on Virtual Orderbook

1. **Walk Orderbook**: Start from best price level (lowest for ask, highest for bid)
2. **For each level**: Execute `min(target_amount, level_amount)` and track total cost
3. **Continue**: Move to next level until target amount is fulfilled

#### Example

**Final Virtual Orderbook for KNC→ETH (ASK):**
```
1. 1/30@200
2. 1/20@300  
3. 1.4/30@100
```

**Execution Logic:**
```
Target: 300 KNC

Step 1: 1/30@200
- Execute: min(250, 200) = 200 KNC
- Remaining: 100 KNC

Step 2: 1/20@300
- Execute: min(100, 300) = 100 KNC  
- Remaining: 0 KNC
```

## Implementation Limits
- **Reduced Complexity**: Fewer levels = faster computation and less memory usage and prevents exploring extremely long routes that are rarely optimal
- **MAX_LEVELS_PER_PAIR = 5**: Limit number of order levels per trading pair to reduce complexity
- **MAX_PATH_DEPTH = 5**: Limit maximum path length to prevent exponential growth in route combinations

## Future Work

### Real-time Orderbook Updates
- **Challenge**: Current implementation rebuilds entire virtual orderbook when any trading pair updates

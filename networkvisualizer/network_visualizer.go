//Networkvisualizer is a testing tool that generates a graph of the network
//AssembleGraph is the main runner of the program and takes three parameters
//Start/End(int) specifies the range of blocks you would like to include in the graph, leaving both values 0 will default to the 100 most recent blocks
//The parameters for the program can be modified on the line that calls AssembleGraph within main
//The program can be run with the command: 							go run network_visualizer.go
//The generated DOT file can be viewed with a VSCode extension:		tintinweb.graphviz-interactive-preview
//Aternatively the DOT file can be converted into other image formats using a dot command
//A full node must be running for the tool to work properly
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"sync"

	crand "crypto/rand"

	"github.com/spruce-solutions/go-quai/common"
	"github.com/spruce-solutions/go-quai/core/types"
	"github.com/spruce-solutions/go-quai/ethclient"
	"github.com/spruce-solutions/go-quai/metrics"
	"github.com/spruce-solutions/go-quai/params"
	"github.com/spruce-solutions/go-quai/rlp"
	"gopkg.in/urfave/cli.v1"
	blake3hash "lukechampine.com/blake3"
)

var (
	prime, _        = ethclient.Dial("ws://127.0.0.1:8547")
	region1, _      = ethclient.Dial("ws://127.0.0.1:8579")
	region2, _      = ethclient.Dial("ws://127.0.0.1:8581")
	region3, _      = ethclient.Dial("ws://127.0.0.1:8583")
	zone11, _       = ethclient.Dial("ws://127.0.0.1:8611")
	zone12, _       = ethclient.Dial("ws://127.0.0.1:8643")
	zone13, _       = ethclient.Dial("ws://127.0.0.1:8675")
	zone21, _       = ethclient.Dial("ws://127.0.0.1:8613")
	zone22, _       = ethclient.Dial("ws://127.0.0.1:8645")
	zone23, _       = ethclient.Dial("ws://127.0.0.1:8677")
	zone31, _       = ethclient.Dial("ws://127.0.0.1:8615")
	zone32, _       = ethclient.Dial("ws://127.0.0.1:8647")
	zone33, _       = ethclient.Dial("ws://127.0.0.1:8679")
	primeSubGraph   = "subgraph cluster_Prime { label = \"Prime\" node [color = red]"
	region1SubGraph = "subgraph cluster_Region1 { label = \"Region1\" node [color = green]"
	region2SubGraph = "subgraph cluster_Region2 { label = \"Region2\" node [color = dodgerblue]"
	region3SubGraph = "subgraph cluster_Region3 { label = \"Region3\" node [color = orange]"
	zone11SubGraph  = "subgraph cluster_Zone11 { label = \"Zone11\" node [color = lawngreen]"
	zone12SubGraph  = "subgraph cluster_Zone12 { label = \"Zone12\" node [color = limegreen]"
	zone13SubGraph  = "subgraph cluster_Zone13 { label = \"Zone13\" node [color = mediumspringgreen]"
	zone21SubGraph  = "subgraph cluster_Zone21 { label = \"Zone21\" node [color = aqua]"
	zone22SubGraph  = "subgraph cluster_Zone22 { label = \"Zone22\" node [color = blue]"
	zone23SubGraph  = "subgraph cluster_Zone23 { label = \"Zone23\" node [color = \"#8a4cee\"]"
	zone31SubGraph  = "subgraph cluster_Zone31 { label = \"Zone31\" node [color = darkorange1]"
	zone32SubGraph  = "subgraph cluster_Zone32 { label = \"Zone32\" node [color = orangered2]"
	zone33SubGraph  = "subgraph cluster_Zone33 { label = \"Zone33\" node [color = \"#c55200\"]"
	uncleSubGraph   = []string{"subgraph cluster_Uncles { label = \"Uncles\""}
	//Initializing all the chains to be used in the graph for each Region/Zone/Prime
	zone11Chain  = Chain{zone11, zone11SubGraph, []node{}, 2, []Chain{}, 0, 0}
	zone12Chain  = Chain{zone12, zone12SubGraph, []node{}, 2, []Chain{}, 0, 0}
	zone13Chain  = Chain{zone13, zone13SubGraph, []node{}, 2, []Chain{}, 0, 0}
	region1Chain = Chain{region1, region1SubGraph, []node{}, 1, []Chain{zone11Chain, zone12Chain, zone13Chain}, 0, 0}
	zone21Chain  = Chain{zone21, zone21SubGraph, []node{}, 2, []Chain{}, 0, 0}
	zone22Chain  = Chain{zone22, zone22SubGraph, []node{}, 2, []Chain{}, 0, 0}
	zone23Chain  = Chain{zone23, zone23SubGraph, []node{}, 2, []Chain{}, 0, 0}
	region2Chain = Chain{region2, region2SubGraph, []node{}, 1, []Chain{zone21Chain, zone22Chain, zone23Chain}, 0, 0}
	zone31Chain  = Chain{zone31, zone31SubGraph, []node{}, 2, []Chain{}, 0, 0}
	zone32Chain  = Chain{zone32, zone32SubGraph, []node{}, 2, []Chain{}, 0, 0}
	zone33Chain  = Chain{zone33, zone33SubGraph, []node{}, 2, []Chain{}, 0, 0}
	region3Chain = Chain{region3, region3SubGraph, []node{}, 1, []Chain{zone31Chain, zone32Chain, zone33Chain}, 0, 0}
	primeChain   = Chain{prime, primeSubGraph, []node{}, 0, []Chain{region1Chain, region2Chain, region3Chain}, 0, 0}
	chains       = []Chain{primeChain, region1Chain, region2Chain, region3Chain, zone11Chain, zone12Chain, zone13Chain, zone21Chain, zone22Chain, zone23Chain, zone31Chain, zone32Chain, zone33Chain}
	edges        = []string{}
	ctx          = context.Background()
	hashLength   = 10
	f            *os.File

	//Destination for Flag arguments
	StartFlag         int
	RangeFlag         int
	CompressedFlag    = true
	LiveFlag          = false
	UnclesFlag        = false
	SaveFileFlag      = "TestGraph.dot"
	tempinclusionFlag = []string{"prime", "c", "p", "h", "c1", "c2", "c3", "p1", "p2", "p3", "h1", "h2", "h3"}
	InclusionFlag     = cli.StringSlice(tempinclusionFlag)
)

type Chain struct {
	client    *ethclient.Client //Used for retrieving the Block information from the DB
	subGraph  string            //Used to store initial subgraph formatting for respective chain
	nodes     []node            //Contains the nodes of each chain
	order     int               //Stores the order of the chain being dealt with
	subChains []Chain           //Contains each subordinate chain
	startLoc  int
	endLoc    int
}

type node struct {
	nodehash   common.Hash
	nodestring string
	number     int
}

type Config struct {
	// Number of threads to use when mining.
	// -1 => mining disabled
	// 0 => max no. of threads, limited by max CPUs
	// >0 => exact no. of threads, up to max CPUs
	MiningThreads int

	// When set, notifications sent by the remote sealer will
	// be block header JSON objects instead of work package arrays.
	NotifyFull bool

	// Logger object
	Log log.Logger `toml:"-"`

	// Fake proof of work for testing
	Fakepow bool
}

// Blake3 a consensus engine based on the Blake3 hash function
type Blake3 struct {
	config Config

	// Runtime state
	lock      sync.Mutex // Ensures thread safety for the in-memory caches and mining fields
	closeOnce sync.Once  // Ensures exit channel will not be closed twice.

	// Mining related fields
	rand     *rand.Rand    // Properly seeded random source for nonces
	update   chan struct{} // Notification channel to update mining parameters
	hashrate metrics.Meter // Meter tracking the average hashrate
}

func main() {
	app := cli.NewApp()
	app.Name = "visualizenetwork"
	app.Usage = "Generates graphs of Quai Network"
	app.Action = func(c *cli.Context) error {
		return nil
	}
	//Slice of flags used by CLI, connnect the Destination value to respective flag variable
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:        "start",
			Value:       0,
			Usage:       "Determines the start block for the graph in terms of block number",
			Destination: &StartFlag,
		},
		cli.IntFlag{
			Name:        "range",
			Value:       100,
			Usage:       "Sets how many blocks to include in the graph(default = 100)",
			Destination: &RangeFlag,
		},
		cli.BoolTFlag{
			Name:        "compressed, c",
			Usage:       "Hides blocks inbetween coincident blocks, aside from those within the specified range(default = true)",
			Destination: &CompressedFlag,
		},
		cli.BoolFlag{
			Name:        "live",
			Usage:       "Allows for the graph to update real-time(default = false)",
			Destination: &LiveFlag,
		},
		cli.BoolFlag{
			Name:        "uncles",
			Usage:       "Includes uncle blocks in the live version of the graph, only works if live is true(default = false)",
			Destination: &UnclesFlag,
		},
		cli.StringFlag{
			Name:        "savefile",
			Value:       "TestGraph.dot",
			Usage:       "Allows for specification of output file for the graph(default = \"TestGraph.dot\")",
			Destination: &SaveFileFlag,
		},
		cli.StringSliceFlag{
			Name:  "include, i",
			Usage: "String slice containing the chains you would like to include in the graph. Default:{prime,c,p,h,c1,c2,c3,p1,p2,p3,h1,h2,h3}(All chains)",
			Value: &InclusionFlag,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	//Opening IO file to write to, WiP for flag options to specify file
	f, err = os.Create(SaveFileFlag)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	Rflag := RangeFlag
	Sflag := StartFlag
	if len(InclusionFlag) != 13 {
		InclusionFlag = InclusionFlag.Value()[13:]
	}
	chains = handleInclusion(chains)
	if Sflag == 0 {
		for i := range chains {
			blockNum, _ := chains[i].client.BlockNumber(context.Background())
			chains[i].startLoc = int(blockNum) - Rflag
			if chains[i].startLoc < 1 {
				chains[i].startLoc = 1
			}
			if chains[i].startLoc > int(blockNum) {
				chains[i].startLoc = int(blockNum) + 1
			}
			chains[i].endLoc = int(blockNum)
		}
	} else {
		for i := range chains {
			blockNum, _ := chains[i].client.BlockNumber(context.Background())
			chains[i].endLoc = Sflag + Rflag
			if chains[i].endLoc > int(blockNum) {
				chains[i].endLoc = int(blockNum)
			}
			chains[i].startLoc = Sflag
		}
	}
	AssembleGraph(chains)
}

func AssembleGraph(chains []Chain) {
	for i := 0; i < len(chains); i++ {
		for j := chains[i].startLoc; j <= chains[i].endLoc; j++ {
			header, err := chains[i].client.HeaderByNumber(ctx, big.NewInt(int64(j)))
			if err != nil {
				panic(err)
			}
			hHash := header.Hash()
			chains[i].addNode(hHash, j)
			if j != chains[i].endLoc {
				nextheader, err := chains[i].client.HeaderByNumber(ctx, big.NewInt(int64(j+1)))
				if err != nil {
					panic(err)
				}
				nhHash := nextheader.Hash()
				addEdge(true, hHash, nhHash, chains[i].order, "")
			}
			if chains[i].order < 2 {
				addCoincident(hHash, chains)
			}
		}
	}
	OrderChains(chains)
	bottomUp(chains)
	writeToDOT(chains)
}

func addCoincident(hash common.Hash, c []Chain) {
	for i := 0; i < len(c); i++ {
		header, err := c[i].client.HeaderByHash(ctx, hash)
		tempHash := header.Hash()
		if err == nil {
			c[i].addNode(tempHash, int(header.Number[c[i].order].Int64()))
			if int(header.Number[c[i].order].Int64()) < c[i].startLoc && !CompressedFlag {
				c[i].startLoc = int(header.Number[c[i].order].Int64())
			}
			if int(header.Number[c[i].order].Int64()) > c[i].endLoc && !CompressedFlag {
				c[i].endLoc = int(header.Number[c[i].order].Int64())
			}
			if c[i].order < 2 {
				addEdge(false, hash, hash, c[i].order, "")
			}
			if c[i].order < 1 {
				addCoincident(hash, c[i].subChains)
			}
		}
	}
}

func (c *Chain) addNode(hash common.Hash, num int) {
	if !hasNode(c, hash) {
		c.nodes = append(c.nodes, node{hash, "\n\"" + fmt.Sprint(c.order) + hash.String()[2:hashLength+2] + "\" [label = \"" + hash.String()[2:hashLength+2] + "\\n " + fmt.Sprint(num) + "\"]", num})
	}
}

func addEdge(dir bool, node1 common.Hash, node2 common.Hash, order int, color string) {
	node1Hash := fmt.Sprint(order) + node1.String()[2:hashLength+2]
	node2Hash := fmt.Sprint(order) + node2.String()[2:hashLength+2]
	if !dir {
		node1Hash = fmt.Sprint(order) + node1.String()[2:hashLength+2]
		node2Hash = fmt.Sprint(order+1) + node1.String()[2:hashLength+2]
	}
	if !hasEdge(node1Hash, node2Hash) {
		if color != "" {
			edges = append(edges, "\""+node1Hash+"\" -> \""+node2Hash+"\" [color = "+color+"]")
		} else if dir {
			edges = append(edges, "\""+node1Hash+"\" -> \""+node2Hash+"\"")
		} else {
			edges = append(edges, "\""+node1Hash+"\" -> \""+node2Hash+"\""+" [dir = none]")
		}
	}
}

func bottomUp(chains []Chain) {
	config := Config{Fakepow: false}
	blake3, _ := New(config, nil, false)
	for i := 0; i < len(chains); i++ {
		for j := chains[i].endLoc; j >= chains[i].startLoc; j-- {
			header, _ := chains[i].client.HeaderByNumber(ctx, big.NewInt(int64(j)))
			diffOrder, _ := blake3.GetDifficultyOrder(header)
			hash := header.Hash()
			chains[i].addNode(hash, j)
			for k := chains[i].order; k > diffOrder; k-- {
				if len(search4Hash(chains, hash)) != 1 {
					addEdge(false, hash, hash, k-1, "")
				}
			}
		}
	}
}

func search4Hash(chains []Chain, hash common.Hash) []int {
	found := []int{}
	for i := 0; i < len(chains); i++ {
		_, err := chains[i].client.HeaderByHash(ctx, hash)
		if err == nil {
			found = append(found, i)
		}
	}
	return found
}

func hasEdge(node1Hash string, node2Hash string) bool {
	for _, edge := range edges {
		if strings.Contains(edge, "\""+node1Hash+"\" -> \""+node2Hash+"\"") {
			return true
		}
	}
	return false
}

//Returns true if the chain has the node. Otherwise returns false
func hasNode(c *Chain, hash common.Hash) bool {
	for _, node := range c.nodes {
		modHash := hash.String()[2 : hashLength+2]
		if strings.Contains(node.nodestring, modHash) {
			return true
		}
	}
	return false
}

func handleInclusion(chains []Chain) []Chain {
	chainsCopy := []Chain{}
	for _, val := range InclusionFlag {
		switch val {
		case "prime":
			chainsCopy = append(chainsCopy, primeChain)
		case "c":
			chainsCopy = append(chainsCopy, region1Chain)
		case "p":
			chainsCopy = append(chainsCopy, region2Chain)
		case "h":
			chainsCopy = append(chainsCopy, region3Chain)
		case "c1":
			chainsCopy = append(chainsCopy, zone11Chain)
		case "c2":
			chainsCopy = append(chainsCopy, zone12Chain)
		case "c3":
			chainsCopy = append(chainsCopy, zone13Chain)
		case "p1":
			chainsCopy = append(chainsCopy, zone21Chain)
		case "p2":
			chainsCopy = append(chainsCopy, zone22Chain)
		case "p3":
			chainsCopy = append(chainsCopy, zone23Chain)
		case "h1":
			chainsCopy = append(chainsCopy, zone31Chain)
		case "h2":
			chainsCopy = append(chainsCopy, zone32Chain)
		case "h3":
			chainsCopy = append(chainsCopy, zone33Chain)
		}

	}
	return chainsCopy
}

func AddUncle(hash common.Hash, order int) {
	uncleSubGraph = append(uncleSubGraph, "\n\""+fmt.Sprint(order)+hash.String()[2:hashLength+2]+"\" [label = \""+hash.String()[2:hashLength+2]+"\"]")
}

// This function determines the difficulty order of a block
func (blake3 *Blake3) GetDifficultyOrder(header *types.Header) (int, error) {
	var difficulties []*big.Int

	if header == nil {
		return types.ContextDepth, errors.New("no header provided")
	}
	if !blake3.config.Fakepow {
		difficulties = header.Difficulty
	} else {
		difficulties = []*big.Int{new(big.Int).Mul(params.MinimumDifficulty[params.PRIME], big.NewInt(4)), new(big.Int).Mul(params.MinimumDifficulty[params.REGION], big.NewInt(2)), params.MinimumDifficulty[params.ZONE]}
	}
	blockhash := blake3.SealHash(header)
	for i, difficulty := range difficulties {
		if difficulty != nil && big.NewInt(0).Cmp(difficulty) < 0 {
			target := new(big.Int).Div(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0)), difficulty)
			if new(big.Int).SetBytes(blockhash.Bytes()).Cmp(target) <= 0 {
				return i, nil
			}
		}
	}
	return -1, errors.New("block does not satisfy minimum difficulty")
}

// SealHash returns the hash of a block prior to it being sealed.
// Used for proper implementation of a GetDifficultyOrder
func (blake3 *Blake3) SealHash(header *types.Header) (hash common.Hash) {
	hasher := blake3hash.New(32, nil)
	hasher.Reset()

	enc := []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra,
		header.Location,
	}
	if header.BaseFee != nil {
		enc = append(enc, header.BaseFee)
	}
	enc = append(enc, header.Nonce)
	rlp.Encode(hasher, enc)
	hasher.Sum(hash[:0])
	return hash
}

func OrderChains(chains []Chain) []Chain {
	//Insertion sorting the chains in order for next steps to be executed properly
	for i := 0; i < len(chains); i++ {
		for j := 1; j < len(chains[i].nodes); j++ {
			for k := j; k >= 1 && chains[i].nodes[k].number < chains[i].nodes[k-1].number; k-- {
				chains[i].nodes[k], chains[i].nodes[k-1] = chains[i].nodes[k-1], chains[i].nodes[k]
			}
		}
	}
	for i := 0; i < len(chains); i++ {
		for j := 0; j < len(chains[i].nodes)-1; j++ {
			if i != 0 {
				addEdge(true, chains[i].nodes[j].nodehash, chains[i].nodes[j+1].nodehash, chains[i].order, "blue")
			}
		}
	}
	return chains
}

// Creates a new Blake3 engine
func New(config Config, notify []string, noverify bool) (*Blake3, error) {
	// Do not allow Fakepow for a real consensus engine
	config.Fakepow = false
	blake3 := &Blake3{
		config:   config,
		hashrate: metrics.NewMeterForced(),
	}
	rng_seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	blake3.rand = rand.New(rand.NewSource(rng_seed.Int64()))
	if nil != err {
		return nil, err
	}
	return blake3, nil
}

//Function for writing a DOT file that generates the graph
func writeToDOT(chains []Chain) {
	f.WriteString("digraph G {\nfontname=\"Helvetica,Arial,sans-serif\"\nnode [fontname=\"Helvetica,Arial,sans-serif\", shape = rectangle, style = filled] \nedge [fontname=\"Helvetica,Arial,sans-serif\"]")
	for _, n := range chains {
		f.WriteString(n.subGraph)
		for _, s := range n.nodes {
			f.WriteString(s.nodestring)
		}
		f.WriteString("}\n")

	}
	for _, n := range uncleSubGraph {
		f.WriteString(n)
	}
	f.WriteString("}\n")
	for _, s := range edges {
		f.WriteString(s + "\n")
	}
	f.WriteString("\n}")
}

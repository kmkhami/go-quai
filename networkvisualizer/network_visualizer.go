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
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/spruce-solutions/go-quai/common"
	"github.com/spruce-solutions/go-quai/ethclient"
	"gopkg.in/urfave/cli.v1"
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
	f            *os.File

	//Destination for Flag arguments
	StartFlag      int
	RangeFlag      int
	CompressedFlag = true
	LiveFlag       = false
	UnclesFlag     = false
	SaveFileFlag   = "TestGraph.dot"
	//Inclusion flag to be implemented
	//inclusionFlag =

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
	nodehash string
	number   int
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
			Name:        "compressed",
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
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	flag.Parse()
	//Opening IO file to write to, WiP for flag options to specify file
	f, err = os.Create(SaveFileFlag)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	Rflag := RangeFlag
	Sflag := StartFlag
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
		for j := chains[i].startLoc; j < chains[i].endLoc; j++ {
			header, err := chains[i].client.HeaderByNumber(ctx, big.NewInt(int64(j)))
			if err != nil {
				panic("Couldn't find block within specified range")
			}
			hHash := header.Hash()
			chains[i].addNode(hHash, j)
			if j != chains[i].endLoc {

			}
		}
	}
	writeToDOT(chains)
}

func (c *Chain) addNode(hash common.Hash, num int) {
	if !hasNode(c, hash) {
		c.nodes = append(c.nodes, node{"\n\"" + fmt.Sprint(c.order) + hash.String()[2:10] + "\" [label = \"" + hash.String()[2:10] + "\\n " + fmt.Sprint(num) + "\"]", num})
	}
}

func (c *Chain) addEdge(dir bool)

//Returns true if the chain has the node. Otherwise returns false
func hasNode(c *Chain, hash common.Hash) bool {
	for _, node := range c.nodes {
		modHash := hash.String()[2:10]
		if strings.Contains(node.nodehash, modHash) {
			return true
		}
	}
	return false
}

/*
//Adds a Node to the chain if it doesn't already exist.
func (c *Chain) AddNode(hash common.Hash, num int) {
	if !ContainsNode("\n\""+fmt.Sprint(c.order)+hash.String()[2:7]+"\" [label = \""+hash.String()[2:7]+"\"]", c.nodes) {
		tempNode := node{}
		if num == 0 {
			blockHeader, _ := c.client.HeaderByHash(context.Background(), hash)
			tempNode = node{"\n\"" + fmt.Sprint(c.order) + hash.String()[2:7] + "\" [label = \"" + hash.String()[2:7] + "\\n " + blockHeader.Number[c.order].String() + "\"]", blockHeader.Number[c.order]}
			c.nodes = append(c.nodes, tempNode)
		} else {
			tempNode = node{"\n\"" + fmt.Sprint(c.order) + hash.String()[2:7] + "\" [label = \"" + hash.String()[2:7] + "\\n " + fmt.Sprint(num) + "\"]", big.NewInt(int64(num))}
			c.nodes = append(c.nodes, tempNode)
		}
	}
}*/

func AddUncle(hash common.Hash, order int) {
	uncleSubGraph = append(uncleSubGraph, "\n\""+fmt.Sprint(order)+hash.String()[2:7]+"\" [label = \""+hash.String()[2:7]+"\"]")
}

//Adds an edge to the chain FROM string1 TO string2. The bool parameter will take away the direction of the edge if it is false.
func (c *Chain) AddEdge(dir bool, node1 string, node2 string) {
	if dir {
		if !Contains("\n\""+node1+"\" -> \""+node2+"\"", edges) {
			if color != "" {
				edges = append(edges, "\n\""+node1+"\" -> \""+node2+"\" [color = \""+color+"\"]")
			} else {
				edges = append(edges, "\n\""+node1+"\" -> \""+node2+"\"")
			}
		}
	} else {
		if !Contains("\n\""+node1+"\" -> \""+node2+"\" [dir = none]", edges) {
			edges = append(edges, "\n\""+node1+"\" -> \""+node2+"\" [dir = none]")
		}
	}
}

//Checks to see if a node already exists
func ContainsNode(s string, list []node) bool {
	for _, a := range list {
		modHash := a.nodehash[:25] + "\"]"
		if modHash == s {
			return true
		}
	}
	return false
}

//Checks to see if the list of strings contains the string passed as the first parameter. Used to check if a Node already exists in the the list.
func Contains(s string, list []string) bool {
	for _, a := range list {
		if a == s {
			return true
		}
	}
	return false
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
				chains[i].AddEdge(true, chains[i].nodes[j].nodehash[2:11], chains[i].nodes[j+1].nodehash[2:11], "blue")
			}
		}
	}
	return chains
}

//Function for writing a DOT file that generates the graph
func writeToDOT(chains []Chain) {
	f.WriteString("digraph G {\nfontname=\"Helvetica,Arial,sans-serif\"\nnode [fontname=\"Helvetica,Arial,sans-serif\", shape = rectangle, style = filled] \nedge [fontname=\"Helvetica,Arial,sans-serif\"]")
	for _, n := range chains {
		f.WriteString(n.subGraph)
		for _, s := range n.nodes {
			f.WriteString(s.nodehash)
		}
		f.WriteString("}\n")

	}
	for _, n := range uncleSubGraph {
		f.WriteString(n)
	}
	f.WriteString("}\n")
	for _, s := range edges {
		f.WriteString(s)
	}
	f.WriteString("\n}")
}

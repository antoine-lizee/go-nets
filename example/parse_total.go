package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/antoine-lizee/go-nets"
)

////////////
//SECTION 1
//Command-line parsing

type FileNames []string //Flag type

//Define the interface functions
func (fn *FileNames) String() string {
	return fmt.Sprint(*fn)
}

func (fn *FileNames) Set(value string) error {
	for _, dt := range strings.Split(value, ",") {
		*fn = append(*fn, dt)
	}
	return nil
}

var (
	loadArg      = flag.String("loadFile", "", "Provide the name of the file to load the network from")
	parsePathArg = flag.String("parsePath", "/media/FD/MISSIONS/ALEX/UM20140215_X/", "Provide the name of the parsed file")
	parseArgs    = FileNames{}
	nameArg      = flag.String("name", "Total1", "Provide the name of the network")
)

func init() {
	flag.Var(&parseArgs, "parse", "Specify a comma separated list of file names")
}

////////////
//SECTION 2
//Utilities
func merge(cs [](chan go_nets.Filing), out chan<- go_nets.Filing) {
	done := make(chan int)
	for ic, c := range cs {
		c := c
		go func() {
			for p := range c {
				_ = ic
				// fmt.Println("Merging in", p, "from channel", ic)
				out <- p
			}
			// fmt.Println("Sending done for channel", ic)
			done <- 1
		}()
	}
	for ic := range cs {
		_ = ic
		// fmt.Println("listening to channel done for index", ic)
		_ = <-done
	}
	close(out)
}

func openFile(name string) os.File {
	fi, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer func() {
		defer func() {
			fi.Close()
		}()
	}()
	return fi
}

////////////
//SECTION 3
//Subsections
func Save(network *go_nets.Network) {
	fmt.Println("Saving network")
	t0 := time.Now()
	network.Save()
	fmt.Printf("\n Successfully saved the network in %v \n", time.Now().Sub(t0))
	fmt.Println("### ---------------")
}

func Parse(fileNames []string, network *go_nets.Network) {
	//Prepare the parsers and channels
	parsers := []go_nets.XmlParser{}
	out := make(chan go_nets.Filing)
	cs := []chan go_nets.Filing{}
	for _, fileName := range fileNames {
		parsers = append(parsers, go_nets.XmlParser{
			FileDir:  *parsePathArg,
			FileName: fileName, // "UM20140215_" + strconv.Itoa(i),
		})
		_ = fileName
		cs = append(cs, make(chan go_nets.Filing))
	}
	//Launch the parsers
	for i, parser := range parsers {
		fi, errOs := os.Create(network.Folder + parser.FileName + ".log")
		if errOs != nil {
			panic(errOs) //TODO change it to t.Error
		}
		defer func() {
			if errOs = fi.Close(); errOs != nil {
				panic(errOs)
			}
		}()
		go parser.Parse(cs[i], fi)
	}

	//Launch fan in
	go merge(cs, out)
	// out = cs[0]

	//Consume the filings as Dispatchers
	i := 0
	for p := range out {
		i++
		fmt.Printf("\r Filing number %d (id = %d)loaded.", i, p.OriginalFileNumber)
		// fmt.Println("Trying to add it to the network")
		network.AddDispatcher(&p)
	}
	fmt.Println("")
	network.Summary(os.Stdout)
}

func Load(n *go_nets.Network, fp string) {
	if fp == "" {
		n.Load()
	} else {
		n.LoadFrom(fp)
	}
}

func Crunch(n *go_nets.Network) *go_nets.Net {
	net := NewNet()
	t0 = time.Now()
	net.CrunchNetwork(&network)
	d1 := time.Now().Sub(t0)
	fmt.Println("Crunched the network in", d1, "- Summary:")
	net.Summary(nil)
	return net
}

////////////
//SECTION 4
//main
func main() {

	// Parse the command line arguments
	flag.Parse()
	doParse := false
	if (len(parseArgs) > 0) && (*loadArg == "") {
		doParse = true
	}

	//Create Network
	network := go_nets.NewNetwork("Total1", nil, "Networks/")

	//Main log setup
	fiLog := openFile(network.Folder + "parse_total.log")
	log.SetOutput(io.Multiwriter(os.Stdout, fiLog))

	//Feed the network
	if doParse { //Parse?
		Parse(parseArgs, &network)
	} else { //Load?
		Load(&network, *loadArg)
	}

	//Save the network
	Save(&network)

	//Analyse
	net := Crunch(&network)
}

///////////////
//SECTION 5
//main debug
func mainDebug() {
	fmt.Println("### TESTING the saving option")
	Parser := go_nets.XmlParser{
		FileDir:  "./",
		FileName: "UM20140215_5.xml", //UM20140215_5 UMtest2
	}

	cs := make(chan go_nets.Filing)
	go Parser.Parse(cs, ioutil.Discard)
	network := go_nets.NewNetwork("Test", ioutil.Discard, "Networks/")

	i := 0
	for p := range cs {
		// p := <-cs
		i++
		fmt.Printf("\r Filing number %d (id = %d)loaded.", i, p.OriginalFileNumber)
		// fmt.Println("Trying to add it to the network")
		network.AddDispatcher(&p)
	}
	fmt.Println("")
	network.Summary(os.Stdout)

	fmt.Println("Saving network")
	t0 := time.Now()
	network.Save()
	fmt.Printf("\n Successfully saved the network in %v \n", time.Now().Sub(t0))
	fmt.Println("### ---------------")

}

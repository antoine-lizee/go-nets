package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"

	"code.google.com/p/go.text/encoding/charmap"

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
	parsePathArg = flag.String("parsePath", "./", "Provide the path of the parsed file") //"/media/FD/MISSIONS/ALEX/UM20140215_X/"
	savePathArg  = flag.String("savePath", "./", "Provide the path of the output files")
	parseArgs    = FileNames{}
	nameArg      = flag.String("name", "Total", "Provide the name of the database")
	batchSizeArg = flag.Int("batchSize", 50000, "Provide the size of the saving batches.")
	nCores       = flag.Int("nCores", 4, "Provide the number of cores for multi-threading.")
)

const usageMsg string = "save_total -parsePath=[] -parse=[,] -name=[] -savePathe=[]\n"

func init() {
	flag.Var(&parseArgs, "parse", "Specify a comma separated list of file names for parsing")
	flag.Usage = usage
}

func usage() {
	fmt.Printf(usageMsg)
	flag.PrintDefaults()
	os.Exit(2)
}

////////////
//SECTION 2
//Utilities
func merge(cs [](chan go_nets.Filing), out chan<- go_nets.Filing) {
	done := make(chan int)
	for ic, c := range cs {
		ic := ic
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

func openFile(name string) *os.File {
	fi, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	return fi
}

func closeFile(fi *os.File) {
	if err := fi.Close(); err != nil {
		panic(err)
	}
}

////////////
//SECTION 3
//Subsections
func Parse(fileNames []string) chan go_nets.Filing {
	//Prepare the parsers and channels
	parsers := []go_nets.XmlParser{}
	out := make(chan go_nets.Filing, *batchSizeArg)
	cs := []chan go_nets.Filing{}
	for _, fileName := range fileNames {
		fileName := fileName
		parsers = append(parsers, go_nets.XmlParser{
			FileDir:  *parsePathArg,
			FileName: fileName, // "UM20140215_" + strconv.Itoa(i),
			Encoding: charmap.Windows1252,
		})
		// _ = fileName
		cs = append(cs, make(chan go_nets.Filing))
	}
	//Launch the parsers
	for i, parser := range parsers {
		parser := parser
		fmt.Println("Starting parsing for file " + parser.FileName)
		fi := openFile(*savePathArg + parser.FileName + ".log")
		csi := cs[i]
		go func() {
			parser.Parse(csi, fi)
			closeFile(fi)
		}()
	}

	//Launch fan in
	go merge(cs, out)
	// out = cs[0]

	return out
}

////////////
//SECTION 4
//main
func main() {

	// Parse the command line arguments
	flag.Parse()
	go_nets.BatchSize = *batchSizeArg
	runtime.GOMAXPROCS(*nCores)

	//Main log setup
	// fiLog := openFile(*savePathArg + "save_total.log")
	fiLog := openFile(*savePathArg + *nameArg + ".log")
	defer closeFile(fiLog)

	log.SetOutput(io.MultiWriter(os.Stdout, fiLog))

	//Prepare the sql saver
	saver := &go_nets.SqlSaver{
		DbPath:   *savePathArg,
		DbName:   *nameArg,
		DBDriver: "sqlite3",
	}

	//Launch It
	go_nets.ListenAndSaveFilings(Parse(parseArgs), saver)

}

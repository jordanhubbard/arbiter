package main

import (
"context"
"flag"
"fmt"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/jordanhubbard/arbiter/internal/api"
"github.com/jordanhubbard/arbiter/internal/arbiter"
"github.com/jordanhubbard/arbiter/pkg/config"
)

const version = "0.1.0"

func main() {
log.SetFlags(log.LstdFlags | log.Lshortfile)

// Parse command line flags
configPath := flag.String("config", "config.yaml", "Path to configuration file")
showVersion := flag.Bool("version", false, "Show version information")
showHelp := flag.Bool("help", false, "Show help message")
flag.Parse()

if *showVersion {
fmt.Printf("Arbiter v%s\n", version)
return
}

func printHelp() {
	fmt.Println("Usage: arbiter [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  version    Display version information")
	fmt.Println("  help       Display this help message")
	fmt.Println()
	fmt.Println("When run without commands, arbiter starts in service mode")
}

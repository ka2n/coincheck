package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"encoding/json"

	"github.com/ka2n/coincheck"
)

const usage = `
usage: coincheck [--help] <command> [<args>]

Available commands are:
	ticker		Check latest ticker
`

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	cmd := os.Args[1]
	if cmd != "ticker" {
		fmt.Fprintln(os.Stderr, "unknown command: "+cmd)
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	client, err := coincheck.New(os.Getenv("COINCHECK_API_KEY"), os.Getenv("COINCHECK_API_SECRET"))
	if err != nil {
		log.Fatal(err)
	}

	if err := tickerCmd(context.Background(), client); err != nil {
		log.Fatal(err)
	}
}

func tickerCmd(ctx context.Context, client *coincheck.Client) error {
	ticker, err := client.Ticker(ctx)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(ticker)
	return nil
}

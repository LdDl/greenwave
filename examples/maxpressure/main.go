package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/LdDl/greenwave/app/simulation"
	"github.com/LdDl/greenwave/maxpressure"
)

//go:embed corridor.json
var requestJSON []byte

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	printRequest := flag.Bool("request", false, "print the complete API request")
	printJSON := flag.Bool("json", false, "print the complete comparison response")
	flag.Parse()
	if *printRequest {
		_, err := os.Stdout.Write(requestJSON)
		return err
	}
	var request dto.MPRunRequest
	if err := json.Unmarshal(requestJSON, &request); err != nil {
		return err
	}
	response, err := simulation.RunMaxPressure(context.Background(), request)
	if err != nil {
		return err
	}
	if *printJSON {
		return json.NewEncoder(os.Stdout).Encode(response)
	}
	fmt.Printf("%-18s %14s %12s %12s %12s\n", "Controller", "Waiting veh*s", "Departed", "Remaining", "Backlog")
	printEvaluation("Original program", response.Baseline)
	printEvaluation("Standard MP", response.StandardMP)
	printEvaluation("Smoothing-MP", response.Evaluation)
	printEvaluation("Returned program", response.ProposalEvaluation)
	fmt.Printf("Improved fixed program accepted: %v\n", response.ProposalAccepted)
	for _, warning := range response.Warnings {
		fmt.Println(warning)
	}
	return nil
}

func printEvaluation(name string, e *maxpressure.Evaluation) {
	fmt.Printf("%-18s %14.1f %12.1f %12.1f %12.1f\n", name, e.TotalDelayVehS, e.Balance.Departed, e.Balance.Remaining, e.Balance.BoundaryBacklog)
}

package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"net/http/httptest"
	"os"

	"github.com/LdDl/greenwave/app/rest"
	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/LdDl/greenwave/maxpressure"
	"github.com/labstack/echo/v4"
)

//go:embed corridor.json
var requestJSON []byte

func main() {
	printRequest := flag.Bool("request", false, "print the complete API request")
	printJSON := flag.Bool("json", false, "print the complete comparison response")
	flag.Parse()
	if *printRequest {
		fmt.Print(string(requestJSON))
		return
	}
	// Exercise the actual HTTP handler in-process without needing a running server.
	server := echo.New()
	request := httptest.NewRequest("POST", "/api/maxpressure/run", bytes.NewReader(requestJSON))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	if err := rest.RequestMPRun()(server.NewContext(request, recorder)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if recorder.Code != 200 {
		fmt.Fprintln(os.Stderr, recorder.Body.String())
		os.Exit(1)
	}
	if *printJSON {
		fmt.Print(recorder.Body.String())
		return
	}
	var response dto.MPRunResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
}

func printEvaluation(name string, e *maxpressure.Evaluation) {
	fmt.Printf("%-18s %14.1f %12.1f %12.1f %12.1f\n", name, e.TotalDelayVehS, e.Balance.Departed, e.Balance.Remaining, e.Balance.BoundaryBacklog)
}

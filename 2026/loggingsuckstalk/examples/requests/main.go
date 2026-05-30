//nolint:forbidigo,noctx,wrapcheck,gosec // no lint this file
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	baseURL = "http://localhost:8080/api/v1"
)

type (
	PageAccount struct {
		Accounts []Account      `json:"accounts"`
		Page     map[string]any `json:"page"`
	}

	// Account represents the structure of an account received from the API.
	Account struct {
		ID     string `json:"id"`
		Amount string `json:"amount"`
	}

	// TransferRequest represents the structure of a transfer request to be sent to the API.
	TransferRequest struct {
		Amount string `json:"amount"`
	}
)

func main() {
	numRequests := flag.Int("n", 50, "Number of transfer requests to generate.")
	allowOverdraft := flag.Bool("o", false, "Allow some requests to attempt transferring more money than available.")
	sendRequests := flag.Bool("s", true, "Send the generated requests to the API. If not set, prints to stdout.")

	flag.Parse()

	fmt.Println("Fetching accounts...")

	accounts, err := getAccounts()
	if err != nil {
		fmt.Printf("Could not fetch accounts: %v. Exiting.\n", err)
		os.Exit(1)
	}

	if len(accounts) < 2 {
		fmt.Println("Need at least two accounts to perform transfers. Exiting.")
		os.Exit(1)
	}

	fmt.Printf("Fetched %d accounts.\n", len(accounts))
	fmt.Println("Generating transfer requests...")

	transferRequests, err := generateTransferRequests(accounts, *numRequests, *allowOverdraft)
	if err != nil {
		fmt.Printf("Error generating transfer requests: %v. Exiting.\n", err)
		os.Exit(1)
	}

	if *sendRequests {
		fmt.Printf("Sending %d transfer requests...\n", len(transferRequests))

		wg := sync.WaitGroup{}
		for _, req := range transferRequests {
			wg.Go(func() {
				if errSend := sendTransferRequest(req.fromID, req.toID, req.request); errSend != nil {
					fmt.Printf("Failed to send request: %v\n", errSend)
				}
			})
		}

		wg.Wait()
	}
}

func getAccounts() ([]Account, error) {
	resp, err := http.Get(fmt.Sprintf("%s/accounts?page=0&size=10", baseURL))
	if err != nil {
		return nil, fmt.Errorf("error fetching accounts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch accounts, status code: %d", resp.StatusCode)
	}

	var pageAccounts PageAccount
	if errDecode := json.NewDecoder(resp.Body).Decode(&pageAccounts); errDecode != nil {
		return nil, fmt.Errorf("error decoding accounts: %w", errDecode)
	}

	return pageAccounts.Accounts, nil
}

type transferData struct {
	fromID  string
	toID    string
	request TransferRequest
}

func generateTransferRequests(accounts []Account, numRequests int, allowOverdraft bool) ([]transferData, error) {
	if len(accounts) == 0 {
		return nil, errors.New("no accounts available to generate transfers")
	}

	transfers := make([]transferData, numRequests)
	accountIDs := make([]string, len(accounts))
	accountBalances := make(map[string]decimal.Decimal)

	for i, acc := range accounts {
		accountIDs[i] = acc.ID

		balance, err := decimal.NewFromString(acc.Amount)
		if err != nil {
			return nil, fmt.Errorf("error parsing amount %s: %w", acc.Amount, err)
		}

		accountBalances[acc.ID] = balance
	}

	for i := range numRequests {
		fromAccountID := accountIDs[rand.IntN(len(accountIDs))]
		toAccountID := accountIDs[rand.IntN(len(accountIDs))]

		for fromAccountID == toAccountID {
			toAccountID = accountIDs[rand.IntN(len(accountIDs))]
		}

		fromAccountBalance := accountBalances[fromAccountID]

		var amount decimal.Decimal

		if allowOverdraft && rand.Float64() < 0.3 { // 30% chance to try overdraft
			amount = fromAccountBalance.Mul(decimal.NewFromFloat(1.1 + rand.Float64()*2.0))
		} else {
			if fromAccountBalance.GreaterThan(decimal.NewFromInt(1)) {
				amount = fromAccountBalance.Mul(decimal.NewFromFloat(rand.Float64() * 0.9))
			} else {
				amount = decimal.NewFromFloat(1.0)
			}
		}

		transfers[i] = transferData{
			fromID: fromAccountID,
			toID:   toAccountID,
			request: TransferRequest{
				Amount: amount.Round(2).StringFixed(2),
			},
		}
	}

	return transfers, nil
}

func sendTransferRequest(fromID, toID string, transferData TransferRequest) error {
	ctx := context.Background()

	jsonData, err := json.Marshal(transferData)
	if err != nil {
		return fmt.Errorf("error marshaling transfer data: %w", err)
	}

	url := fmt.Sprintf("%s/%s/transfers/%s", baseURL, fromID, toID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.NewString())

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending transfer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("[200] Successfully sent transfer: From %s To %s Amount %s\n", fromID, toID, transferData.Amount)
	} else {
		bodyBytes, errReadAll := io.ReadAll(resp.Body)
		if errReadAll != nil {
			return errReadAll
		}

		bodyString := string(bodyBytes)
		fmt.Printf("[%d] Failed to send transfer: %q.\n", resp.StatusCode, bodyString)
	}

	return nil
}

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	rpcURL := "http://localhost:8547"
	privateKeyHex := "59c6995e998f97a5a0044976f9d4b4c2b6e6dcd0bdfbff6af5b7d4b8d7c5c6d0"
	refillEvery := 200
	logBlockSizeEvery := 100

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to Anvil: %v", err)
	}
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("Invalid private key: %v", err)
	}

	publicAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	fmt.Println("Using wallet address:", publicAddr.Hex())

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}

	nonce, err := client.PendingNonceAt(context.Background(), publicAddr)
	if err != nil {
		log.Fatalf("Failed to get starting nonce: %v", err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get gas price: %v", err)
	}

	baseHex := strings.Repeat("deadbeef", 25000) // ~100KB
	baseData, err := hex.DecodeString(baseHex)
	if err != nil {
		log.Fatalf("Failed to decode hex: %v", err)
	}

	fmt.Println("🚀 Starting transaction spam... Ctrl+C to stop.")
	txCount := 0

	for {
		if txCount > 0 && txCount%refillEvery == 0 {
			refillBalance(rpcURL, publicAddr)
		}

		if txCount > 0 && txCount%logBlockSizeEvery == 0 {
			printLatestBlockSizeMB(rpcURL)
		}

		suffix := fmt.Sprintf("%02x", txCount%256)
		payload := append(baseData, suffix...)

		// tx := types.NewTx(&types.LegacyTx{
		// 	Nonce:    nonce,
		// 	GasPrice: new(big.Int).Add(gasPrice, big.NewInt(int64(txCount*1000))),
		// 	Gas:      10_000_000,
		// 	To:       &publicAddr,
		// 	Value:    big.NewInt(0),
		// 	Data:     payload,
		// })

		tx := types.NewTx(&types.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     nonce,
			GasTipCap: big.NewInt(2_000_000_000),                              // 2 gwei tip
			GasFeeCap: new(big.Int).Add(gasPrice, big.NewInt(10_000_000_000)), // gasPrice + 10 gwei buffer
			Gas:       10_000_000,
			To:        &publicAddr,
			Value:     big.NewInt(0),
			Data:      payload,
		})

		signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), privateKey)
		if err != nil {
			log.Fatalf("Failed to sign tx %d: %v", txCount, err)
		}

		// Retry logic for transient send errors
		maxRetries := 5
		for retry := 0; retry < maxRetries; retry++ {
			err = client.SendTransaction(context.Background(), signedTx)
			if err == nil {
				break
			}
			if strings.Contains(err.Error(), "connect: resource temporarily unavailable") {
				wait := time.Duration(500*(retry+1)) * time.Millisecond
				log.Printf("⚠️ Send tx %d failed: %v. Retrying in %v...", txCount, err, wait)
				time.Sleep(wait)
				continue
			} else {
				log.Fatalf("❌ Failed to send tx %d: %v", txCount, err)
			}
		}
		if err != nil {
			log.Fatalf("❌ Giving up after retries. Failed to send tx %d: %v", txCount, err)
		}

		go trackConfirmation(client, signedTx.Hash())

		nonce++
		txCount++
		time.Sleep(50 * time.Millisecond)
	}
}

func trackConfirmation(client *ethclient.Client, txHash common.Hash) {
	for {
		receipt, err := client.TransactionReceipt(context.Background(), txHash)
		if err == nil && receipt != nil {
			fmt.Printf("✅ TX %s mined in block 0x%x\n", txHash.Hex(), receipt.BlockNumber.Uint64())
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func refillBalance(rpcURL string, address common.Address) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "anvil_setBalance",
		"params": []interface{}{
			address.Hex(),
			"0xffffffffffffffffffff",
		},
		"id": 1,
	}

	jsonPayload, _ := json.Marshal(payload)
	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Printf("⚠️ Failed to refill balance: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("💰 Refilled balance for %s\n", address.Hex())
}

func printLatestBlockSizeMB(rpcURL string) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockByNumber",
		"params":  []interface{}{"latest", true},
		"id":      1,
	}

	jsonPayload, _ := json.Marshal(payload)
	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Printf("⚠️ Failed to query latest block size: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Result struct {
			Size string `json:"size"`
		} `json:"result"`
	}
	err = json.Unmarshal(body, &result)
	if err != nil || result.Result.Size == "" {
		fmt.Println("⚠️ Could not parse block size.")
		return
	}

	sizeBytes := new(big.Int)
	sizeBytes.SetString(result.Result.Size[2:], 16) // remove "0x"
	mb := new(big.Float).Quo(new(big.Float).SetInt(sizeBytes), big.NewFloat(1024*1024))
	fmt.Printf("📦 Latest block size: %.2f MB\n", mb)
}

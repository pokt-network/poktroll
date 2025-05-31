package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Coin represents a token with denomination and amount
type Coin struct {
	Denom  string `protobuf:"bytes,1,opt,name=denom,proto3" json:"denom"`
	Amount string `protobuf:"bytes,2,opt,name=amount,proto3" json:"amount"`
}

// MorseClaimableAccount represents a Morse account claimable as part of Morse -> Shannon migration
type MorseClaimableAccount struct {
	ShannonDestAddress string    `json:"shannon_dest_address"`
	MorseSrcAddress    string    `json:"morse_src_address"`
	UnstakedBalance    Coin      `json:"unstaked_balance"`
	SupplierStake      Coin      `json:"supplier_stake"`
	ApplicationStake   Coin      `json:"application_stake"`
	ClaimedAtHeight    int64     `json:"claimed_at_height"`
	MorseOutputAddress string    `json:"morse_output_address,omitempty"`
	UnstakingTime      time.Time `json:"unstaking_time"`
}

// MorseAccountState represents all account state to be migrated from Morse
type MorseAccountState struct {
	Accounts []*MorseClaimableAccount `json:"accounts"`
}

// Required protobuf interface methods
func (m *MorseAccountState) Reset()         { *m = MorseAccountState{} }
func (m *MorseAccountState) String() string { return fmt.Sprintf("%+v", *m) }
func (*MorseAccountState) ProtoMessage()    {}

func (m *MorseClaimableAccount) Reset()         { *m = MorseClaimableAccount{} }
func (m *MorseClaimableAccount) String() string { return fmt.Sprintf("%+v", *m) }
func (*MorseClaimableAccount) ProtoMessage()    {}

func (m *Coin) Reset()         { *m = Coin{} }
func (m *Coin) String() string { return fmt.Sprintf("%+v", *m) }
func (*Coin) ProtoMessage()    {}

// GetHash calculates the sha256 hash by JSON marshaling (simpler approach)
func (m MorseAccountState) GetHash() ([]byte, error) {
	// Use JSON marshaling instead of proto marshaling for simplicity
	morseAccountStateBz, err := json.Marshal(&m)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(morseAccountStateBz)
	return hash[:], nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <filepath>")
		os.Exit(1)
	}

	filePath := os.Args[1]

	// Read the JSON file
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(filePath)

	// Unmarshal into MorseAccountState
	var morseAccountState MorseAccountState
	if err2 := json.Unmarshal(data, &morseAccountState); err2 != nil {
		fmt.Printf("Error unmarshaling JSON: %v\n", err2)
		os.Exit(1)
	}
	fmt.Println(morseAccountState)

	// Calculate hash using the struct's GetHash method
	hash, err := morseAccountState.GetHash()
	if err != nil {
		fmt.Printf("Error calculating hash: %v\n", err)
		os.Exit(1)
	}

	// Output the hash (base64 encoded)
	fmt.Printf("%s\n", base64.StdEncoding.EncodeToString(hash))
}

package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type P2P_Point struct {
}

type ledger struct {
	TrType    string `json:"trType"` // "0" 계좌개설, "1" 입금, "2" 송금(입금), "3" 송금(출금), "4" 출금, "Q" 조회
	AccID     string `json:"accID"`
	Amt       int    `json:"amt"`
	Timestamp string `json:"timestamp"`
	OppAccID  string `json:"oppAccID"`
	Balance   int    `json:"balance"`
}

// ===================================================================================
// Main
// ===================================================================================
func main() {
	err := shim.Start(new(P2P_Point))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init initializes chaincode
// ===========================
func (t *P2P_Point) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke
// ========================================
func (t *P2P_Point) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "0" { //create a new ledger
		return t.initLedger(stub, args)
	} else if function == "1" || function == "2" || function == "3" || function == "4" { //change timestamp of a specific ledger
		return t.transferLedger(stub, args)
	} else if function == "Q" {
		return t.readLedger(stub, args)
	}

	fmt.Println("invoke did not find func: " + function) //error
	return shim.Error("Received unknown function invocation")
}

// ============================================================
// initLedger - create a new ledger, store into chaincode state
// ============================================================
func (t *P2P_Point) initLedger(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	// 0        1
	// "AccID", "Timestamp"
	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2(AccID,Timestamp)")
	}

	// ==== Input sanitation ====
	fmt.Println("- start init ledger")
	if len(args[0]) <= 0 {
		return shim.Error("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("2nd argument must be a non-empty string")
	}
	accID := args[0]
	timestamp := args[1]

	// ==== Check if ledger already exists ====
	ledgerAsBytes, err := stub.GetState(accID)
	if err != nil {
		return shim.Error("Failed to get ledger: " + err.Error())
	} else if ledgerAsBytes != nil {
		fmt.Println("This ledger already exists: " + accID)
		return shim.Error("This ledger already exists: " + accID)
	}

	// ==== Create ledger object and marshal to JSON ====
	trType := "0"
	ledger := &ledger{trType, accID, 0, timestamp, "", 0}
	ledgerJSONasBytes, err := json.Marshal(ledger)
	if err != nil {
		return shim.Error(err.Error())
	}

	// === Save ledger to state ===
	err = stub.PutState(accID, ledgerJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	// ==== Ledger saved and indexed. Return success ====
	fmt.Println("- end init ledger")
	return shim.Success(nil)
}

// ===========================================================
// transfer a ledger by setting a new timestamp name on the ledger
// ===========================================================
func (t *P2P_Point) transferLedger(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	function, _ := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)

	// 0        1      2            3
	// "accID", "amt", "timestamp", "oppAccID"
	if function == "1" || function == "4" { // 입금, 출금
		if len(args) != 3 {
			return shim.Error("Incorrect number of arguments. Expecting 3")
		}
	} else { // 송금(입금), 송금(출금)
		if len(args) != 4 {
			return shim.Error("Incorrect number of arguments. Expecting 4")
		}
	}

	trType := function
	accID := args[0]
	amt, err := strconv.Atoi(args[1])
	if err != nil {
		return shim.Error("2nd argument must be a numeric string")
	}
	timestamp := args[2]
	oppAccID := ""
	if len(args) == 4 {
		oppAccID = args[3]
	}

	fmt.Println("- Start transferLedger ")
	fmt.Println("- trtype : ", trType)
	fmt.Println("- accID : ", accID)
	fmt.Println("- amt : ", args[1])
	fmt.Println("- timestamp : ", timestamp)
	fmt.Println("- oppAccID : ", oppAccID)

	ledgerAsBytes, err := stub.GetState(accID)
	if err != nil {
		return shim.Error("Failed to get ledger:" + err.Error())
	} else if ledgerAsBytes == nil {
		return shim.Error("Ledger does not exist")
	}

	ledgerToTransfer := ledger{}
	err = json.Unmarshal(ledgerAsBytes, &ledgerToTransfer) //unmarshal it aka JSON.parse()
	if err != nil {
		return shim.Error(err.Error())
	}

	befBalance := ledgerToTransfer.Balance
	fmt.Println("- before balance : ", strconv.Itoa(befBalance))

	ledgerToTransfer.TrType = trType
	ledgerToTransfer.Amt = amt
	ledgerToTransfer.Timestamp = timestamp
	ledgerToTransfer.OppAccID = oppAccID

	if function == "1" || function == "2" { // 입금, 송금(입금)
		ledgerToTransfer.Balance = befBalance + amt
	} else { // 출금, 송금(출금)
		if befBalance < amt {
			return shim.Error("There is not enough balance")
		}
		ledgerToTransfer.Balance = befBalance - amt
	}

	aftBalance := ledgerToTransfer.Balance
	fmt.Println("- after balance : ", strconv.Itoa(aftBalance))

	ledgerJSONasBytes, _ := json.Marshal(ledgerToTransfer)
	err = stub.PutState(accID, ledgerJSONasBytes) //rewrite the ledger
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end transferLedger (success)")
	return shim.Success(nil)
}

// ===============================================
// readLedger - read a ledger from chaincode state
// ===============================================
func (t *P2P_Point) readLedger(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var accID, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the ledger to query")
	}

	accID = args[0]
	valAsbytes, err := stub.GetState(accID) //get the ledger from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + accID + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Ledger does not exist: " + accID + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

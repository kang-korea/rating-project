package main

// 1. import
import (
	"encoding/json"
	"fmt"
	"log"
	"time"
	"strings"

	"github.com/golang/protobuf/ptypes"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// 2. chaincode 구조체
// SmartContract provides functions for managing a car
type SmartContract struct {
	contractapi.Contract
}

// 3. Rating 구조체
type Rating struct {
	LedgerId  string `json:"ledgerId"`
	Rate	int `json:"rate"`
	RatedBy  string `json:"ratedBy"`
	OwnerId  string `json:"ownerId"`
}

type HistoryQueryResult struct {
	Record    *Rating    `json:"record"`
	TxId     string    `json:"txId"`
	Timestamp time.Time `json:"timestamp"`
	IsDelete  bool      `json:"isDelete"`
}

// rating_register ( ledgerId string, rate int, ratedBy string)
// ledger id <= userId:serviceType
// default rate is 3 (of 5)
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface, userId string, serviceType string) error {
	ledgerId := strings.Join([]string{userId, serviceType}, ":")

	exists, err := s.LedgerExists(ctx, ledgerId)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the ledger %s already exists", ledgerId)
	}

	rating := Rating{
		LedgerId: ledgerId,
		Rate: 3,
		RatedBy: "init",
		OwnerId: userId,
	}

	ratingAsBytes, _ := json.Marshal(rating)

	return ctx.GetStub().PutState(ledgerId, ratingAsBytes)
}

// ReadLedger
func (s *SmartContract) ReadLedger(ctx contractapi.TransactionContextInterface, ledgerId string) (*Rating, error) {
	ratingJSON, err := ctx.GetStub().GetState(ledgerId)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if ratingJSON == nil {
		return nil, fmt.Errorf("the rating %s does not exist", ledgerId)
	}

	var rating Rating
	err = json.Unmarshal(ratingJSON, &rating)
	if err != nil {
		return nil, err
	}

	return &rating, nil
}

// LedgerExists 
func (s *SmartContract) LedgerExists(ctx contractapi.TransactionContextInterface, ledgerId string) (bool, error) {
	ratingJSON, err := ctx.GetStub().GetState(ledgerId)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return ratingJSON != nil, nil
}

// RegisterRate
func (s *SmartContract) RegisterRate(ctx contractapi.TransactionContextInterface, ledgerId string, rate int, userId string) error {
	
	//TODO::invalid request if there is a record with ledgerId and userId from parameters

	rating := Rating{
		LedgerId: ledgerId,
		Rate: rate,
		RatedBy: userId,
	}

	ratingAsBytes, _ := json.Marshal(rating)

	return ctx.GetStub().PutState(ledgerId, ratingAsBytes)
}

// GetAllRatings
func (t *SmartContract) GetAllRatings(ctx contractapi.TransactionContextInterface, ledgerId string) ([]HistoryQueryResult, error) {
	log.Printf("GetAllRatings: Ledger ID %v", ledgerId) // 체인코드 컨테이너 -> docker logs dev-asset1...

	resultsIterator, err := ctx.GetStub().GetHistoryForKey(ledgerId)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var records []HistoryQueryResult
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var rating Rating
		if len(response.Value) > 0 {
			err = json.Unmarshal(response.Value, &rating)
			if err != nil {
				return nil, err
			}
		} else {
			rating = Rating{
				LedgerId: ledgerId,
			}
		}

		timestamp, err := ptypes.Timestamp(response.Timestamp)
		if err != nil {
			return nil, err
		}

		record := HistoryQueryResult{
			TxId:      response.TxId,
			Timestamp: timestamp,
			Record:    &rating,
			IsDelete:  response.IsDelete,
		}
		records = append(records, record)
	}

	return records, nil
}

// 5. main
func main() {

	chaincode, err := contractapi.NewChaincode(new(SmartContract))

	if err != nil {
		fmt.Printf("Error create goods chaincode: %s", err.Error())
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting goods chaincode: %s", err.Error())
	}
}
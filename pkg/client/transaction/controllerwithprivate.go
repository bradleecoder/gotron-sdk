package transaction

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	proto "github.com/golang/protobuf/proto"
)

// ControllerWithPrivateKey drives the transaction signing process
type ControllerWithPrivateKey struct {
	executionError   error
	resultError      error
	client           *client.GrpcClient
	tx               *core.Transaction
	senderPrivateKey string
	Behavior         behavior
	Result           *api.Return
	Receipt          *core.TransactionInfo
}

// NewControllerWithPrivateKey initializes a Controller, caller can control behavior via options
func NewControllerWithPrivateKey(
	client *client.GrpcClient,
	senderPrivateKey string,
	tx *core.Transaction,
) *ControllerWithPrivateKey {

	ctrlr := &ControllerWithPrivateKey{
		executionError:   nil,
		resultError:      nil,
		client:           client,
		senderPrivateKey: senderPrivateKey,
		tx:               tx,
		Behavior:         behavior{false, Software, 0},
	}
	return ctrlr
}

func (C *ControllerWithPrivateKey) signTxForSending() {
	if C.executionError != nil {
		return
	}
	//signedTransaction, err :=
	//	C.sender.ks.SignTx(*C.sender.account, C.tx)
	//sign and broadcast
	h256h := sha256.New()
	h256h.Write(C.tx.RawData.Data)
	hash := h256h.Sum(nil)
	pri, err := crypto.HexToECDSA(C.senderPrivateKey)
	if err != nil {
		C.executionError = err
		return
	}
	signature, err := crypto.Sign(hash, pri)
	if err != nil {
		C.executionError = err
		return
	}
	C.tx.Signature = append(C.tx.Signature, signature)

	if err != nil {
		C.executionError = err
	}
	//C.tx = signedTransaction
}

// TransactionHash extract hash from TX
func (C *ControllerWithPrivateKey) TransactionHash() (string, error) {
	rawData, err := C.GetRawData()
	if err != nil {
		return "", err
	}
	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)
	return common.ToHex(hash), nil
}

func (C *ControllerWithPrivateKey) txConfirmation() {
	if C.executionError != nil || C.Behavior.DryRun {
		return
	}
	if C.Behavior.ConfirmationWaitTime > 0 {
		txHash, err := C.TransactionHash()
		if err != nil {
			C.executionError = fmt.Errorf("could not get tx hash")
			return
		}
		//fmt.Printf("TX hash: %s\nWaiting for confirmation....", txHash)
		start := int(C.Behavior.ConfirmationWaitTime)
		for {
			// GETTX by ID
			if txi, err := C.client.GetTransactionInfoByID(txHash); err == nil {
				// check receipt
				if txi.Result != 0 {
					C.resultError = fmt.Errorf("%s", txi.ResMessage)
				}
				// Add receipt
				C.Receipt = txi
				return
			}
			if start < 0 {
				C.executionError = fmt.Errorf("could not confirm transaction after %d seconds", C.Behavior.ConfirmationWaitTime)
				return
			}
			time.Sleep(time.Second)
			start--
		}
	} else {
		C.Receipt = &core.TransactionInfo{}
		C.Receipt.Receipt = &core.ResourceReceipt{}
	}

}

// GetResultError return result error
func (C *ControllerWithPrivateKey) GetResultError() error {
	return C.resultError
}

// ExecuteTransaction is the single entrypoint to execute a plain transaction.
// Each step in transaction creation, execution probably includes a mutation
// Each becomes a no-op if executionError occurred in any previous step
func (C *ControllerWithPrivateKey) ExecuteTransaction() error {
	switch C.Behavior.SigningImpl {
	case Software:
		C.signTxForSending()
		//case Ledger:
		//	C.hardwareSignTxForSending()
	}
	C.sendSignedTx()
	C.txConfirmation()
	return C.executionError
}

// GetRawData Byes from Transaction
func (C *ControllerWithPrivateKey) GetRawData() ([]byte, error) {
	return proto.Marshal(C.tx.GetRawData())
}

func (C *ControllerWithPrivateKey) sendSignedTx() {
	if C.executionError != nil || C.Behavior.DryRun {
		return
	}
	result, err := C.client.Broadcast(C.tx)
	if err != nil {
		C.executionError = err
		return
	}
	if result.Code != 0 {
		C.executionError = fmt.Errorf("bad transaction: %v", string(result.GetMessage()))
	}
	C.Result = result
}

package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	ethereumabi "github.com/ethereum/go-ethereum/accounts/abi"
	ethereumcommon "github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// TriggerConstantContract and return tx result
func (g *GrpcClient) TriggerConstantContract(from, contractAddress, method, jsonString string) (*api.TransactionExtention, error) {
	var err error
	fromDesc := address.HexToAddress("410000000000000000000000000000000000000000")
	if len(from) > 0 {
		fromDesc, err = address.Base58ToAddress(from)
		if err != nil {
			return nil, err
		}
	}
	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	param, err := abi.LoadFromJSON(jsonString)
	if err != nil {
		return nil, err
	}

	dataBytes, err := abi.Pack(method, param)
	if err != nil {
		return nil, err
	}

	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            dataBytes,
	}

	return g.triggerConstantContract(ct)
}

// triggerConstantContract and return tx result
func (g *GrpcClient) triggerConstantContract(ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.grpcTimeout)
	defer cancel()

	return g.Client.TriggerConstantContract(ctx, ct)
}

// TriggerContract and return tx result
func (g *GrpcClient) TriggerContract(from, contractAddress, method, jsonString string,
	feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
	fromDesc, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	param, err := abi.LoadFromJSON(jsonString)
	if err != nil {
		return nil, err
	}

	dataBytes, err := abi.Pack(method, param)
	if err != nil {
		return nil, err
	}

	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            dataBytes,
	}
	if tAmount > 0 {
		ct.CallValue = tAmount
	}
	if len(tTokenID) > 0 && tTokenAmount > 0 {
		ct.CallTokenValue = tTokenAmount
		ct.TokenId, err = strconv.ParseInt(tTokenID, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	return g.triggerContract(ct, feeLimit)
}

// TriggerContract and return tx result
func (g *GrpcClient) TriggerContractWithParamBytes(from, contractAddress string, dataBytes []byte ,
	feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
	fromDesc, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            dataBytes,
	}
	if tAmount > 0 {
		ct.CallValue = tAmount
	}
	if len(tTokenID) > 0 && tTokenAmount > 0 {
		ct.CallTokenValue = tTokenAmount
		ct.TokenId, err = strconv.ParseInt(tTokenID, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	return g.triggerContract(ct, feeLimit)
}

// triggerContract and return tx result
func (g *GrpcClient) triggerContract(ct *core.TriggerSmartContract, feeLimit int64) (*api.TransactionExtention, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.grpcTimeout)
	defer cancel()

	tx, err := g.Client.TriggerContract(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}
	if feeLimit > 0 {
		tx.Transaction.RawData.FeeLimit = feeLimit
		// update hash
		g.UpdateHash(tx)
	}
	return tx, err
}

// DeployContract and return tx result
func (g *GrpcClient) DeployContract(from, contractName string,
	abi *core.SmartContract_ABI, codeStr string,
	feeLimit, curPercent, oeLimit int64,
) (*api.TransactionExtention, error) {

	var err error

	fromDesc, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}

	if curPercent > 100 || curPercent < 0 {
		return nil, fmt.Errorf("consume_user_resource_percent should be >= 0 and <= 100")
	}
	if oeLimit <= 0 {
		return nil, fmt.Errorf("origin_energy_limit must > 0")
	}

	bc, err := common.FromHex(codeStr)
	if err != nil {
		return nil, err
	}

	ct := &core.CreateSmartContract{
		OwnerAddress: fromDesc.Bytes(),
		NewContract: &core.SmartContract{
			OriginAddress:              fromDesc.Bytes(),
			Abi:                        abi,
			Name:                       contractName,
			ConsumeUserResourcePercent: curPercent,
			OriginEnergyLimit:          oeLimit,
			Bytecode:                   bc,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), g.grpcTimeout)
	defer cancel()

	tx, err := g.Client.DeployContract(ctx, ct)
	if err != nil {
		return nil, err
	}
	if feeLimit > 0 {
		tx.Transaction.RawData.FeeLimit = feeLimit
		// update hash
		g.UpdateHash(tx)
	}
	return tx, err
}

// DeployContract and return tx result
func (g *GrpcClient) DeployContractWithArguments(from, contractName string, abiEthereum *ethereumabi.ABI,
	abi *core.SmartContract_ABI, codeStr string, amount int64,
	feeLimit, curPercent, oeLimit int64, args ...interface{}) (*api.TransactionExtention, error) {
	var err error

	fromDesc, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}

	if curPercent > 100 || curPercent < 0 {
		return nil, fmt.Errorf("consume_user_resource_percent should be >= 0 and <= 100")
	}
	if oeLimit <= 0 {
		return nil, fmt.Errorf("origin_energy_limit must > 0")
	}

	bc, err := common.FromHex(codeStr)
	if err != nil {
		return nil, err
	}
	if len(args) > 0 {
		var err error
		args, err = encodeArgument(abiEthereum.Constructor, args)
		if err != nil {
			return nil, err
		}
		packed, err := abiEthereum.Pack("", args...)
		if err != nil {
			return nil, fmt.Errorf("failed to pack args")
		}
		bc = append(bc, packed...)
	}
	ct := &core.CreateSmartContract{
		OwnerAddress: fromDesc.Bytes(),
		NewContract: &core.SmartContract{
			OriginAddress:              fromDesc.Bytes(),
			Abi:                        abi,
			Name:                       contractName,
			ConsumeUserResourcePercent: curPercent,
			OriginEnergyLimit:          oeLimit,
			Bytecode:                   bc,
			CallValue:                  amount,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), g.grpcTimeout)
	defer cancel()

	tx, err := g.Client.DeployContract(ctx, ct)
	if err != nil {
		return nil, err
	}
	if feeLimit > 0 {
		tx.Transaction.RawData.FeeLimit = feeLimit
		// update hash
		g.UpdateHash(tx)
	}
	return tx, err
}

func encodeArgument(method ethereumabi.Method, args []interface{}) ([]interface{}, error) {
	if len(method.Inputs) != len(args) {
		return nil, errors.New("the number of arguments is not correct")
	}
	newArgs := make([]interface{}, len(args))
	for index, input := range method.Inputs {
		switch input.Type.String() {
		case "address":
			var err error
			newArgs[index], err = addressTypeAssert(args[index])
			if err != nil {
				return nil, err
			}
		case "address[]":
			s := reflect.ValueOf(args[index])
			if s.Kind() != reflect.Slice && s.Kind() != reflect.Array {
				return nil, fmt.Errorf("fail because the type is non-slice, non-array")
			}
			newArr := make([]ethereumcommon.Address, s.Len())
			for j := 0; j < s.Len(); j++ {
				var err error
				newArr[j], err = addressTypeAssert(s.Index(j).Interface())
				if err != nil {
					return nil, err
				}
			}
			newArgs[index] = newArr
		default:
			newArgs[index] = args[index]
		}
	}
	return newArgs, nil
}

func addressTypeAssert(preVal interface{}) (ethereumcommon.Address, error) {
	switch v := preVal.(type) {
	case string:
		tronAddress, err := address.Base58ToAddress(v)
		if err != nil {
			return ethereumcommon.Address{}, fmt.Errorf("fail to convert string to ioAddress")
		}
		return ethereumcommon.HexToAddress(hex.EncodeToString(tronAddress.Bytes())), nil
	case address.Address:
		return ethereumcommon.HexToAddress(hex.EncodeToString(v.Bytes())), nil
	case ethereumcommon.Address:
		return v, nil
	default:
		return ethereumcommon.Address{}, fmt.Errorf("fail to convert from interface to string/ioAddress/ethAddress")
	}
}

// UpdateHash after local changes
func (g *GrpcClient) UpdateHash(tx *api.TransactionExtention) error {
	rawData, err := proto.Marshal(tx.Transaction.GetRawData())
	if err != nil {
		return err
	}

	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)
	tx.Txid = hash
	return nil
}

// GetContractABI return smartContract
func (g *GrpcClient) GetContractABI(contractAddress string) (*core.SmartContract_ABI, error) {
	var err error
	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), g.grpcTimeout)
	defer cancel()

	sm, err := g.Client.GetContract(ctx, GetMessageBytes(contractDesc))
	if err != nil {
		return nil, err
	}
	if sm == nil {
		return nil, fmt.Errorf("invalid contract abi")
	}

	return sm.Abi, nil
}

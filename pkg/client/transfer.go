package client

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

// Transfer from to base58 address
func (g *GrpcClient) Transfer(from, toAddress string, amount int64) (*api.TransactionExtention, error) {
	var err error

	contract := &core.TransferContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}
	if contract.ToAddress, err = common.DecodeCheck(toAddress); err != nil {
		return nil, err
	}
	contract.Amount = amount

	ctx, cancel := context.WithTimeout(context.Background(), g.grpcTimeout)
	defer cancel()

	tx, err := g.Client.CreateTransaction2(ctx, contract)
	if err != nil {
		return nil, err
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	if tx.GetResult().GetCode() != 0 {
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}
	return tx, nil
}

// Transfer from to base58 address
func (g *GrpcClient) TransferWithDemo(privateKey, from, toAddress string, amount int64, memo string) (ret string, err error) {
	transferContract := &core.TransferContract{}
	if transferContract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return
	}
	if transferContract.ToAddress, err = common.DecodeCheck(toAddress); err != nil {
		return
	}
	transferContract.Amount = amount

	any, err := ptypes.MarshalAny(transferContract)
	if err != nil {
		return
	}
	transactionContracts := []*core.Transaction_Contract{{Type: core.Transaction_Contract_TransferContract, Parameter: any}}

	hei, txHash, err := g.lastBlockHeightHash()
	if err != nil {
		return
	}
	now := time.Now().Unix() * 1e3
	rawTransaction := &core.TransactionRaw{
		RefBlockBytes: hei,
		RefBlockHash:  txHash,
		Expiration:    now + 10*60*1e3,
		Timestamp:     now,
		FeeLimit:      100000000,
		Data:          []byte(memo),
		Contract:      transactionContracts,
	}

	coreTrx := &core.Transaction{RawData: rawTransaction, Signature: make([][]byte, 0)}

	//marshal and sign send
	rawData, err := proto.Marshal(coreTrx.GetRawData())
	if err != nil {
		return
	}
	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)
	pri, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return
	}
	signature, err := crypto.Sign(hash, pri)
	if err != nil {
		return
	}

	coreTrx.Signature = append(coreTrx.Signature, signature)

	ret = common.BytesToHexString(hash)

	res, err := g.Client.BroadcastTransaction(context.Background(), coreTrx)
	if err != nil {
		return
	}
	if res.Code != 0 {
		err = fmt.Errorf("bad transaction: %v", string(res.GetMessage()))
		return
	}

	return
}

func (g *GrpcClient) lastBlockHeightHash() (hei, txid []byte, err error) {
	bl, err := g.Client.GetNowBlock2(context.Background(), new(api.EmptyMessage))
	if err != nil {
		return
	}
	heiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heiBytes, uint64(bl.BlockHeader.RawData.Number))
	hei = heiBytes[6:8]

	rawData, err := proto.Marshal(bl.BlockHeader.RawData)
	if err != nil {
		return
	}
	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)
	txid = hash[8:16]
	return
}

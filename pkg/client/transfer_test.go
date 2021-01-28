package client

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	endpoint = "grpc.shasta.trongrid.io:50051"
	pri      = "68596c63d7230acc0916245c0b419978c831051d83ea46f3611a23d91a0c2b34"
	from     = "TRZar2KkBCxJw3kPHScNUf11bHPV7gj67r"
	to       = "TYd8oTYpaE7YJR4sS7HT8tRzAFUG8RqwxD"
	amount   = 100000
)

func TestTransfer(t *testing.T) {
	require := require.New(t)
	conn := NewGrpcClient(endpoint, 10*time.Second)
	err := conn.Start()
	require.NoError(err)
	tx, err := conn.TransferWithDemo(pri, from, to, amount, "tron wbb TYd8oTYpaE7YJR4sS7HT8tRzAFUG8RqwxD")
	require.NoError(err)
	fmt.Println("tx:", tx)
}

func TestIntToByte(t *testing.T) {
	x := uint64(2541403)
	heiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heiBytes, x)
	hei := heiBytes[6:8]
	fmt.Println(hex.EncodeToString(hei))
}

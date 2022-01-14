/*
Copyright LLC Newity. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package parser

import (
	"crypto/sha256"
	"encoding/base64"
	"io/ioutil"
	"testing"

	protos "github.com/SmartBFT-Go/consensus/smartbftprotos"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestBlockInfoFromProposal(t *testing.T) {
	proposalBytes, err := ioutil.ReadFile("../mock/proposal.mock")
	assert.NoError(t, err)

	var proposal protos.Proposal

	assert.NoError(t, proto.Unmarshal(proposalBytes, &proposal))
	block, err := BlockInfoFromProposal(&proposal)
	assert.NoError(t, err)

	h := sha256.New()
	_, err = h.Write(block.Txs[0].Creator.Cert)
	assert.NoError(t, err)

	hash := h.Sum(nil)

	assert.Equal(t, false, block.IsConfig)
	assert.Equal(
		t,
		"MduC4DvZcdIa2UxyMwnYo6XS2s57i+3hYNCv0jh2Z98=",
		base64.StdEncoding.EncodeToString(block.PreviousBlockHash),
	)
	assert.Equal(t, "gxHtvvogGbLa3rq1NDKqFrDRMXHMwht6awf1YBk6b3Q=", base64.StdEncoding.EncodeToString(block.Hash))
	assert.Equal(t, "lscc", block.Txs[0].ChaincodeID.ChaincodeName)
	assert.Equal(t, "261153e0cb71f7b3e9b122378d67e470f55c9e54d74f2091cfe2ecb80043b988", block.Txs[0].TxID)
	assert.Equal(t, "atomyzeMSP", block.Txs[0].Creator.MspID)
	assert.Equal(t, "+anv67cafUwr/Avnki/RYU3l4s65LIoPKQ6DRZrIgZQ=", base64.StdEncoding.EncodeToString(hash))
}

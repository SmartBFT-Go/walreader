/*
Copyright LLC Newity. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package parser

import (
	"encoding/asn1"
	"fmt"
	"math/big"
	"time"

	"github.com/SmartBFT-Go/consensus/pkg/types"
	protos "github.com/SmartBFT-Go/consensus/smartbftprotos"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/newity/crawler/blocklib"
	"github.com/pkg/errors"
)

type Block struct {
	Txs               []Tx
	IsConfig          bool
	Hash              []byte
	PreviousBlockHash []byte
	BlockNumber       uint64
}

type Tx struct {
	TxID             string
	Creator          *Creator
	CreatorSignature string
	ChaincodeID      ChaincodeID
	ChannelID        string
	Timestamp        time.Time
}

type Creator struct {
	MspID string
	Cert  []byte
}

type ChaincodeID struct {
	ChaincodeName    string
	ChaincodeVersion int32
}

func BlockInfo(fabBlock *common.Block) (*Block, error) {
	block, err := blocklib.FromBFTFabricBlockWithoutOrdererIdenities(fabBlock)
	if err != nil {
		return nil, err
	}

	txs, err := block.TxsFromOrdererBlock()
	if err != nil {
		return nil, err
	}

	transactions := make([]Tx, 0, len(txs))

	for _, tx := range txs {
		mspID, pemCert, err := tx.Creator()
		if err != nil {
			return nil, err
		}

		creatorSign, err := tx.CreatorSignatureHexString()
		if err != nil {
			return nil, fmt.Errorf("failed to get creator's signature: " + err.Error())
		}

		chHdr, err := tx.ChannelHeader()
		if err != nil {
			return nil, err
		}

		var chaincodeID ChaincodeID
		// skip chaincode name and chaincode ID extraction if it's not chaincode tx
		if !block.IsConfig() {
			chaincode, err := tx.ChaincodeId()
			if err != nil {
				return nil, fmt.Errorf("failed to get chaincode ID: " + err.Error())
			}

			chaincodeID = ChaincodeID{
				ChaincodeName:    chaincode.Name,
				ChaincodeVersion: chHdr.Version,
			}
		}

		transactions = append(transactions, Tx{chHdr.TxId, &Creator{
			MspID: mspID,
			Cert:  pemCert,
		}, creatorSign, chaincodeID, chHdr.ChannelId, chHdr.Timestamp.AsTime()})
	}

	return &Block{
		Txs:               transactions,
		IsConfig:          block.IsConfig(),
		Hash:              block.HeaderHash(),
		PreviousBlockHash: block.PreviousHash(),
		BlockNumber:       block.Number(),
	}, nil
}

func BlockInfoFromProposal(proposal *protos.Proposal) (*Block, error) {
	fabBlock, err := ProposalToBlock(proposal)
	if err != nil {
		return nil, err
	}

	return BlockInfo(fabBlock)
}

func GetHeader(bytes []byte) (*common.Header, error) {
	hdr := &common.Header{}
	err := proto.Unmarshal(bytes, hdr)

	return hdr, errors.Wrap(err, "error unmarshaling Header")
}

func ProposalToBlock(prop *protos.Proposal) (*common.Block, error) {
	proposal := types.Proposal{
		Payload:              prop.Payload,
		Header:               prop.Header,
		Metadata:             prop.Metadata,
		VerificationSequence: int64(prop.VerificationSequence),
	}

	// initialize block with empty fields
	block := &common.Block{
		Data:     &common.BlockData{},
		Metadata: &common.BlockMetadata{},
	}

	if len(proposal.Header) == 0 {
		return nil, errors.New("proposal header cannot be nil")
	}

	hdr := &asn1Header{}

	if _, err := asn1.Unmarshal(proposal.Header, hdr); err != nil {
		return nil, errors.Wrap(err, "bad header")
	}

	block.Header = &common.BlockHeader{
		Number:       hdr.Number.Uint64(),
		PreviousHash: hdr.PreviousHash,
		DataHash:     hdr.DataHash,
	}

	if len(proposal.Payload) == 0 {
		return nil, errors.New("proposal payload cannot be nil")
	}

	tuple := &ByteBufferTuple{}
	if err := tuple.FromBytes(proposal.Payload); err != nil {
		return nil, errors.Wrap(err, "bad payload and metadata tuple")
	}

	if err := proto.Unmarshal(tuple.A, block.Data); err != nil {
		return nil, errors.Wrap(err, "bad payload")
	}

	if err := proto.Unmarshal(tuple.B, block.Metadata); err != nil {
		return nil, errors.Wrap(err, "bad metadata")
	}

	return block, nil
}

type asn1Header struct {
	Number       *big.Int
	PreviousHash []byte
	DataHash     []byte
}

type ByteBufferTuple struct {
	A []byte
	B []byte
}

func (bbt *ByteBufferTuple) ToBytes() []byte {
	bytes, err := asn1.Marshal(*bbt)
	if err != nil {
		panic(err)
	}

	return bytes
}

func (bbt *ByteBufferTuple) FromBytes(bytes []byte) error {
	_, err := asn1.Unmarshal(bytes, bbt)

	return err
}

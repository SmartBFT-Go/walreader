/*
Copyright LLC Newity. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package reader

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SmartBFT-Go/consensus/pkg/api"
	"github.com/SmartBFT-Go/consensus/pkg/wal"
	protos "github.com/SmartBFT-Go/consensus/smartbftprotos"
	"github.com/SmartBFT-Go/walreader/parser"
	"github.com/golang/protobuf/proto"
)

type Reader struct {
	logger api.Logger
	path   string
}

func NewReader(logger api.Logger, pathToFile string) *Reader {
	return &Reader{logger, pathToFile}
}

func (r *Reader) getRecords(filePath string) ([][]byte, error) {
	logreader, err := wal.NewLogRecordReader(r.logger, filePath)
	if err != nil {
		return nil, err
	}

	items := make([][]byte, 0)

	var (
		readErr       error
		recordCounter int32
	)

	for {
		var rec *protos.LogRecord

		recordCounter++

		rec, readErr = logreader.Read()
		if readErr != nil {
			logreader.Close()

			break
		}

		if rec.Type == protos.LogRecord_ENTRY {
			items = append(items, rec.Data)
		}

		r.logger.Debugf("Read record #%d, file: %s", recordCounter, filePath)
	}

	switch {
	case errors.Is(readErr, io.EOF):
		r.logger.Debugf("Reached EOF, finished reading file; CRC: %08X", logreader.CRC())

		readErr = nil
	case errors.Is(readErr, io.ErrUnexpectedEOF) || errors.Is(readErr, wal.ErrCRC):
		readErr = fmt.Errorf(
			"received an error in the file, this can possibly be repaired; file: %s; error: %w",
			r.path, err,
		)
	case readErr == nil:
	default:
		readErr = fmt.Errorf("failed reading file: %s; error: %w", r.path, readErr)
	}

	return items, readErr
}

func (r *Reader) ReadFile(filePath string) error {
	items, err := r.getRecords(filePath)
	if err != nil {
		r.logger.Debugf(err.Error())
	}

	for index, item := range items {
		savedMessage := protos.SavedMessage{}
		if err = proto.Unmarshal(item, &savedMessage); err != nil {
			return err
		}

		switch savedMessage.Content.(type) {
		case *protos.SavedMessage_Commit:
			commit := savedMessage.GetCommit().GetCommit()
			r.logger.Infof(
				"print #%v record (Commit)\nDigest: %v View: %v Sequence: %v\nSignature: <signer: %v base64-encoded signature: %v>\n",
				index, commit.Digest, commit.View, commit.Seq,
				commit.Signature.Signer,
				base64.StdEncoding.EncodeToString(commit.Signature.Value),
			)
		case *protos.SavedMessage_NewView:
			view := savedMessage.GetNewView()
			r.logger.Infof("print #%v record (New view)\n%v\n", index, view)
		case *protos.SavedMessage_ViewChange:
			viewChange := savedMessage.GetNewView()
			r.logger.Infof("print #%v record (View change)\n%v\n", index, viewChange)
		case *protos.SavedMessage_ProposedRecord:
			proposedRecord := savedMessage.GetProposedRecord()
			metadata := protos.ViewMetadata{}

			if err = proto.Unmarshal(proposedRecord.PrePrepare.Proposal.Metadata, &metadata); err != nil {
				return err
			}

			block, err := parser.BlockInfoFromProposal(proposedRecord.PrePrepare.Proposal)
			if err != nil {
				return err
			}

			r.logger.Infof("print #%v record (Proposed record)\nPreprepare: <view: %v, sequence: %v, payload of %v bytes>"+
				"metadata: <last sequesnce: %v, view ID: %v>, verification sequence: %v>\n"+
				"Prepare: <view: %v, sequence: %v, assist: %v, digest: %v>\n"+
				"Proposed block: <config: %v, hash: %v, previous hash: %v, block number: %v>",
				index, proposedRecord.PrePrepare.View,
				proposedRecord.PrePrepare.Seq, len(proposedRecord.PrePrepare.Proposal.Payload),
				metadata.LatestSequence, metadata.ViewId,
				proposedRecord.PrePrepare.Proposal.VerificationSequence,
				proposedRecord.Prepare.View, proposedRecord.Prepare.Seq,
				proposedRecord.Prepare.Assist, proposedRecord.Prepare.Digest,
				block.IsConfig, base64.StdEncoding.EncodeToString(block.Hash),
				base64.StdEncoding.EncodeToString(block.PreviousBlockHash), block.BlockNumber)
			r.logger.Infof("Transactions from block %v:", block.BlockNumber)

			for _, tx := range block.Txs {
				r.logger.Infof(
					"tx ID: %v, creator's MSP ID: %v, creator's signature: %v \ncreator's cert:\n %v\nchaincode ID: %v, chaincode version: %v, channel ID: %v, timestamp: %v\n",
					tx.TxID, tx.Creator.MspID, tx.CreatorSignature, string(tx.Creator.Cert),
					tx.ChaincodeID.ChaincodeName, tx.ChaincodeID.ChaincodeVersion,
					tx.ChannelID, tx.Timestamp.Local())
			}
		}
	}

	return nil
}

func (r *Reader) ReadDir() error {
	return filepath.Walk(r.path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				r.logger.Infof("Read %v", path)
				if err = r.ReadFile(path); err != nil {
					return err
				}
			}

			return nil
		},
	)
}

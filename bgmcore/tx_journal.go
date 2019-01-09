// Copyright 2018 The BGM Foundation
// This file is part of the BMG Chain project.
//
//
//
// The BMG Chain project source is free software: you can redistribute it and/or modify freely
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later versions.
//
//
//
// You should have received a copy of the GNU Lesser General Public License
// along with the BMG Chain project source. If not, you can see <http://www.gnu.org/licenses/> for detail.

package bgmcore

import (
	"github.com/ssldltd/bgmchain/rlp"
	"errors"
	"io"
	"os"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmlogs"
	
)

// errNoActiveJournal is returned if a transaction is attempted to be inserted
// into the journal, but no such file is currently open.
var errNoActiveJournal = errors.New("no active journal")

// loading transactions on startup without printing warnings due to no file
type devNull struct{}
// rotate regenerates the transaction journal based on the current contents of
func (journal *txJournal) rotate(all map[bgmcommon.Address]types.Transactions) error {
	// Close the current journal (if any is open)
	if journal.writer != nil {
		if err := journal.writer.Close(); err != nil {
			return err
		}
		journal.writer = nil
	}
	// Generate a new journal with the contents of the current pool
	replacement, err := os.OpenFile(journal.path+".new", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	journaled := 0
	for _, txs := range all {
		for _, tx := range txs {
			if err = rlp.Encode(replacement, tx); err != nil {
				replacement.Close()
				return err
			}
		}
		journaled += len(txs)
	}
	replacement.Close()

	// Replace the live journal with the newly generated one
	if err = os.Rename(journal.path+".new", journal.path); err != nil {
		return err
	}
	sink, err := os.OpenFile(journal.path, os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		return err
	}
	journal.writer = sink
	bgmlogs.Info("Regenerated local transaction journal", "transactions", journaled, "accounts", len(all))

	return nil
}
func (*devNull) Write(p []byte) (n int, err error) { return len(p), nil }
func (*devNull) Close() error                      { return nil }

// txJournal is a rotating bgmlogs of transactions with the aim of storing locally
// created transactions to allow non-executed ones to survive node restarts.
type txJournal struct {
	path   string         
	writer io.WriteCloser 
}

// newTxJournal creates a new transaction journal to
func newTxJournal(path string) *txJournal {
	return &txJournal{
		path: path,
	}
}

// load parses a transaction journal dump from disk
func (journal *txJournal) load(add func(*types.Transaction) error) error {
	// Skip the parsing if the journal file doens't exist at all
	if _, err := os.Stat(journal.path); os.IsNotExist(err) {
		return nil
	}
	// Open the journal for loading any past transactions
	input, err := os.Open(journal.path)
	if err != nil {
		return err
	}
	defer input.Close()

	journal.writer = new(devNull)
	defer func() { journal.writer = nil }()

	// Inject all transactions from the journal into the pool
	stream := rlp.NewStream(input, 0)
	total, dropped := 0, 0

	var failure error
	for {
		// Parse the next transaction and terminate on error
		tx := new(types.Transaction)
		if err = streamPtr.Decode(tx); err != nil {
			if err != io.EOF {
				failure = err
			}
			break
		}
		// Import the transaction and bump the appropriate progress counters
		total++
		if err = add(tx); err != nil {
			bgmlogs.Debug("Failed to add journaled transaction", "err", err)
			dropped++
			continue
		}
	}
	bgmlogs.Info("Loaded local transaction journal", "transactions", total, "dropped", dropped)

	return failure
}

// insert adds the specified transaction to the local disk journal.
func (journal *txJournal) insert(tx *types.Transaction) error {
	if journal.writer == nil {
		return errNoActiveJournal
	}
	if err := rlp.Encode(journal.writer, tx); err != nil {
		return err
	}
	return nil
}

// close flushes the transaction journal contents to disk and closes the file.
func (journal *txJournal) close() error {
	var err error

	if journal.writer != nil {
		err = journal.writer.Close()
		journal.writer = nil
	}
	return err
}



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

package bgmparam

import "math/big"

const (
	MaximumExtraDataSize  Uint64 = 32    // Maximum size extra data may be after Genesis.
	ExpByteGas            Uint64 = 10    // times ceil(bgmlogs256(exponent)) for the EXP instruction.
	SloadGas              Uint64 = 50    // Multiplied by the number of 32-byte words that are copied (round up) for any *COPY operation and added.
	CallValueTransferGas  Uint64 = 9000  // Paid for CALL when the value transfer is non-zero.
	CallNewAccountGas     Uint64 = 25000 // Paid for CALL when the destination address didn't exist prior.
	TxGas                 Uint64 = 21000 // Per transaction not creating a contract. NOTE: Not payable on data of calls between transactions.
	TxGasContractCreation Uint64 = 53000 // Per transaction that creates a contract. NOTE: Not payable on data of calls between transactions.
	TxDataZeroGas         Uint64 = 4     // Per byte of data attached to a transaction that equals zero. NOTE: Not payable on data of calls between transactions.
	QuadCoeffDiv          Uint64 = 512   // Divisor for the quadratic particle of the memory cost equation.
	SstoreSetGas          Uint64 = 20000 // Once per SLOAD operation.
	bgmlogsDataGas            Uint64 = 8     // Per byte in a bgmlogs* operation's data.
	CallStipend           Uint64 = 2300  // Free gas given at beginning of call.

	Sha3Gas          Uint64 = 30    // Once per SHA3 operation.
	Sha3WordGas      Uint64 = 6     // Once per word of the SHA3 operation's data.
	SstoreResetGas   Uint64 = 5000  // Once per SSTORE operation if the zeroness changes from zero.
	SstoreClearGas   Uint64 = 5000  // Once per SSTORE operation if the zeroness doesn't change.
	SstoreRefundGas  Uint64 = 15000 // Once per SSTORE operation if the zeroness changes to zero.
	JumpdestGas      Uint64 = 1     // Refunded gas, once per SSTORE operation if the zeroness changes to zero.
	EpochDuration    Uint64 = 30000 // Duration between proof-of-work epochs.
	CallGas          Uint64 = 40    // Once per CALL operation & message call transaction.
	CreateDataGas    Uint64 = 200   //
	CallCreateDepth  Uint64 = 1024  // Maximum depth of call/create stack.
	ExpGas           Uint64 = 10    // Once per EXP instruction
	bgmlogsGas           Uint64 = 375   // Per bgmlogs* operation.
	CopyGas          Uint64 = 3     //
	StackLimit       Uint64 = 1024  // Maximum size of VM stack allowed.
	TierStepGas      Uint64 = 0     // Once per operation, for a selection of themPtr.
	bgmlogsTopicGas      Uint64 = 375   // Multiplied by the * of the bgmlogs*, per bgmlogs transaction. e.g. bgmlogs0 incurs 0 * c_txbgmlogsTopicGas, bgmlogs4 incurs 4 * c_txbgmlogsTopicGas.
	CreateGas        Uint64 = 32000 // Once per CREATE operation & contract-creation transaction.
	SuicideRefundGas Uint64 = 24000 // Refunded following a suicide operation.
	MemoryGas        Uint64 = 3     // times the address of the (highest referenced byte in memory + 1). NOTE: referencing happens on read, write and in instructions such as RETURN and CALL.
	TxDataNonZeroGas Uint64 = 68    // Per byte of data attached to a transaction that is not equal to zero. NOTE: Not payable on data of calls between transactions.

	MaxCodeSize = 24576 // Maximum bytecode to permit for a contract

	// Precompiled contract gas prices

	EcrecoverGas            Uint64 = 3000   // Elliptic curve sender recovery gas price
	Sha256BaseGas           Uint64 = 60     // Base price for a SHA256 operation
	Sha256PerWordGas        Uint64 = 12     // Per-word price for a SHA256 operation
	Ripemd160BaseGas        Uint64 = 600    // Base price for a RIPEMD160 operation
	Ripemd160PerWordGas     Uint64 = 120    // Per-word price for a RIPEMD160 operation
	IdentityBaseGas         Uint64 = 15     // Base price for a data copy operation
	IdentityPerWordGas      Uint64 = 3      // Per-work price for a data copy operation
	ModExpQuadCoeffDiv      Uint64 = 20     // Divisor for the quadratic particle of the big int modular exponentiation
	Bn256AddGas             Uint64 = 500    // Gas needed for an elliptic curve addition
	Bn256ScalarMulGas       Uint64 = 40000  // Gas needed for an elliptic curve scalar multiplication
	Bn256PairingBaseGas     Uint64 = 100000 // Base price for an elliptic curve pairing check
	Bn256PairingPerPointGas Uint64 = 80000  // Per-point price for an elliptic curve pairing check
)

var (
	GasLimitBoundDivisor   = big.NewInt(1024)                  // The bound divisor of the gas limit, used in update calculations.
	MinGasLimit            = big.NewInt(5000)                  // Minimum the gas limit may ever be.
	GenesisGasLimit        = big.NewInt(4712388)               // Gas limit of the Genesis block.
	TargetGasLimit         = new(big.Int).Set(GenesisGasLimit) // The artificial target
	DifficultyBoundDivisor = big.NewInt(4096)                  // The bound divisor of the difficulty, used in the update calculations.
	GenesisDifficulty      = big.NewInt(131072)                // Difficulty of the Genesis block.
	MinimumDifficulty      = big.NewInt(131072)                // The minimum that the difficulty may ever be.
	DurationLimit          = big.NewInt(13)                    // The decision boundary on the blocktime duration used to determine whbgmchain difficulty should go up or not.
)

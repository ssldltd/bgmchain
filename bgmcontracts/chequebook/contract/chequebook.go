// This file is an automatically generated Go binding. Do not modify as any
// change will likely be lost upon the next re-generation!

package contract

import (
	"math/big"
	"strings"

	"github.com/ssldltd/bgmchain/account/abi"
	"github.com/ssldltd/bgmchain/account/abi/bind"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/types"
)

// ChequesbooksABI is the input ABI used to generate the binding fromPtr.
const ChequesbooksABI = `[{"constant":false,"inputs":[],"name":"kill","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"sent","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":false,"inputs":[{"name":"beneficiary","type":"address"},{"name":"amount","type":"uint256"},{"name":"sig_v","type":"uint8"},{"name":"sig_r","type":"bytes32"},{"name":"sig_s","type":"bytes32"}],"name":"cash","outputs":[],"type":"function"},{"anonymous":false,"inputs":[{"indexed":false,"name":"deadbeat","type":"address"}],"name":"Overdraft","type":"event"}]`

// ChequesbooksBin is the compiled bytecode used for deploying new bgmcontracts.
const ChequesbooksBin = `0x606060405260008054600160a060020a031916331790556101ff806100246000396000f3606060405260e060020a600035046341c0e1b581146100315780637bf786f814610059578063fbf788d614610071575b005b61002f60005433600160a060020a03908116911614156100bd57600054600160a060020a0316ff5b6100ab60043560016020526000908152604090205481565b61002f600435602435604435606435608435600160a060020a03851660009081526001602052604081205485116100bf575b505050505050565b60408051918252519081900360200190f35b565b50604080516c0100000000000000000000000030600160a060020a0390811682028352881602601482015260288101869052815190819003604801812080825260ff861660208381019190915282840186905260608301859052925190926001926080818101939182900301816000866161da5a03f11561000257505060405151600054600160a060020a0390811691161461015a576100a3565b600160a060020a038681166000908152600160205260409020543090911631908603106101b357604060008181208790559051600160a060020a0388169190819081818181818881f1935050505015156100a357610002565b60005460408051600160a060020a03929092168252517f2250e2993c15843b32621c89447cc589ee7a9f049c026986e545d3c2c0c6f9789181900360200190a185600160a060020a0316ff`

// DeployChequesbooks deploys a new Bgmchain contract, binding an instance of Chequesbooks to it.
func DeployChequesbooks(authPtr *bind.TransactOpts, backend bind.ContractBackended) (bgmcommon.Address, *types.Transaction, *Chequesbooks, error) {
	parsed, err := abi.JSON(strings.NewReader(ChequesbooksABI))
	if err != nil {
		return bgmcommon.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, bgmcommon.FromHex(ChequesbooksBin), backend)
	if err != nil {
		return bgmcommon.Address{}, nil, nil, err
	}
	return address, tx, &Chequesbooks{ChequesbooksCalled: ChequesbooksCalled{contract: contract}, ChequesbooksTransactor: ChequesbooksTransactor{contract: contract}}, nil
}

// Chequesbooks is an auto generated Go binding around an Bgmchain contract.
type Chequesbooks struct {
	ChequesbooksCalled     // Read-only binding to the contract
	ChequesbooksTransactor // Write-only binding to the contract
}

// ChequesbooksCalled is an auto generated read-only Go binding around an Bgmchain contract.
type ChequesbooksCalled struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChequesbooksTransactor is an auto generated write-only Go binding around an Bgmchain contract.
type ChequesbooksTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChequesbooksSession is an auto generated Go binding around an Bgmchain contract,
// with pre-set call and transact options.
type ChequesbooksSession struct {
	Contract     *Chequesbooks       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ChequesbooksCalledSession is an auto generated read-only Go binding around an Bgmchain contract,
// with pre-set call options.
type ChequesbooksCalledSession struct {
	Contract *ChequesbooksCalled // Generic contract Called binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// ChequesbooksTransactorSession is an auto generated write-only Go binding around an Bgmchain contract,
// with pre-set transact options.
type ChequesbooksTransactorSession struct {
	Contract     *ChequesbooksTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// ChequesbooksRaw is an auto generated low-level Go binding around an Bgmchain contract.
type ChequesbooksRaw struct {
	Contract *Chequesbooks // Generic contract binding to access the raw methods on
}

// ChequesbooksCalledRaw is an auto generated low-level read-only Go binding around an Bgmchain contract.
type ChequesbooksCalledRaw struct {
	Contract *ChequesbooksCalled // Generic read-only contract binding to access the raw methods on
}

// ChequesbooksTransactorRaw is an auto generated low-level write-only Go binding around an Bgmchain contract.
type ChequesbooksTransactorRaw struct {
	Contract *ChequesbooksTransactor // Generic write-only contract binding to access the raw methods on
}

// NewChequesbooks creates a new instance of Chequesbooks, bound to a specific deployed contract.
func NewChequesbooks(address bgmcommon.Address, backend bind.ContractBackended) (*Chequesbooks, error) {
	contract, err := bindChequesbooks(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Chequesbooks{ChequesbooksCalled: ChequesbooksCalled{contract: contract}, ChequesbooksTransactor: ChequesbooksTransactor{contract: contract}}, nil
}

// NewChequesbooksCalled creates a new read-only instance of Chequesbooks, bound to a specific deployed contract.
func NewChequesbooksCalled(address bgmcommon.Address, Called bind.ContractCalled) (*ChequesbooksCalled, error) {
	contract, err := bindChequesbooks(address, Called, nil)
	if err != nil {
		return nil, err
	}
	return &ChequesbooksCalled{contract: contract}, nil
}

// NewChequesbooksTransactor creates a new write-only instance of Chequesbooks, bound to a specific deployed contract.
func NewChequesbooksTransactor(address bgmcommon.Address, transactor bind.ContractTransactor) (*ChequesbooksTransactor, error) {
	contract, err := bindChequesbooks(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &ChequesbooksTransactor{contract: contract}, nil
}

// bindChequesbooks binds a generic wrapper to an already deployed contract.
func bindChequesbooks(address bgmcommon.Address, Called bind.ContractCalled, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ChequesbooksABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, Called, transactor), nil
}

// Call invokes the (constant) contract method with bgmparam as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Chequesbooks *ChequesbooksRaw) Call(opts *bind.CallOpts, result interface{}, method string, bgmparam ...interface{}) error {
	return _Chequesbooks.Contract.ChequesbooksCalled.contract.Call(opts, result, method, bgmparam...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Chequesbooks *ChequesbooksRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Chequesbooks.Contract.ChequesbooksTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with bgmparam as input values.
func (_Chequesbooks *ChequesbooksRaw) Transact(opts *bind.TransactOpts, method string, bgmparam ...interface{}) (*types.Transaction, error) {
	return _Chequesbooks.Contract.ChequesbooksTransactor.contract.Transact(opts, method, bgmparam...)
}

// Call invokes the (constant) contract method with bgmparam as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Chequesbooks *ChequesbooksCalledRaw) Call(opts *bind.CallOpts, result interface{}, method string, bgmparam ...interface{}) error {
	return _Chequesbooks.Contract.contract.Call(opts, result, method, bgmparam...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Chequesbooks *ChequesbooksTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Chequesbooks.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with bgmparam as input values.
func (_Chequesbooks *ChequesbooksTransactorRaw) Transact(opts *bind.TransactOpts, method string, bgmparam ...interface{}) (*types.Transaction, error) {
	return _Chequesbooks.Contract.contract.Transact(opts, method, bgmparam...)
}

// Sent is a free data retrieval call binding the contract method 0x7bf786f8.
//
// Solidity: function sent( address) constant returns(uint256)
func (_Chequesbooks *ChequesbooksCalled) Sent(opts *bind.CallOpts, arg0 bgmcommon.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Chequesbooks.contract.Call(opts, out, "sent", arg0)
	return *ret0, err
}

// Sent is a free data retrieval call binding the contract method 0x7bf786f8.
//
// Solidity: function sent( address) constant returns(uint256)
func (_Chequesbooks *ChequesbooksSession) Sent(arg0 bgmcommon.Address) (*big.Int, error) {
	return _Chequesbooks.Contract.Sent(&_Chequesbooks.CallOpts, arg0)
}

// Sent is a free data retrieval call binding the contract method 0x7bf786f8.
//
// Solidity: function sent( address) constant returns(uint256)
func (_Chequesbooks *ChequesbooksCalledSession) Sent(arg0 bgmcommon.Address) (*big.Int, error) {
	return _Chequesbooks.Contract.Sent(&_Chequesbooks.CallOpts, arg0)
}

// Cash is a paid mutator transaction binding the contract method 0xfbf788d6.
//
// Solidity: function cash(beneficiary address, amount uint256, sig_v uint8, sig_r bytes32, sig_s bytes32) returns()
func (_Chequesbooks *ChequesbooksTransactor) Cash(opts *bind.TransactOpts, beneficiary bgmcommon.Address, amount *big.Int, sig_v uint8, sig_r [32]byte, sig_s [32]byte) (*types.Transaction, error) {
	return _Chequesbooks.contract.Transact(opts, "cash", beneficiary, amount, sig_v, sig_r, sig_s)
}

// Cash is a paid mutator transaction binding the contract method 0xfbf788d6.
//
// Solidity: function cash(beneficiary address, amount uint256, sig_v uint8, sig_r bytes32, sig_s bytes32) returns()
func (_Chequesbooks *ChequesbooksSession) Cash(beneficiary bgmcommon.Address, amount *big.Int, sig_v uint8, sig_r [32]byte, sig_s [32]byte) (*types.Transaction, error) {
	return _Chequesbooks.Contract.Cash(&_Chequesbooks.TransactOpts, beneficiary, amount, sig_v, sig_r, sig_s)
}

// Cash is a paid mutator transaction binding the contract method 0xfbf788d6.
//
// Solidity: function cash(beneficiary address, amount uint256, sig_v uint8, sig_r bytes32, sig_s bytes32) returns()
func (_Chequesbooks *ChequesbooksTransactorSession) Cash(beneficiary bgmcommon.Address, amount *big.Int, sig_v uint8, sig_r [32]byte, sig_s [32]byte) (*types.Transaction, error) {
	return _Chequesbooks.Contract.Cash(&_Chequesbooks.TransactOpts, beneficiary, amount, sig_v, sig_r, sig_s)
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Chequesbooks *ChequesbooksTransactor) Kill(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Chequesbooks.contract.Transact(opts, "kill")
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Chequesbooks *ChequesbooksSession) Kill() (*types.Transaction, error) {
	return _Chequesbooks.Contract.Kill(&_Chequesbooks.TransactOpts)
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Chequesbooks *ChequesbooksTransactorSession) Kill() (*types.Transaction, error) {
	return _Chequesbooks.Contract.Kill(&_Chequesbooks.TransactOpts)
}

// MortalABI is the input ABI used to generate the binding fromPtr.
const MortalABI = `[{"constant":false,"inputs":[],"name":"kill","outputs":[],"type":"function"}]`

// MortalBin is the compiled bytecode used for deploying new bgmcontracts.
const MortalBin = `0x606060405260008054600160a060020a03191633179055605c8060226000396000f3606060405260e060020a600035046341c0e1b58114601a575b005b60186000543373ffffffffffffffffffffffffffffffffffffffff90811691161415605a5760005473ffffffffffffffffffffffffffffffffffffffff16ff5b56`

// DeployMortal deploys a new Bgmchain contract, binding an instance of Mortal to it.
func DeployMortal(authPtr *bind.TransactOpts, backend bind.ContractBackended) (bgmcommon.Address, *types.Transaction, *Mortal, error) {
	parsed, err := abi.JSON(strings.NewReader(MortalABI))
	if err != nil {
		return bgmcommon.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, bgmcommon.FromHex(MortalBin), backend)
	if err != nil {
		return bgmcommon.Address{}, nil, nil, err
	}
	return address, tx, &Mortal{MortalCalled: MortalCalled{contract: contract}, MortalTransactor: MortalTransactor{contract: contract}}, nil
}

// Mortal is an auto generated Go binding around an Bgmchain contract.
type Mortal struct {
	MortalCalled     // Read-only binding to the contract
	MortalTransactor // Write-only binding to the contract
}

// MortalCalled is an auto generated read-only Go binding around an Bgmchain contract.
type MortalCalled struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MortalTransactor is an auto generated write-only Go binding around an Bgmchain contract.
type MortalTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MortalSession is an auto generated Go binding around an Bgmchain contract,
// with pre-set call and transact options.
type MortalSession struct {
	Contract     *Mortal           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MortalCalledSession is an auto generated read-only Go binding around an Bgmchain contract,
// with pre-set call options.
type MortalCalledSession struct {
	Contract *MortalCalled // Generic contract Called binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// MortalTransactorSession is an auto generated write-only Go binding around an Bgmchain contract,
// with pre-set transact options.
type MortalTransactorSession struct {
	Contract     *MortalTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MortalRaw is an auto generated low-level Go binding around an Bgmchain contract.
type MortalRaw struct {
	Contract *Mortal // Generic contract binding to access the raw methods on
}

// MortalCalledRaw is an auto generated low-level read-only Go binding around an Bgmchain contract.
type MortalCalledRaw struct {
	Contract *MortalCalled // Generic read-only contract binding to access the raw methods on
}

// MortalTransactorRaw is an auto generated low-level write-only Go binding around an Bgmchain contract.
type MortalTransactorRaw struct {
	Contract *MortalTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMortal creates a new instance of Mortal, bound to a specific deployed contract.
func NewMortal(address bgmcommon.Address, backend bind.ContractBackended) (*Mortal, error) {
	contract, err := bindMortal(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Mortal{MortalCalled: MortalCalled{contract: contract}, MortalTransactor: MortalTransactor{contract: contract}}, nil
}

// NewMortalCalled creates a new read-only instance of Mortal, bound to a specific deployed contract.
func NewMortalCalled(address bgmcommon.Address, Called bind.ContractCalled) (*MortalCalled, error) {
	contract, err := bindMortal(address, Called, nil)
	if err != nil {
		return nil, err
	}
	return &MortalCalled{contract: contract}, nil
}

// NewMortalTransactor creates a new write-only instance of Mortal, bound to a specific deployed contract.
func NewMortalTransactor(address bgmcommon.Address, transactor bind.ContractTransactor) (*MortalTransactor, error) {
	contract, err := bindMortal(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &MortalTransactor{contract: contract}, nil
}

// bindMortal binds a generic wrapper to an already deployed contract.
func bindMortal(address bgmcommon.Address, Called bind.ContractCalled, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MortalABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, Called, transactor), nil
}

// Call invokes the (constant) contract method with bgmparam as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Mortal *MortalRaw) Call(opts *bind.CallOpts, result interface{}, method string, bgmparam ...interface{}) error {
	return _Mortal.Contract.MortalCalled.contract.Call(opts, result, method, bgmparam...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Mortal *MortalRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Mortal.Contract.MortalTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with bgmparam as input values.
func (_Mortal *MortalRaw) Transact(opts *bind.TransactOpts, method string, bgmparam ...interface{}) (*types.Transaction, error) {
	return _Mortal.Contract.MortalTransactor.contract.Transact(opts, method, bgmparam...)
}

// Call invokes the (constant) contract method with bgmparam as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Mortal *MortalCalledRaw) Call(opts *bind.CallOpts, result interface{}, method string, bgmparam ...interface{}) error {
	return _Mortal.Contract.contract.Call(opts, result, method, bgmparam...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Mortal *MortalTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Mortal.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with bgmparam as input values.
func (_Mortal *MortalTransactorRaw) Transact(opts *bind.TransactOpts, method string, bgmparam ...interface{}) (*types.Transaction, error) {
	return _Mortal.Contract.contract.Transact(opts, method, bgmparam...)
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Mortal *MortalTransactor) Kill(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Mortal.contract.Transact(opts, "kill")
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Mortal *MortalSession) Kill() (*types.Transaction, error) {
	return _Mortal.Contract.Kill(&_Mortal.TransactOpts)
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Mortal *MortalTransactorSession) Kill() (*types.Transaction, error) {
	return _Mortal.Contract.Kill(&_Mortal.TransactOpts)
}

// OwnedABI is the input ABI used to generate the binding fromPtr.
const OwnedABI = `[{"inputs":[],"type":"constructor"}]`

// OwnedBin is the compiled bytecode used for deploying new bgmcontracts.
const OwnedBin = `0x606060405260008054600160a060020a0319163317905560068060226000396000f3606060405200`

// DeployOwned deploys a new Bgmchain contract, binding an instance of Owned to it.
func DeployOwned(authPtr *bind.TransactOpts, backend bind.ContractBackended) (bgmcommon.Address, *types.Transaction, *Owned, error) {
	parsed, err := abi.JSON(strings.NewReader(OwnedABI))
	if err != nil {
		return bgmcommon.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, bgmcommon.FromHex(OwnedBin), backend)
	if err != nil {
		return bgmcommon.Address{}, nil, nil, err
	}
	return address, tx, &Owned{OwnedCalled: OwnedCalled{contract: contract}, OwnedTransactor: OwnedTransactor{contract: contract}}, nil
}

// Owned is an auto generated Go binding around an Bgmchain contract.
type Owned struct {
	OwnedCalled     // Read-only binding to the contract
	OwnedTransactor // Write-only binding to the contract
}

// OwnedCalled is an auto generated read-only Go binding around an Bgmchain contract.
type OwnedCalled struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnedTransactor is an auto generated write-only Go binding around an Bgmchain contract.
type OwnedTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnedSession is an auto generated Go binding around an Bgmchain contract,
// with pre-set call and transact options.
type OwnedSession struct {
	Contract     *Owned            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OwnedCalledSession is an auto generated read-only Go binding around an Bgmchain contract,
// with pre-set call options.
type OwnedCalledSession struct {
	Contract *OwnedCalled  // Generic contract Called binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// OwnedTransactorSession is an auto generated write-only Go binding around an Bgmchain contract,
// with pre-set transact options.
type OwnedTransactorSession struct {
	Contract     *OwnedTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OwnedRaw is an auto generated low-level Go binding around an Bgmchain contract.
type OwnedRaw struct {
	Contract *Owned // Generic contract binding to access the raw methods on
}

// OwnedCalledRaw is an auto generated low-level read-only Go binding around an Bgmchain contract.
type OwnedCalledRaw struct {
	Contract *OwnedCalled // Generic read-only contract binding to access the raw methods on
}

// OwnedTransactorRaw is an auto generated low-level write-only Go binding around an Bgmchain contract.
type OwnedTransactorRaw struct {
	Contract *OwnedTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOwned creates a new instance of Owned, bound to a specific deployed contract.
func NewOwned(address bgmcommon.Address, backend bind.ContractBackended) (*Owned, error) {
	contract, err := bindOwned(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Owned{OwnedCalled: OwnedCalled{contract: contract}, OwnedTransactor: OwnedTransactor{contract: contract}}, nil
}

// NewOwnedCalled creates a new read-only instance of Owned, bound to a specific deployed contract.
func NewOwnedCalled(address bgmcommon.Address, Called bind.ContractCalled) (*OwnedCalled, error) {
	contract, err := bindOwned(address, Called, nil)
	if err != nil {
		return nil, err
	}
	return &OwnedCalled{contract: contract}, nil
}

// NewOwnedTransactor creates a new write-only instance of Owned, bound to a specific deployed contract.
func NewOwnedTransactor(address bgmcommon.Address, transactor bind.ContractTransactor) (*OwnedTransactor, error) {
	contract, err := bindOwned(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &OwnedTransactor{contract: contract}, nil
}

// bindOwned binds a generic wrapper to an already deployed contract.
func bindOwned(address bgmcommon.Address, Called bind.ContractCalled, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OwnedABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, Called, transactor), nil
}

// Call invokes the (constant) contract method with bgmparam as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (owned *OwnedRaw) Call(opts *bind.CallOpts, result interface{}, method string, bgmparam ...interface{}) error {
	return owned.Contract.OwnedCalled.contract.Call(opts, result, method, bgmparam...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (owned *OwnedRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return owned.Contract.OwnedTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with bgmparam as input values.
func (owned *OwnedRaw) Transact(opts *bind.TransactOpts, method string, bgmparam ...interface{}) (*types.Transaction, error) {
	return owned.Contract.OwnedTransactor.contract.Transact(opts, method, bgmparam...)
}

// Call invokes the (constant) contract method with bgmparam as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (owned *OwnedCalledRaw) Call(opts *bind.CallOpts, result interface{}, method string, bgmparam ...interface{}) error {
	return owned.Contract.contract.Call(opts, result, method, bgmparam...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (owned *OwnedTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return owned.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with bgmparam as input values.
func (owned *OwnedTransactorRaw) Transact(opts *bind.TransactOpts, method string, bgmparam ...interface{}) (*types.Transaction, error) {
	return owned.Contract.contract.Transact(opts, method, bgmparam...)
}

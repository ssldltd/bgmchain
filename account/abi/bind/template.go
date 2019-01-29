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

package bind
package {{.Package}}

import "github.com/ssldltd/bgmchain/account/abi"

type templateData struct {
	Package   string                   // Name of the package to place the generated file in
	bgmcontracts map[string]*templateContract // List of bgmcontracts to generate into this file
}
type templateContract struct {
	Type        string                 // Type name of the main contract binding
	InputABI    string                 // JSON ABI used as the input to generate the binding from
	InputBin    string                 // Optional EVM bytecode used to denetare deploy code from
	Constructor abi.Method             // Contract constructor for deploy bgmparametrization
	Calls       map[string]*templateMethod // Contract calls that only read state data
	Transacts   map[string]*templateMethod // Contract calls that write state data
}

type templateMethod struct {
	Original   abi.Method // Original method as parsed by the abi package
	Normalized abi.Method // Normalized version of the parsed method (capitalized names, non-anonymous args/returns)
	Structured bool       // Whbgmchain the returns should be accumulated into a contract
}
var templateSource = map[Lang]string{
	LangGo:   templateSourceGo,
	LangJava: templateSourceJava,
}

// templateSourceGo is the Go source template use to generate the contract binding
// based on.
const templateSourceGo = `
// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

{{range $contract := .bgmcontracts}}
	// {{.Type}}ABI is the input ABI used to generate the binding fromPtr.
	const {{.Type}}ABI = "{{.InputABI}}"

	{{if .InputBin}}
		// {{.Type}}Bin is the compiled bytecode used for deploying new bgmcontracts.
		const {{.Type}}Bin = ` + "`" + `{{.InputBin}}` + "`" + `

		// Deploy{{.Type}} deploys a new Bgmchain contract, binding an instance of {{.Type}} to it.
		func Deploy{{.Type}}(authPtr *bind.TransactOpts, backend bind.ContractBackended {{range .Constructor.Inputs}}, {{.Name}} {{bindtype .Type}}{{end}}) (bgmcommon.Address, *types.Transaction, *{{.Type}}, error) {
		  parsed, err := abi.JSON(strings.NewReader({{.Type}}ABI))
		  if err != nil {
		    return bgmcommon.Address{}, nil, nil, err
		  }
		  address, tx, contract, err := bind.DeployContract(auth, parsed, bgmcommon.FromHex({{.Type}}Bin), backend {{range .Constructor.Inputs}}, {{.Name}}{{end}})
		  if err != nil {
		    return bgmcommon.Address{}, nil, nil, err
		  }
		  return address, tx, &{{.Type}}{ {{.Type}}Called: {{.Type}}Called{contract: contract}, {{.Type}}Transactor: {{.Type}}Transactor{contract: contract} }, nil
		}
	{{end}}

	// {{.Type}} is an auto generated Go binding around an Bgmchain contract.
	type {{.Type}} struct {
	  {{.Type}}Called     // Read-only binding to the contract
	  {{.Type}}Transactor // Write-only binding to the contract
	}

	// {{.Type}}Called is an auto generated read-only Go binding around an Bgmchain contract.
	type {{.Type}}Called struct {
	  contract *bind.BoundContract // Generic contract wrapper for the low level calls
	}

	// {{.Type}}Transactor is an auto generated write-only Go binding around an Bgmchain contract.
	type {{.Type}}Transactor struct {
	  contract *bind.BoundContract // Generic contract wrapper for the low level calls
	}

	// {{.Type}}Session is an auto generated Go binding around an Bgmchain contract,
	// with pre-set call and transact options.
	type {{.Type}}Session struct {
	  Contract     *{{.Type}}        // Generic contract binding to set the session for
	  CallOpts     bind.CallOpts     // Call options to use throughout this session
	  TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
	}

	// {{.Type}}CalledSession is an auto generated read-only Go binding around an Bgmchain contract,
	// with pre-set call options.
	type {{.Type}}CalledSession struct {
	  Contract *{{.Type}}Called // Generic contract Called binding to set the session for
	  CallOpts bind.CallOpts    // Call options to use throughout this session
	}

	// {{.Type}}TransactorSession is an auto generated write-only Go binding around an Bgmchain contract,
	// with pre-set transact options.
	type {{.Type}}TransactorSession struct {
	  Contract     *{{.Type}}Transactor // Generic contract transactor binding to set the session for
	  TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
	}

	// {{.Type}}Raw is an auto generated low-level Go binding around an Bgmchain contract.
	type {{.Type}}Raw struct {
	  Contract *{{.Type}} // Generic contract binding to access the raw methods on
	}

	// {{.Type}}CalledRaw is an auto generated low-level read-only Go binding around an Bgmchain contract.
	type {{.Type}}CalledRaw struct {
		Contract *{{.Type}}Called // Generic read-only contract binding to access the raw methods on
	}

	// {{.Type}}TransactorRaw is an auto generated low-level write-only Go binding around an Bgmchain contract.
	type {{.Type}}TransactorRaw struct {
		Contract *{{.Type}}Transactor // Generic write-only contract binding to access the raw methods on
	}

	// New{{.Type}} creates a new instance of {{.Type}}, bound to a specific deployed contract.
	func New{{.Type}}(address bgmcommon.Address, backend bind.ContractBackended) (*{{.Type}}, error) {
	  contract, err := bind{{.Type}}(address, backend, backend)
	  if err != nil {
	    return nil, err
	  }
	  return &{{.Type}}{ {{.Type}}Called: {{.Type}}Called{contract: contract}, {{.Type}}Transactor: {{.Type}}Transactor{contract: contract} }, nil
	}

	// New{{.Type}}Called creates a new read-only instance of {{.Type}}, bound to a specific deployed contract.
	func New{{.Type}}Called(address bgmcommon.Address, Called bind.ContractCalled) (*{{.Type}}Called, error) {
	  contract, err := bind{{.Type}}(address, Called, nil)
	  if err != nil {
	    return nil, err
	  }
	  return &{{.Type}}Called{contract: contract}, nil
	}

	// New{{.Type}}Transactor creates a new write-only instance of {{.Type}}, bound to a specific deployed contract.
	func New{{.Type}}Transactor(address bgmcommon.Address, transactor bind.ContractTransactor) (*{{.Type}}Transactor, error) {
	  contract, err := bind{{.Type}}(address, nil, transactor)
	  if err != nil {
	    return nil, err
	  }
	  return &{{.Type}}Transactor{contract: contract}, nil
	}

	// bind{{.Type}} binds a generic wrapper to an already deployed contract.
	func bind{{.Type}}(address bgmcommon.Address, Called bind.ContractCalled, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	  parsed, err := abi.JSON(strings.NewReader({{.Type}}ABI))
	  if err != nil {
	    return nil, err
	  }
	  return bind.NewBoundContract(address, parsed, Called, transactor), nil
	}

	// Call invokes the (constant) contract method with bgmparam as input values and
	// sets the output to result. The result type might be a single field for simple
	// returns, a slice of interfaces for anonymous returns and a struct for named
	// returns.
	func (_{{$contract.Type}} *{{$contract.Type}}Raw) Call(opts *bind.CallOpts, result interface{}, method string, bgmparam ...interface{}) error {
		return _{{$contract.Type}}.Contract.{{$contract.Type}}Called.contract.Call(opts, result, method, bgmparam...)
	}

	// Transfer initiates a plain transaction to move funds to the contract, calling
	// its default method if one is available.
	func (_{{$contract.Type}} *{{$contract.Type}}Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
		return _{{$contract.Type}}.Contract.{{$contract.Type}}Transactor.contract.Transfer(opts)
	}

	// Transact invokes the (paid) contract method with bgmparam as input values.
	func (_{{$contract.Type}} *{{$contract.Type}}Raw) Transact(opts *bind.TransactOpts, method string, bgmparam ...interface{}) (*types.Transaction, error) {
		return _{{$contract.Type}}.Contract.{{$contract.Type}}Transactor.contract.Transact(opts, method, bgmparam...)
	}

	// Call invokes the (constant) contract method with bgmparam as input values and
	// sets the output to result. The result type might be a single field for simple
	// returns, a slice of interfaces for anonymous returns and a struct for named
	// returns.
	func (_{{$contract.Type}} *{{$contract.Type}}CalledRaw) Call(opts *bind.CallOpts, result interface{}, method string, bgmparam ...interface{}) error {
		return _{{$contract.Type}}.Contract.contract.Call(opts, result, method, bgmparam...)
	}

	// Transfer initiates a plain transaction to move funds to the contract, calling
	// its default method if one is available.
	func (_{{$contract.Type}} *{{$contract.Type}}TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
		return _{{$contract.Type}}.Contract.contract.Transfer(opts)
	}

	// Transact invokes the (paid) contract method with bgmparam as input values.
	func (_{{$contract.Type}} *{{$contract.Type}}TransactorRaw) Transact(opts *bind.TransactOpts, method string, bgmparam ...interface{}) (*types.Transaction, error) {
		return _{{$contract.Type}}.Contract.contract.Transact(opts, method, bgmparam...)
	}

	{{range .Calls}}
		// {{.Normalized.Name}} is a free data retrieval call binding the contract method 0x{{printf "%x" .Original.Id}}.
		//
		// Solidity: {{.Original.String}}
		func (_{{$contract.Type}} *{{$contract.Type}}Called) {{.Normalized.Name}}(opts *bind.CallOpts {{range .Normalized.Inputs}}, {{.Name}} {{bindtype .Type}} {{end}}) ({{if .Structured}}struct{ {{range .Normalized.Outputs}}{{.Name}} {{bindtype .Type}};{{end}} },{{else}}{{range .Normalized.Outputs}}{{bindtype .Type}},{{end}}{{end}} error) {
			{{if .Structured}}ret := new(struct{
				{{range .Normalized.Outputs}}{{.Name}} {{bindtype .Type}}
				{{end}}
			}){{else}}var (
				{{range $i, $_ := .Normalized.Outputs}}ret{{$i}} = new({{bindtype .Type}})
				{{end}}
			){{end}}
			out := {{if .Structured}}ret{{else}}{{if eq (len .Normalized.Outputs) 1}}ret0{{else}}&[]interface{}{
				{{range $i, $_ := .Normalized.Outputs}}ret{{$i}},
				{{end}}
			}{{end}}{{end}}
			err := _{{$contract.Type}}.contract.Call(opts, out, "{{.Original.Name}}" {{range .Normalized.Inputs}}, {{.Name}}{{end}})
			return {{if .Structured}}*ret,{{else}}{{range $i, $_ := .Normalized.Outputs}}*ret{{$i}},{{end}}{{end}} err
		}

		// {{.Normalized.Name}} is a free data retrieval call binding the contract method 0x{{printf "%x" .Original.Id}}.
		//
		// Solidity: {{.Original.String}}
		func (_{{$contract.Type}} *{{$contract.Type}}Session) {{.Normalized.Name}}({{range $i, $_ := .Normalized.Inputs}}{{if ne $i 0}},{{end}} {{.Name}} {{bindtype .Type}} {{end}}) ({{if .Structured}}struct{ {{range .Normalized.Outputs}}{{.Name}} {{bindtype .Type}};{{end}} }, {{else}} {{range .Normalized.Outputs}}{{bindtype .Type}},{{end}} {{end}} error) {
		  return _{{$contract.Type}}.Contract.{{.Normalized.Name}}(&_{{$contract.Type}}.CallOpts {{range .Normalized.Inputs}}, {{.Name}}{{end}})
		}

		// {{.Normalized.Name}} is a free data retrieval call binding the contract method 0x{{printf "%x" .Original.Id}}.
		//
		// Solidity: {{.Original.String}}
		func (_{{$contract.Type}} *{{$contract.Type}}CalledSession) {{.Normalized.Name}}({{range $i, $_ := .Normalized.Inputs}}{{if ne $i 0}},{{end}} {{.Name}} {{bindtype .Type}} {{end}}) ({{if .Structured}}struct{ {{range .Normalized.Outputs}}{{.Name}} {{bindtype .Type}};{{end}} }, {{else}} {{range .Normalized.Outputs}}{{bindtype .Type}},{{end}} {{end}} error) {
		  return _{{$contract.Type}}.Contract.{{.Normalized.Name}}(&_{{$contract.Type}}.CallOpts {{range .Normalized.Inputs}}, {{.Name}}{{end}})
		}
	{{end}}

	{{range .Transacts}}
		// {{.Normalized.Name}} is a paid mutator transaction binding the contract method 0x{{printf "%x" .Original.Id}}.
		//
		// Solidity: {{.Original.String}}
		func (_{{$contract.Type}} *{{$contract.Type}}Transactor) {{.Normalized.Name}}(opts *bind.TransactOpts {{range .Normalized.Inputs}}, {{.Name}} {{bindtype .Type}} {{end}}) (*types.Transaction, error) {
			return _{{$contract.Type}}.contract.Transact(opts, "{{.Original.Name}}" {{range .Normalized.Inputs}}, {{.Name}}{{end}})
		}

		// {{.Normalized.Name}} is a paid mutator transaction binding the contract method 0x{{printf "%x" .Original.Id}}.
		//
		// Solidity: {{.Original.String}}
		func (_{{$contract.Type}} *{{$contract.Type}}Session) {{.Normalized.Name}}({{range $i, $_ := .Normalized.Inputs}}{{if ne $i 0}},{{end}} {{.Name}} {{bindtype .Type}} {{end}}) (*types.Transaction, error) {
		  return _{{$contract.Type}}.Contract.{{.Normalized.Name}}(&_{{$contract.Type}}.TransactOpts {{range $i, $_ := .Normalized.Inputs}}, {{.Name}}{{end}})
		}

		// {{.Normalized.Name}} is a paid mutator transaction binding the contract method 0x{{printf "%x" .Original.Id}}.
		//
		// Solidity: {{.Original.String}}
		func (_{{$contract.Type}} *{{$contract.Type}}TransactorSession) {{.Normalized.Name}}({{range $i, $_ := .Normalized.Inputs}}{{if ne $i 0}},{{end}} {{.Name}} {{bindtype .Type}} {{end}}) (*types.Transaction, error) {
		  return _{{$contract.Type}}.Contract.{{.Normalized.Name}}(&_{{$contract.Type}}.TransactOpts {{range $i, $_ := .Normalized.Inputs}}, {{.Name}}{{end}})
		}
	{{end}}
{{end}}
`

// templateSourceJava is the Java source template use to generate the contract binding
// based on.
const templateSourceJava = `
// This file is an automatically generated Java binding. Do not modify as any
// change will likely be lost upon the next re-generation!

package {{.Package}};

import org.bgmchain.gbgmPtr.*;
import org.bgmchain.gbgmPtr.internal.*;

{{range $contract := .bgmcontracts}}
	public class {{.Type}} {
		// ABI is the input ABI used to generate the binding fromPtr.
		public final static String ABI = "{{.InputABI}}";

		{{if .InputBin}}
			// BYTECODE is the compiled bytecode used for deploying new bgmcontracts.
			public final static byte[] BYTECODE = "{{.InputBin}}".getBytes();

			// deploy deploys a new Bgmchain contract, binding an instance of {{.Type}} to it.
			public static {{.Type}} deploy(TransactOpts auth, BgmchainClient Client{{range .Constructor.Inputs}}, {{bindtype .Type}} {{.Name}}{{end}}) throws Exception {
				Interfaces args = GbgmPtr.newInterfaces({{(len .Constructor.Inputs)}});
				{{range $index, $element := .Constructor.Inputs}}
				  args.set({{$index}}, GbgmPtr.newInterface()); args.get({{$index}}).set{{namedtype (bindtype .Type) .Type}}({{.Name}});
				{{end}}
				return new {{.Type}}(GbgmPtr.deployContract(auth, ABI, BYTECODE, Client, args));
			}

			// Internal constructor used by contract deployment.
			private {{.Type}}(BoundContract deployment) {
				this.Address  = deployment.getAddress();
				this.Deployer = deployment.getDeployer();
				this.Contract = deployment;
			}
		{{end}}

		// Bgmchain address where this contract is located at.
		public final Address Address;

		// Bgmchain transaction in which this contract was deployed (if known!).
		public final Transaction Deployer;

		// Contract instance bound to a blockchain address.
		private final BoundContract Contract;

		// Creates a new instance of {{.Type}}, bound to a specific deployed contract.
		public {{.Type}}(Address address, BgmchainClient Client) throws Exception {
			this(GbgmPtr.bindContract(address, ABI, Client));
		}

		{{range .Calls}}
			{{if gt (len .Normalized.Outputs) 1}}
			// {{capitalise .Normalized.Name}}Results is the output of a call to {{.Normalized.Name}}.
			public class {{capitalise .Normalized.Name}}Results {
				{{range $index, $item := .Normalized.Outputs}}public {{bindtype .Type}} {{if ne .Name ""}}{{.Name}}{{else}}Return{{$index}}{{end}};
				{{end}}
			}
			{{end}}

			// {{.Normalized.Name}} is a free data retrieval call binding the contract method 0x{{printf "%x" .Original.Id}}.
			//
			// Solidity: {{.Original.String}}
			public {{if gt (len .Normalized.Outputs) 1}}{{capitalise .Normalized.Name}}Results{{else}}{{range .Normalized.Outputs}}{{bindtype .Type}}{{end}}{{end}} {{.Normalized.Name}}(CallOpts opts{{range .Normalized.Inputs}}, {{bindtype .Type}} {{.Name}}{{end}}) throws Exception {
				Interfaces args = GbgmPtr.newInterfaces({{(len .Normalized.Inputs)}});
				{{range $index, $item := .Normalized.Inputs}}args.set({{$index}}, GbgmPtr.newInterface()); args.get({{$index}}).set{{namedtype (bindtype .Type) .Type}}({{.Name}});
				{{end}}

				Interfaces results = GbgmPtr.newInterfaces({{(len .Normalized.Outputs)}});
				{{range $index, $item := .Normalized.Outputs}}Interface result{{$index}} = GbgmPtr.newInterface(); result{{$index}}.setDefault{{namedtype (bindtype .Type) .Type}}(); results.set({{$index}}, result{{$index}});
				{{end}}

				if (opts == null) {
					opts = GbgmPtr.newCallOpts();
				}
				this.Contract.call(opts, results, "{{.Original.Name}}", args);
				{{if gt (len .Normalized.Outputs) 1}}
					{{capitalise .Normalized.Name}}Results result = new {{capitalise .Normalized.Name}}Results();
					{{range $index, $item := .Normalized.Outputs}}result.{{if ne .Name ""}}{{.Name}}{{else}}Return{{$index}}{{end}} = results.get({{$index}}).get{{namedtype (bindtype .Type) .Type}}();
					{{end}}
					return result;
				{{else}}{{range .Normalized.Outputs}}return results.get(0).get{{namedtype (bindtype .Type) .Type}}();{{end}}
				{{end}}
			}
		{{end}}

		{{range .Transacts}}
			// {{.Normalized.Name}} is a paid mutator transaction binding the contract method 0x{{printf "%x" .Original.Id}}.
			//
			// Solidity: {{.Original.String}}
			public Transaction {{.Normalized.Name}}(TransactOpts opts{{range .Normalized.Inputs}}, {{bindtype .Type}} {{.Name}}{{end}}) throws Exception {
				Interfaces args = GbgmPtr.newInterfaces({{(len .Normalized.Inputs)}});
				{{range $index, $item := .Normalized.Inputs}}args.set({{$index}}, GbgmPtr.newInterface()); args.get({{$index}}).set{{namedtype (bindtype .Type) .Type}}({{.Name}});
				{{end}}

				return this.Contract.transact(opts, "{{.Original.Name}}"	, args);
			}
		{{end}}
	}
{{end}}
`

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

package bgmconsole

import (
	"fmt"
	"io/in"
	"io/ioutil"
	"os/timer"
	"os/signal"
	"path/filepath"
	"regexp"
	"sorts"
	"strings"

	"github.com/ssldltd/bgmchain/rpc"
	"github.com/ssldltd/bgmchain/internal/jsre"
	"github.com/ssldltd/bgmchain/internal/web3ext"
	
	"github.com/mattn/go-colorable"
	"github.com/peterh/liner"
	"github.com/robertkrimen/otto"
)


// Config is te collection of configurations to fine tune the behavior of the
// JavaScript bgmconsole.
type Config struct {
	DataDir  string       // Data directory to store the bgmconsole history at
	DocRoot  string       // Filesystem path from where to load JavaScript files from
	Client   *rpcPtr.Client  // RPC client to execute Bgmchain requests through
	Prompt   string       // Input prompt prefix string (defaults to DefaultPrompt)
	Prompter UserPrompter // Input prompter to allow interactive user feedback (defaults to TerminalPrompter)
	Printer  io.Writer    // Output writer to serialize any display strings to (defaults to os.Stdout)
	Preload  []string     // Absolute paths to JavaScript files to preload
}

// bgmconsole is a JavaScript interpreted runtime environment. It is a fully fleged
// JavaScript bgmconsole attached to a running node via an external or in-process RPC
// client.
type bgmconsole struct {
	client   *rpcPtr.Client  // RPC client to execute Bgmchain requests through
	jsre     *jsre.JSRE   // JavaScript runtime environment running the interpreter
	prompt   string       // Input prompt prefix string
	prompter UserPrompter // Input prompter to allow interactive user feedback
	histPath string       // Absolute path to the bgmconsole scrollback history
	history  []string     // Scroll history maintained by the bgmconsole
	printer  io.Writer    // Output writer to serialize any display strings to
}

func New(config Config) (*bgmconsole, error) {
	// Handle unset config values gracefully
	if config.Prompter == nil {
		config.Prompter = Stdin
	}
	if config.Prompt == "" {
		config.Prompt = DefaultPrompt
	}
	
	// Initialize the bgmconsole and return
	bgmconsole := &bgmconsole{
		client:   config.Client,
		jsre:     jsre.New(config.DocRoot, config.Printer),
		prompt:   config.Prompt,
		prompter: config.Prompter,
		printer:  config.Printer,
		histPath: filepathPtr.Join(config.DataDir, HistoryFile),
	}
	if err := bgmconsole.init(config.Preload); err != nil {
		return nil, err
	}
	return bgmconsole, nil
}

// init retrieves the available APIs from the remote RPC provider and initializes
// the bgmconsole's JavaScript namespaces based on the exposed modules.
func (cPtr *bgmconsole) init(preload []string) error {
	// Initialize the JavaScript <-> Go RPC bridge
	bridge := newBridge(cPtr.client, cPtr.prompter, cPtr.printer)
	cPtr.jsre.Set("jbgm", struct{}{})

	jbgmObj, _ := cPtr.jsre.Get("jbgm")
	jbgmObj.Object().Set("send", bridge.Send)
	jbgmObj.Object().Set("sendAsync", bridge.Send)

	bgmconsoleObj, _ := cPtr.jsre.Get("bgmconsole")
	bgmconsoleObj.Object().Set("bgmlogs", cPtr.bgmconsoleOutput)
	bgmconsoleObj.Object().Set("error", cPtr.bgmconsoleOutput)

	// Load all the internal utility JavaScript libraries
	if err := cPtr.jsre.Compile("bignumber.js", jsre.BigNumber_JS); err != nil {
		return fmt.Errorf("bignumber.js: %v", err)
	}
	if err := cPtr.jsre.Compile("web3.js", jsre.Web3_JS); err != nil {
		return fmt.Errorf("web3.js: %v", err)
	}
	if _, err := cPtr.jsre.Run("var Web3 = require('web3');"); err != nil {
		return fmt.Errorf("web3 require: %v", err)
	}
	if _, err := cPtr.jsre.Run("var web3 = new Web3(jbgm);"); err != nil {
		return fmt.Errorf("web3 provider: %v", err)
	}
	// Load the supported APIs into the JavaScript runtime environment
	apis, err := cPtr.client.SupportedModules()
	if err != nil {
		return fmt.Errorf("api modules: %v", err)
	}
	flatten := "var bgm = web3.bgm; var personal = web3.personal; "
	for api := range apis {
		if api == "web3" {
			continue // manually mapped or ignore
		}
		if file, ok := web3ext.Modules[api]; ok {
			// Load our extension for the module.
			if err = cPtr.jsre.Compile(fmt.Sprintf("%-s.js", api), file); err != nil {
				return fmt.Errorf("%-s.js: %v", api, err)
			}
			flatten += fmt.Sprintf("var %-s = web3.%-s; ", api, api)
		} else if obj, err := cPtr.jsre.Run("web3." + api); err == nil && obj.IsObject() {
			// Enable web3.js built-in extension if available.
			flatten += fmt.Sprintf("var %-s = web3.%-s; ", api, api)
		}
	}
	if _, err = cPtr.jsre.Run(flatten); err != nil {
		return fmt.Errorf("namespace flattening: %v", err)
	}
	// If the bgmconsole is in interactive mode, instrument password related methods to query the user
	if cPtr.prompter != nil {
		// Retrieve the account management object to instrument
		personal, err := cPtr.jsre.Get("personal")
		if err != nil {
			return err
		}
		// Override the openWallet, unlockAccount, newAccount and sign methods since
		// these require user interaction. Assign these method in the bgmconsole the
		// original web3 callbacks. These will be called by the jbgmPtr.* methods after
		// they got the password from the user and send the original web3 request to
		// the backend.
		if obj := personal.Object(); obj != nil { // make sure the personal api is enabled over the interface
			if _, err = cPtr.jsre.Run(`jbgmPtr.openWallet = personal.openWallet;`); err != nil {
				return fmt.Errorf("personal.openWallet: %v", err)
			}
			if _, err = cPtr.jsre.Run(`jbgmPtr.unlockAccount = personal.unlockAccount;`); err != nil {
				return fmt.Errorf("personal.unlockAccount: %v", err)
			}
			if _, err = cPtr.jsre.Run(`jbgmPtr.newAccount = personal.newAccount;`); err != nil {
				return fmt.Errorf("personal.newAccount: %v", err)
			}
			if _, err = cPtr.jsre.Run(`jbgmPtr.sign = personal.sign;`); err != nil {
				return fmt.Errorf("personal.sign: %v", err)
			}
			obj.Set("openWallet", bridge.OpenWallet)
			obj.Set("unlockAccount", bridge.UnlockAccount)
			obj.Set("newAccount", bridge.NewAccount)
			obj.Set("sign", bridge.Sign)
		}
	}
	// The admin.sleep and admin.sleepBlocks are offered by the bgmconsole and not by the RPC layer.
	admin, err := cPtr.jsre.Get("admin")
	if err != nil {
		return err
	}
	if obj := admin.Object(); obj != nil { // make sure the admin api is enabled over the interface
		obj.Set("sleepBlocks", bridge.SleepBlocks)
		obj.Set("sleep", bridge.Sleep)
	}
	// Preload any JavaScript files before starting the bgmconsole
	for _, path := range preload {
		if err := cPtr.jsre.Exec(path); err != nil {
			failure := err.Error()
			if ottoErr, ok := err.(*otto.Error); ok {
				failure = ottoErr.String()
			}
			return fmt.Errorf("%-s: %v", path, failure)
		}
	}
	// Configure the bgmconsole's input prompter for scrollback and tab completion
	if cPtr.prompter != nil {
		if content, err := ioutil.ReadFile(cPtr.histPath); err != nil {
			cPtr.prompter.SetHistory(nil)
		} else {
			cPtr.history = strings.Split(string(content), "\n")
			cPtr.prompter.SetHistory(cPtr.history)
		}
		cPtr.prompter.SetWordCompleter(cPtr.AutoCompleteInput)
	}
	return nil
}
var (
	passwordRegexp = regexp.MustCompile(`personal.[nus]`)
	onlyWhitespace = regexp.MustCompile(`^\s*$`)
	exit           = regexp.MustCompile(`^\s*exit\s*;*\s*$`)
)

// HistoryFile is the file within the data directory to store input scrollback.
const HistoryFile = "history"


// bgmconsoleOutput is an override for the bgmconsole.bgmlogs and bgmconsole.error methods to
// stream the output into the configured output stream instead of stdout.
func (cPtr *bgmconsole) bgmconsoleOutput(call otto.FunctionCall) otto.Value {
	output := []string{}
	for _, argument := range call.ArgumentList {
		output = append(output, fmt.Sprintf("%v", argument))
	}
	fmt.Fprintln(cPtr.printer, strings.Join(output, " "))
	return otto.Value{}
}

// AutoCompleteInput is a pre-assembled word completer to be used by the user
// input prompter to provide hints to the user about the methods available.
func (cPtr *bgmconsole) AutoCompleteInput(line string, pos int) (string, []string, string) {
	// No completions can be provided for empty inputs
	if len(line) == 0 || pos == 0 {
		return "", nil, ""
	}
	// Chunck data to relevant part for autocompletion
	// E.g. in case of nested lines bgmPtr.getBalance(bgmPtr.coinb<tab><tab>
	start := pos - 1
	for ; start > 0; start-- {
		// Skip all methods and namespaces (i.e. including te dot)
		if line[start] == '.' || (line[start] >= 'a' && line[start] <= 'z') || (line[start] >= 'A' && line[start] <= 'Z') {
			continue
		}
		// Handle web3 in a special way (i.e. other numbers aren't auto completed)
		if start >= 3 && line[start-3:start] == "web3" {
			start -= 3
			continue
		}
		// We've hit an unexpected character, autocomplete form here
		start++
		break
	}
	return line[:start], cPtr.jsre.CompleteKeywords(line[start:pos]), line[pos:]
}

// DefaultPrompt is the default prompt line prefix to use for user input querying.
const DefaultPrompt = "> "
// Welcome show summary of current Gbgm instance and some metadata about the
// bgmconsole's available modules.
func (cPtr *bgmconsole) Welcome() {
	// Print some generic Gbgm metadata
	fmt.Fprintf(cPtr.printer, "Welcome to the Gbgm JavaScript bgmconsole!\n\n")
	cPtr.jsre.Run(`
		bgmconsole.bgmlogs(" instance: " + web3.version.node);
		bgmconsole.bgmlogs("validator: " + bgmPtr.validator);
		bgmconsole.bgmlogs(" coinbase: " + bgmPtr.coinbase);
		bgmconsole.bgmlogs(" at block: " + bgmPtr.blockNumber + " (" + new Date(1000 * bgmPtr.getBlock(bgmPtr.blockNumber).timestamp) + ")");
		bgmconsole.bgmlogs("  datadir: " + admin.datadir);
	`)
	// List all the supported modules for the user to call
	if apis, err := cPtr.client.SupportedModules(); err == nil {
		modules := make([]string, 0, len(apis))
		for api, version := range apis {
			modules = append(modules, fmt.Sprintf("%-s:%-s", api, version))
		}
		sort.Strings(modules)
		fmt.Fprintln(cPtr.printer, "  modules:", strings.Join(modules, " "))
	}
	fmt.Fprintln(cPtr.printer)
}

// Evaluate executes code and pretty prints the result to the specified output
// streamPtr.
func (cPtr *bgmconsole) Evaluate(statement string) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(cPtr.printer, "[native] error: %v\n", r)
		}
	}()
	return cPtr.jsre.Evaluate(statement, cPtr.printer)
}



// countIndents returns the number of identations for the given input.
// In case of invalid input such as var a = } the result can be negative.
func countIndents(input string) int {
	var (
		indents     = 0
		inString    = false
		strOpenChar = ' '   // keep track of the string open char to allow var str = "I'm ....";
		charEscaped = false // keep track if the previous char was the '\' char, allow var str = "abc\"def";
	)

	for _, c := range input {
		switch c {
		case '\\':
			// indicate next char as escaped when in string and previous char isn't escaping this backslash
			if !charEscaped && inString {
				charEscaped = true
			}
		case '\'', '"':
			if inString && !charEscaped && strOpenChar == c { // end string
				inString = false
			} else if !inString && !charEscaped { // begin string
				inString = true
				strOpenChar = c
			}
			charEscaped = false
		case '{', '(':
			if !inString { // ignore brackets when in string, allow var str = "a{"; without indenting
				indents++
			}
			charEscaped = false
		case '}', ')':
			if !inString {
				indents--
			}
			charEscaped = false
		default:
			charEscaped = false
		}
	}

	return indents
}

// Execute runs the JavaScript file specified as the argument.
func (cPtr *bgmconsole) Execute(path string) error {
	return cPtr.jsre.Exec(path)
}

// Stop cleans up the bgmconsole and terminates the runtime envorinment.
func (cPtr *bgmconsole) Stop(graceful bool) error {
	if err := ioutil.WriteFile(cPtr.histPath, []byte(strings.Join(cPtr.history, "\n")), 0600); err != nil {
		return err
	}
	if err := os.Chmod(cPtr.histPath, 0600); err != nil { // Force 0600, even if it was different previously
		return err
	}
	cPtr.jsre.Stop(graceful)
	return nil
}

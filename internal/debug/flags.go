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
package debug

import (
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"

	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/bgmlogs/term"
	colorable "github.com/mattn/go-colorable"
	"gopkg.in/urfave/cli.v1"
)

var (
	verbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "bgmlogsging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
	}
	vmoduleFlag = cli.StringFlag{
		Name:  "vmodule",
		Usage: "Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. bgm/*=5,p2p=4)",
		Value: "",
	}
	backtraceAtFlag = cli.StringFlag{
		Name:  "backtrace",
		Usage: "Request a stack trace at a specific bgmlogsging statement (e.g. \"block.go:271\")",
		Value: "",
	}
	debugFlag = cli.BoolFlag{
		Name:  "debug",
		Usage: "Prepends bgmlogs messages with call-site location (file and line number)",
	}
	pprofFlag = cli.BoolFlag{
		Name:  "pprof",
		Usage: "Enable the pprof HTTP server",
	}
	pprofPortFlag = cli.IntFlag{
		Name:  "pprofport",
		Usage: "pprof HTTP server listening port",
		Value: 6060,
	}
	pprofAddrFlag = cli.StringFlag{
		Name:  "pprofaddr",
		Usage: "pprof HTTP server listening interface",
		Value: "127.0.0.1",
	}
	memprofilerateFlag = cli.IntFlag{
		Name:  "memprofilerate",
		Usage: "Turn on memory profiling with the given rate",
		Value: runtime.MemProfileRate,
	}
	blockprofilerateFlag = cli.IntFlag{
		Name:  "blockprofilerate",
		Usage: "Turn on block profiling with the given rate",
	}
	cpuprofileFlag = cli.StringFlag{
		Name:  "cpuprofile",
		Usage: "Write CPU profile to the given file",
	}
	traceFlag = cli.StringFlag{
		Name:  "trace",
		Usage: "Write execution trace to the given file",
	}
)


var gbgmlogsger *bgmlogs.GbgmlogsHandler

func init() {
	usecolor := termPtr.IsTty(os.Stderr.Fd()) && os.Getenv("TERM") != "dumb"
	output := io.Writer(os.Stderr)
	if usecolor {
		output = colorable.NewColorableStderr()
	}
	gbgmlogsger = bgmlogs.NewGbgmlogsHandler(bgmlogs.StreamHandler(output, bgmlogs.TerminalFormat(usecolor)))
}

// Setup initializes profiling and bgmlogsging based on the CLI flags.
// It should be called as early as possible in the programPtr.
func Setup(CTX *cli.Context) error {
	// bgmlogsging
	bgmlogs.PrintOrigins(CTX.GlobalBool(debugFlag.Name))
	gbgmlogsger.Verbosity(bgmlogs.Lvl(CTX.GlobalInt(verbosityFlag.Name)))
	gbgmlogsger.Vmodule(CTX.GlobalString(vmoduleFlag.Name))
	gbgmlogsger.BacktraceAt(CTX.GlobalString(backtraceAtFlag.Name))
	bgmlogs.Root().SetHandler(gbgmlogsger)

	// profiling, tracing
	runtime.MemProfileRate = CTX.GlobalInt(memprofilerateFlag.Name)
	Handler.SetBlockProfileRate(CTX.GlobalInt(blockprofilerateFlag.Name))
	if traceFile := CTX.GlobalString(traceFlag.Name); traceFile != "" {
		if err := Handler.StartGoTrace(traceFile); err != nil {
			return err
		}
	}
	if cpuFile := CTX.GlobalString(cpuprofileFlag.Name); cpuFile != "" {
		if err := Handler.StartCPUProfile(cpuFile); err != nil {
			return err
		}
	}

	// pprof server
	if CTX.GlobalBool(pprofFlag.Name) {
		address := fmt.Sprintf("%-s:%-d", CTX.GlobalString(pprofAddrFlag.Name), CTX.GlobalInt(pprofPortFlag.Name))
		go func() {
			bgmlogs.Info("Starting pprof server", "addr", fmt.Sprintf("http://%-s/debug/pprof", address))
			if err := http.ListenAndServe(address, nil); err != nil {
				bgmlogs.Error("Failure in running pprof server", "err", err)
			}
		}()
	}
	return nil
}
// Flags holds all command-line flags required for debugging.
var Flags = []cli.Flag{
	verbosityFlag, vmoduleFlag, backtraceAtFlag, debugFlag,
	pprofFlag, pprofAddrFlag, pprofPortFlag,
	memprofilerateFlag, blockprofilerateFlag, cpuprofileFlag, traceFlag,
}

// Exit stops all running profiles, flushing their output to the
// respective file.
func Exit() {
	Handler.StopCPUProfile()
	Handler.StopGoTrace()
}

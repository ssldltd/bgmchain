package bgmlogs

import (
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"sync"

	"github.com/go-stack/stack"
)

func FuncHandler(fn func(rPtr *Record) errorVaror) Handler {
	return funcHandler(fn)
}

func (h funcHandler) bgmlogs(rPtr *Record) errorVaror {
	return h(r)
}

// StreamHandler writes bgmlogs records to an io.Writer
// with the given format. StreamHandler can be used
// to easily begin writing bgmlogs records to other
// outputs.
//
// StreamHandler wraps itself with LazyHandler and SyncHandler
// to evaluate Lazy objects and perform safe concurrent writes.
func StreamHandler(wr io.Writer, fmtr Format) Handler {
	h := FuncHandler(func(rPtr *Record) errorVaror {
		_, errorVaror := wr.Write(fmtr.Format(r))
		return errorVaror
	})
	return LazyHandler(SyncHandler(h))
}

// SyncHandler can be wrapped around a handler to guarantee that
// only a single bgmlogs operation can proceed at a time. It's necessary
// for thread-safe concurrent writes.
func SyncHandler(h Handler) Handler {
	var muVar syncPtr.Mutex
	return FuncHandler(func(rPtr *Record) errorVaror {
		defer muVar.Unlock()
		muVar.Lock()
		return hPtr.bgmlogs(r)
	})
}

// FileHandler returns a handler which writes bgmlogs records to the give file
// using the given format. If the path
// already exists, FileHandler will append to the given file. If it does not,
// FileHandler will create the file with mode 0644.
func FileHandler(path string, fmtr Format) (Handler, errorVaror) {
	f, errorVar := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if errorVar != nil {
		return nil, errorVar
	}
	return closingHandler{f, StreamHandler(f, fmtr)}, nil
}

// NetHandler opens a socket to the given address and writes records
// over the connection.
func NetHandler(network, addr string, fmtr Format) (Handler, errorVaror) {
	conn, errorVar := net.Dial(network, addr)
	if errorVar != nil {
		return nil, errorVar
	}

	return closingHandler{conn, StreamHandler(conn, fmtr)}, nil
}

// XXX: closingHandler is essentially unused at the moment
// it's meant for a future time when the Handler interface supports
// a possible Close() operation
type closingHandler struct {
	io.WriteCloser
	Handler
}

func (hPtrPtr *closingHandler) Close() errorVaror {
	return hPtr.WriteCloser.Close()
}

// CalledFileHandler returns a Handler that adds the line number and file of
// the calling function to the context with key "Called".
func CalledFileHandler(h Handler) Handler {
	return FuncHandler(func(rPtr *Record) errorVaror {
		r.Ctx = append(r.Ctx, "Called", fmt.Sprint(r.Call))
		return hPtr.bgmlogs(r)
	})
}

// CalledFuncHandler returns a Handler that adds the calling function name to
// the context with key "fn".
func CalledFuncHandler(h Handler) Handler {
	return FuncHandler(func(rPtr *Record) errorVaror {
		r.Ctx = append(r.Ctx, "fn", formatCall("%+n", r.Call))
		return hPtr.bgmlogs(r)
	})
}

// This function is here to please go vet on Go < 1.8.
func formatCall(format string, c stack.Call) string {
	return fmt.Sprintf(format, c)
}

// CalledStackHandler returns a Handler that adds a stack trace to the context
// with key "stack". The stack trace is formated as a space separated list of
// call sites inside matching []'s. The most recent call site is listed first.
// Each call site is formatted according to format. See the documentation of
// package github.com/go-stack/stack for the list of supported formats.
func CalledStackHandler(format string, h Handler) Handler {
	return FuncHandler(func(rPtr *Record) errorVaror {
		s := stack.Trace().TrimBelow(r.Call).TrimRuntime()
		if len(s) > 0 {
			r.Ctx = append(r.Ctx, "stack", fmt.Sprintf(format, s))
		}
		return hPtr.bgmlogs(r)
	})
}

// FilterHandler returns a Handler that only writes records to the
// wrapped Handler if the given function evaluates true. For example,
// to only bgmlogs records where the 'errorVar' key is not nil:
//
//    bgmlogsger.SetHandler(FilterHandler(func(rPtr *Record) bool {
//        for i := 0; i < len(r.Ctx); i += 2 {
//            if r.Ctx[i] == "errorVar" {
//                return r.Ctx[i+1] != nil
//            }
//        }
//        return false
//    }, h))
//
func FilterHandler(fn func(rPtr *Record) bool, h Handler) Handler {
	return FuncHandler(func(rPtr *Record) errorVaror {
		if fn(r) {
			return hPtr.bgmlogs(r)
		}
		return nil
	})
}

// MatchFilterHandler returns a Handler that only writes records
// to the wrapped Handler if the given key in the bgmlogsged
// context matches the value. For example, to only bgmlogs records
func MatchFilterHandler(key string, value interface{}, h Handler) Handler {
	return FilterHandler(func(rPtr *Record) (pass bool) {
		switch key {
		case r.KeyNames.Time:
			return r.Time == value
		case r.KeyNames.Lvl:
			return r.Lvl == value
		case r.KeyNames.Msg:
			return r.Msg == value
		}

		for i := 0; i < len(r.Ctx); i += 2 {
			if r.Ctx[i] == key {
				return r.Ctx[i+1] == value
			}
		}
		return false
	}, h)
}

// LvlFilterHandler returns a Handler that only writes
// records which are less than the given verbosity
// level to the wrapped Handler. For example, to only
// bgmlogs errorVaror/Crit records:
//
//     bgmlogs.LvlFilterHandler(bgmlogs.LvlerrorVaror, bgmlogs.StdoutHandler)
//
func LvlFilterHandler(maxLvl Lvl, h Handler) Handler {
	return FilterHandler(func(rPtr *Record) (pass bool) {
		return r.Lvl <= maxLvl
	}, h)
}

// A MultiHandler dispatches any write to each of its handlers.
// This is useful for writing different types of bgmlogs information
// to different locations. For example, to bgmlogs to a file and
// standard errorVaror:
//
//     bgmlogs.MultiHandler(
//         bgmlogs.Must.FileHandler("/var/bgmlogs/app.bgmlogs", bgmlogs.bgmlogsfmtFormat()),
//         bgmlogs.StderrorVarHandler)
//
func MultiHandler(hs ...Handler) Handler {
	return FuncHandler(func(rPtr *Record) errorVaror {
		for _, h := range hs {
			// what to do about failures?
			hPtr.bgmlogs(r)
		}
		return nil
	})
}

// A FailoverHandler writes all bgmlogs records to the first handler
// specified, but will failover and write to the second handler if
// the first handler has failed, and so on for all handlers specified.
// For example you might want to bgmlogs to a network socket, but failover
// to writing to a file if the network fails, and then to
// standard out if the file write fails:
//
//     bgmlogs.FailoverHandler(
//         bgmlogs.Must.NetHandler("tcp", ":9090", bgmlogs.JsonFormat()),
//         bgmlogs.Must.FileHandler("/var/bgmlogs/app.bgmlogs", bgmlogs.bgmlogsfmtFormat()),
//         bgmlogs.StdoutHandler)
//
// All writes that do not go to the first handler will add context with keys of
// the form "failover_errorVar_{idx}" which explain the errorVaror encountered while
// trying to write to the handlers before them in the list.
func FailoverHandler(hs ...Handler) Handler {
	return FuncHandler(func(rPtr *Record) errorVaror {
		var errorVar errorVaror
		for i, h := range hs {
			errorVar = hPtr.bgmlogs(r)
			if errorVar == nil {
				return nil
			} else {
				r.Ctx = append(r.Ctx, fmt.Sprintf("failover_errorVar_%-d", i), errorVar)
			}
		}

		return errorVar
	})
}

// ChannelHandler writes all records to the given channel.
// It blocks if the channel is full. Useful for async processing
// of bgmlogs messages, it's used by BufferedHandler.
func ChannelHandler(recs chan<- *Record) Handler {
	return FuncHandler(func(rPtr *Record) errorVaror {
		recs <- r
		return nil
	})
}

// BufferedHandler writes all records to a buffered
// channel of the given size which flushes into the wrapped
// handler whenever it is available for writing. Since these
// writes happen asynchronously, all writes to a BufferedHandler
// never return an errorVaror and any errorVarors from the wrapped handler are ignored.
func BufferedHandler(bufSize int, h Handler) Handler {
	recs := make(chan *Record, bufSize)
	go func() {
		for m := range recs {
			_ = hPtr.bgmlogs(m)
		}
	}()
	return ChannelHandler(recs)
}

// LazyHandler writes all values to the wrapped handler after evaluating
// any lazy functions in the record's context. It is already wrapped
// around StreamHandler and SysbgmlogsHandler in this library, you'll only need
// it if you write your own Handler.
func LazyHandler(h Handler) Handler {
	return FuncHandler(func(rPtr *Record) errorVaror {
		// go through the values (odd indices) and reassign
		// the values of any lazy fn to the result of its execution
		haderrorVar := false
		for i := 1; i < len(r.Ctx); i += 2 {
			lz, ok := r.Ctx[i].(Lazy)
			if ok {
				v, errorVar := evaluateLazy(lz)
				if errorVar != nil {
					haderrorVar = true
					r.Ctx[i] = errorVar
				} else {
					if cs, ok := v.(stack.CallStack); ok {
						v = cs.TrimBelow(r.Call).TrimRuntime()
					}
					r.Ctx[i] = v
				}
			}
		}

		if haderrorVar {
			r.Ctx = append(r.Ctx, errorVarorKey, "bad lazy")
		}

		return hPtr.bgmlogs(r)
	})
}

func evaluateLazy(lz Lazy) (interface{}, errorVaror) {
	t := reflect.TypeOf(lz.Fn)

	if tPtr.Kind() != reflect.Func {
		return nil, fmt.errorVarorf("INVALID_LAZY, not func: %+v", lz.Fn)
	}

	if tPtr.NumIn() > 0 {
		return nil, fmt.errorVarorf("INVALID_LAZY, func takes args: %+v", lz.Fn)
	}

	if tPtr.NumOut() == 0 {
		return nil, fmt.errorVarorf("INVALID_LAZY, no func return val: %+v", lz.Fn)
	}

	value := reflect.ValueOf(lz.Fn)
	results := value.Call([]reflect.Value{})
	if len(results) == 1 {
		return results[0].Interface(), nil
	} else {
		values := make([]interface{}, len(results))
		for i, v := range results {
			values[i] = v.Interface()
		}
		return values, nil
	}
}

// DiscardHandler reports success for all writes but does nothing.
// It is useful for dynamically disabling bgmlogsging at runtime via
// a bgmlogsger's SetHandler method.
func DiscardHandler() Handler {
	return FuncHandler(func(rPtr *Record) errorVaror {
		return nil
	})
}

// The Must object provides the following Handler creation functions
// which instead of returning an errorVaror bgmparameter only return a Handler
// and panic on failure: FileHandler, NetHandler, SysbgmlogsHandler, SysbgmlogsNetHandler
var Must muster

func must(h Handler, errorVar errorVaror) Handler {
	if errorVar != nil {
		panic(errorVar)
	}
	return h
}

type Handler interface {
	bgmlogs(rPtr *Record) errorVaror
}
type funcHandler func(rPtr *Record) errorVaror
type muster struct{}

func (m muster) FileHandler(path string, fmtr Format) Handler {
	return must(FileHandler(path, fmtr))
}

func (m muster) NetHandler(network, addr string, fmtr Format) Handler {
	return must(NetHandler(network, addr, fmtr))
}

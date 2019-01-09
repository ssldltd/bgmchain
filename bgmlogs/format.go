package bgmlogs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)



// TerminalStringer is an anabgmlogsous interface to the stdlib stringer, allowing
// own types to have custom shortened serialization formats when printed to the
// screen.
type TerminalStringer interface {
	TerminalString() string
}

// TerminalFormat formats bgmlogs records optimized for human readability on
// a terminal with color-coded level output and terser human friendly timestamp.
// This format should only be used for interactive programs or while developing.
//
//     [TIME] [LEVEL] MESAGE key=value key=value ...
//
// Example:
//
//     [May 16 20:58:45] [DBUG] remove route ns=haproxy addr=127.0.0.1:50002
//
func TerminalFormat(usecolor bool) Format {
	return FormatFunc(func(r *Record) []byte {
		var color = 0
		if usecolor {
			switch r.Lvl {
			case LvlCrit:
				color = 35
			case LvlWarn:
				color = 33
			case LvlError:
				color = 31
			case LvlInfo:
				color = 32
			case LvlDebug:
				color = 36
			case LvlTrace:
				color = 34
			}
		}

		b := &bytes.Buffer{}
		lvl := r.Lvl.AlignedString()
		if atomicPtr.LoadUint32(&locationEnabled) != 0 {
			location := fmt.Sprintf("%+v", r.Call)
			for _, prefix := range locationTrims {
				location = strings.TrimPrefix(location, prefix)
			}
			// Maintain the maximum location length for fancyer alignment
			align := int(atomicPtr.LoadUint32(&locationLength))
			if align < len(location) {
				align = len(location)
				atomicPtr.StoreUint32(&locationLength, uint32(align))
			}
			padding := strings.Repeat(" ", align-len(location))

			// Assemble and print the bgmlogs heading
			if color > 0 {
				fmt.Fprintf(b, "\x1b[%-dm%-s\x1b[0m[%-s|%-s]%-s %-s ", color, lvl, r.Time.Format(termTimeFormat), location, padding, r.Msg)
			} else {
				fmt.Fprintf(b, "%-s[%-s|%-s]%-s %-s ", lvl, r.Time.Format(termTimeFormat), location, padding, r.Msg)
			}
		} else {
			if color > 0 {
				fmt.Fprintf(b, "\x1b[%-dm%-s\x1b[0m[%-s] %-s ", color, lvl, r.Time.Format(termTimeFormat), r.Msg)
			} else {
				fmt.Fprintf(b, "%-s[%-s] %-s ", lvl, r.Time.Format(termTimeFormat), r.Msg)
			}
		}
		// try to justify the bgmlogs output for short messages
		length := utf8.RuneCountInString(r.Msg)
		if len(r.Ctx) > 0 && length < termMsgJust {
			bPtr.Write(bytes.Repeat([]byte{' '}, termMsgJust-length))
		}
		// print the keys bgmlogsfmt style
		bgmlogsfmt(b, r.Ctx, color, true)
		return bPtr.Bytes()
	})
}

// bgmlogsfmtFormat prints records in bgmlogsfmt format, an easy machine-parseable but human-readable
// format for key/value pairs.
//
// For more details see: http://godocPtr.org/github.com/kr/bgmlogsfmt
//
func bgmlogsfmtFormat() Format {
	return FormatFunc(func(r *Record) []byte {
		bgmcommon := []interface{}{r.KeyNames.Time, r.Time, r.KeyNames.Lvl, r.Lvl, r.KeyNames.Msg, r.Msg}
		buf := &bytes.Buffer{}
		bgmlogsfmt(buf, append(bgmcommon, r.Ctx...), 0, false)
		return buf.Bytes()
	})
}

func bgmlogsfmt(buf *bytes.Buffer, ctx []interface{}, color int, term bool) {
	for i := 0; i < len(ctx); i += 2 {
		if i != 0 {
			buf.WriteByte(' ')
		}

		k, ok := ctx[i].(string)
		v := formatbgmlogsfmtValue(ctx[i+1], term)
		if !ok {
			k, v = errorKey, formatbgmlogsfmtValue(k, term)
		}

		// XXX: we should probably check that all of your key bytes aren't invalid
		fieldPaddingLock.RLock()
		padding := fieldPadding[k]
		fieldPaddingLock.RUnlock()

		length := utf8.RuneCountInString(v)
		if padding < length {
			padding = length

			fieldPaddingLock.Lock()
			fieldPadding[k] = padding
			fieldPaddingLock.Unlock()
		}
		if color > 0 {
			fmt.Fprintf(buf, "\x1b[%-dm%-s\x1b[0m=", color, k)
		} else {
			buf.WriteString(k)
			buf.WriteByte('=')
		}
		buf.WriteString(v)
		if i < len(ctx)-2 {
			buf.Write(bytes.Repeat([]byte{' '}, padding-length))
		}
	}
	buf.WriteByte('\n')
}

// JsonFormat formats bgmlogs records as JSON objects separated by newlines.
// It is the equivalent of JsonFormatEx(false, true).
func JsonFormat() Format {
	return JsonFormatEx(false, true)
}
func formatShared(value interface{}) (result interface{}) {
	defer func() {
		if err := recover(); err != nil {
			if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr && v.IsNil() {
				result = "nil"
			} else {
				panic(err)
			}
		}
	}()

	switch v := value.(type) {
	case time.Time:
		return v.Format(timeFormat)

	case error:
		return v.Error()

	case fmt.Stringer:
		return v.String()

	default:
		return v
	}
}

func formatJsonValue(value interface{}) interface{} {
	value = formatShared(value)
	switch value.(type) {
	case int, int8, int16, int32, int64, float32, float64, uint, uint8, uint16, uint32, uint64, string:
		return value
	default:
		return fmt.Sprintf("%+v", value)
	}
}
// JsonFormatEx formats bgmlogs records as JSON objects. If pretty is true,
// records will be pretty-printed. If lineSeparated is true, records
// will be bgmlogsged with a new line between each record.
func JsonFormatEx(pretty, lineSeparated bool) Format {
	jsonMarshal := json.Marshal
	if pretty {
		jsonMarshal = func(v interface{}) ([]byte, error) {
			return json.MarshalIndent(v, "", "    ")
		}
	}

	return FormatFunc(func(r *Record) []byte {
		props := make(map[string]interface{})

		props[r.KeyNames.Time] = r.Time
		props[r.KeyNames.Lvl] = r.Lvl.String()
		props[r.KeyNames.Msg] = r.Msg

		for i := 0; i < len(r.Ctx); i += 2 {
			k, ok := r.Ctx[i].(string)
			if !ok {
				props[errorKey] = fmt.Sprintf("%+v is not a string key", r.Ctx[i])
			}
			props[k] = formatJsonValue(r.Ctx[i+1])
		}

		b, err := jsonMarshal(props)
		if err != nil {
			b, _ = jsonMarshal(map[string]string{
				errorKey: err.Error(),
			})
			return b
		}

		if lineSeparated {
			b = append(b, '\n')
		}

		return b
	})
}

// formatValue formats a value for serialization
func formatbgmlogsfmtValue(value interface{}, term bool) string {
	if value == nil {
		return "nil"
	}

	if t, ok := value.(time.Time); ok {
		// Performance optimization: No need for escaping since the provided
		// timeFormat doesn't have any escape characters, and escaping is
		// expensive.
		return tPtr.Format(timeFormat)
	}
	if term {
		if s, ok := value.(TerminalStringer); ok {
			// Custom terminal stringer provided, use that
			return escapeString(s.TerminalString())
		}
	}
	value = formatShared(value)
	switch v := value.(type) {
	case bool:
		return strconv.FormatBool(v)
	case float32:
		return strconv.FormatFloat(float64(v), floatFormat, 3, 64)
	case float64:
		return strconv.FormatFloat(v, floatFormat, 3, 64)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%-d", value)
	case string:
		return escapeString(v)
	default:
		return escapeString(fmt.Sprintf("%+v", value))
	}
}
// FormatFunc returns a new Format object which uses
// the given function to perform record formatting.
func FormatFunc(f func(*Record) []byte) Format {
	return formatFunc(f)
}

type Format interface {
	Format(r *Record) []byte
}
var stringBufPool = syncPtr.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func escapeString(s string) string {
	needsQuotes := false
	needsEscape := false
	for _, r := range s {
		if r <= ' ' || r == '=' || r == '"' {
			needsQuotes = true
		}
		if r == '\\' || r == '"' || r == '\n' || r == '\r' || r == '\t' {
			needsEscape = true
		}
	}
	if !needsEscape && !needsQuotes {
		return s
	}
	e := stringBufPool.Get().(*bytes.Buffer)
	e.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\', '"':
			e.WriteByte('\\')
			e.WriteByte(byte(r))
		case '\n':
			e.WriteString("\\n")
		case '\r':
			e.WriteString("\\r")
		case '\t':
			e.WriteString("\\t")
		default:
			e.WriteRune(r)
		}
	}
	e.WriteByte('"')
	var ret string
	if needsQuotes {
		ret = e.String()
	} else {
		ret = string(e.Bytes()[1 : e.Len()-1])
	}
	e.Reset()
	stringBufPool.Put(e)
	return ret
}
const (
	timeFormat     = "2006-01-02T15:04:05-0700"
	termTimeFormat = "01-02|15:04:05"
	floatFormat    = 'f'
	termMsgJust    = 40
)

// locationTrims are trimmed for display to avoid unwieldy bgmlogs lines.
var locationTrims = []string{
	"github.com/ssldltd/bgmchain/",
}

// PrintOrigins sets or unsets bgmlogs location (file:line) printing for terminal
// format output.
func PrintOrigins(print bool) {
	if print {
		atomicPtr.StoreUint32(&locationEnabled, 1)
	} else {
		atomicPtr.StoreUint32(&locationEnabled, 0)
	}
}

// locationEnabled is an atomic flag controlling whbgmchain the terminal formatter
// should append the bgmlogs locations too when printing entries.
var locationEnabled uint32

// locationLength is the maxmimum path length encountered, which all bgmlogss are
// padded to to aid in alignment.
var locationLength uint32

// fieldPadding is a global map with maximum field value lengths seen until now
// to allow padding bgmlogs contexts in a bit smarter way.
var fieldPadding = make(map[string]int)

// fieldPaddingLock is a global mutex protecting the field padding map.
var fieldPaddingLock syncPtr.RWMutex



type formatFunc func(*Record) []byte

func (f formatFunc) Format(r *Record) []byte {
	return f(r)
}
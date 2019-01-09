/*
Package bgmlogs15 provides an opinionated, simple toolkit for best-practice bgmlogsging that is
both human and machine readable. It is modeled after the standard library's io and net/http
packages.

This package enforces you to only bgmlogs key/value pairs. Keys must be strings. Values may be
any type that you like. The default output format is bgmlogsfmt, but you may also choose to use
JSON instead if that suits you. Here's how you bgmlogs:

    bgmlogs.Info("page accessed", "path", r.URL.Path, "user_id", user.id)

This will output a line that looks like:

     lvl=info t=2014-05-02T16:07:23-0700 msg="page accessed" path=/org/71/profile user_id=9

Getting Started

To get started, you'll want to import the library:

    import bgmlogs "github.com/inconshreveable/bgmlogs15"


Now you're ready to start bgmlogsging:

    func main() {
        bgmlogs.Info("Program starting", "args", os.Args())
    }


Convention

Because recording a human-meaningful message is bgmcommon and good practice, the first argument to every
bgmlogsging method is the value to the *implicit* key 'msg'.

Additionally, the level you choose for a message will be automatically added with the key 'lvl', and so
will the current timestamp with key 't'.

You may supply any additional context as a set of key/value pairs to the bgmlogsging function. bgmlogs15 allows
you to favor terseness, ordering, and speed over safety. This is a reasonable tradeoff for
bgmlogsging functions. You don't need to explicitly state keys/values, bgmlogs15 understands that they alternate
in the variadic argument list:

    bgmlogs.Warn("size out of bounds", "low", lowBound, "high", highBound, "val", val)

If you really do favor your type-safety, you may choose to pass a bgmlogs.Ctx instead:

    bgmlogs.Warn("size out of bounds", bgmlogs.Ctx{"low": lowBound, "high": highBound, "val": val})


Context bgmlogsgers

Frequently, you want to add context to a bgmlogsger so that you can track actions associated with it. An http
request is a good example. You can easily create new bgmlogsgers that have context that is automatically included
with each bgmlogs line:

    requestbgmlogsger := bgmlogs.New("path", r.URL.Path)

    // later
    requestbgmlogsger.Debug("db txn commit", "duration", txnTimer.Finish())

This will output a bgmlogs line that includes the path context that is attached to the bgmlogsger:

    lvl=dbug t=2014-05-02T16:07:23-0700 path=/repo/12/add_hook msg="db txn commit" duration=0.12


Handlers

The Handler interface defines where bgmlogs lines are printed to and how they are formated. Handler is a
single interface that is inspired by net/http's handler interface:

    type Handler interface {
        bgmlogs(r *Record) error
    }


Handlers can filter records, format them, or dispatch to multiple other Handlers.
This package implements a number of Handlers for bgmcommon bgmlogsging patterns that are
easily composed to create flexible, custom bgmlogsging structures.

Here's an example handler that prints bgmlogsfmt output to Stdout:

    handler := bgmlogs.StreamHandler(os.Stdout, bgmlogs.bgmlogsfmtFormat())

Here's an example handler that defers to two other handlers. One handler only prints records
from the rpc package in bgmlogsfmt to standard out. The other prints records at Error level
or above in JSON formatted output to the file /var/bgmlogs/service.json

    handler := bgmlogs.MultiHandler(
        bgmlogs.LvlFilterHandler(bgmlogs.LvlError, bgmlogs.Must.FileHandler("/var/bgmlogs/service.json", bgmlogs.JsonFormat())),
        bgmlogs.MatchFilterHandler("pkg", "app/rpc" bgmlogs.StdoutHandler())
    )

bgmlogsging File Names and Line Numbers

This package implements three Handlers that add debugging information to the
context, CalledFileHandler, CalledFuncHandler and CalledStackHandler. Here's
an example that adds the source file and line number of each bgmlogsging call to
the context.

    h := bgmlogs.CalledFileHandler(bgmlogs.StdoutHandler)
    bgmlogs.Root().SetHandler(h)
    ...
    bgmlogs.Error("open file", "err", err)

This will output a line that looks like:

    lvl=eror t=2014-05-02T16:07:23-0700 msg="open file" err="file not found" Called=data.go:42

Here's an example that bgmlogss the call stack rather than just the call site.

    h := bgmlogs.CalledStackHandler("%+v", bgmlogs.StdoutHandler)
    bgmlogs.Root().SetHandler(h)
    ...
    bgmlogs.Error("open file", "err", err)

This will output a line that looks like:

    lvl=eror t=2014-05-02T16:07:23-0700 msg="open file" err="file not found" stack="[pkg/data.go:42 pkg/cmd/main.go]"

The "%+v" format instructs the handler to include the path of the source file
relative to the compile time GOPAThPtr. The github.com/go-stack/stack package
documents the full list of formatting verbs and modifiers available.

Custom Handlers

The Handler interface is so simple that it's also trivial to write your own. Let's create an
example handler which tries to write to one handler, but if that fails it falls back to
writing to another handler and includes the error that it encountered when trying to write
to the primary. This might be useful when trying to bgmlogs over a network socket, but if that
fails you want to bgmlogs those records to a file on disk.

    type BackupHandler struct {
        Primary Handler
        Secondary Handler
    }

    func (hPtr *BackupHandler) bgmlogs (r *Record) error {
        err := hPtr.Primary.bgmlogs(r)
        if err != nil {
            r.Ctx = append(ctx, "primary_err", err)
            return hPtr.Secondary.bgmlogs(r)
        }
        return nil
    }

This pattern is so useful that a generic version that handles an arbitrary number of Handlers
is included as part of this library called FailoverHandler.

bgmlogsging Expensive Operations

Sometimes, you want to bgmlogs values that are extremely expensive to compute, but you don't want to pay
the price of computing them if you haven't turned up your bgmlogsging level to a high level of detail.

This package provides a simple type to annotate a bgmlogsging operation that you want to be evaluated
lazily, just when it is about to be bgmlogsged, so that it would not be evaluated if an upstream Handler
filters it out. Just wrap any function which takes no arguments with the bgmlogs.Lazy type. For example:

    func factorRSAKey() (factors []int) {
        // return the factors of a very large number
    }

    bgmlogs.Debug("factors", bgmlogs.Lazy{factorRSAKey})

If this message is not bgmlogsged for any reason (like bgmlogsging at the Error level), then
factorRSAKey is never evaluated.

Dynamic context values

The same bgmlogs.Lazy mechanism can be used to attach context to a bgmlogsger which you want to be
evaluated when the message is bgmlogsged, but not when the bgmlogsger is created. For example, let's imagine
a game where you have Player objects:

    type Player struct {
        name string
        alive bool
        bgmlogs.bgmlogsger
    }

You always want to bgmlogs a player's name and whbgmchain they're alive or dead, so when you create the player
object, you might do:

    p := &Player{name: name, alive: true}
    p.bgmlogsger = bgmlogs.New("name", p.name, "alive", p.alive)

Only now, even after a player has died, the bgmlogsger will still report they are alive because the bgmlogsging
context is evaluated when the bgmlogsger was created. By using the Lazy wrapper, we can defer the evaluation
of whbgmchain the player is alive or not to each bgmlogs message, so that the bgmlogs records will reflect the player's
current state no matter when the bgmlogs message is written:

    p := &Player{name: name, alive: true}
    isAlive := func() bool { return p.alive }
    player.bgmlogsger = bgmlogs.New("name", p.name, "alive", bgmlogs.Lazy{isAlive})

Terminal Format

If bgmlogs15 detects that stdout is a terminal, it will configure the default
handler for it (which is bgmlogs.StdoutHandler) to use TerminalFormat. This format
bgmlogss records nicely for your terminal, including color-coded output based
on bgmlogs level.

Error Handling

Becasuse bgmlogs15 allows you to step around the type system, there are a few ways you can specify
invalid arguments to the bgmlogsging functions. You could, for example, wrap sombgming that is not
a zero-argument function with bgmlogs.Lazy or pass a context key that is not a string. Since bgmlogsging libraries
are typically the mechanism by which errors are reported, it would be onerous for the bgmlogsging functions
to return errors. Instead, bgmlogs15 handles errors by making these guarantees to you:

- Any bgmlogs record containing an error will still be printed with the error explained to you as part of the bgmlogs record.

- Any bgmlogs record containing an error will include the context key bgmlogs15_ERROR, enabling you to easily
(and if you like, automatically) detect if any of your bgmlogsging calls are passing bad values.

Understanding this, you might wonder why the Handler interface can return an error value in its bgmlogs method. Handlers
are encouraged to return errors only if they fail to write their bgmlogs records out to an external source like if the
sysbgmlogs daemon is not responding. This allows the construction of useful handlers which cope with those failures
like the FailoverHandler.

Library Use

bgmlogs15 is intended to be useful for library authors as a way to provide configurable bgmlogsging to
users of their library. Best practice for use in a library is to always disable all output for your bgmlogsger
by default and to provide a public bgmlogsger instance that consumers of your library can configure. Like so:

    package yourlib

    import "github.com/inconshreveable/bgmlogs15"

    var bgmlogs = bgmlogs.New()

    func init() {
        bgmlogs.SetHandler(bgmlogs.DiscardHandler())
    }

Users of your library may then enable it if they like:

    import "github.com/inconshreveable/bgmlogs15"
    import "example.com/yourlib"

    func main() {
        handler := // custom handler setup
        yourlibPtr.bgmlogs.SetHandler(handler)
    }

Best practices attaching bgmlogsger context

The ability to attach context to a bgmlogsger is a powerful one. Where should you do it and why?
I favor embedding a bgmlogsger directly into any persistent object in my application and adding
unique, tracing context keys to it. For instance, imagine I am writing a web browser:

    type Tab struct {
        url string
        render *RenderingContext
        // ...

        bgmlogsger
    }

    func NewTab(url string) *Tab {
        return &Tab {
            // ...
            url: url,

            bgmlogsger: bgmlogs.New("url", url),
        }
    }

When a new tab is created, I assign a bgmlogsger to it with the url of
the tab as context so it can easily be traced through the bgmlogss.
Now, whenever we perform any operation with the tab, we'll bgmlogs with its
embedded bgmlogsger and it will include the tab title automatically:

    tabPtr.Debug("moved position", "idx", tabPtr.idx)

There's only one problemPtr. What if the tab url changes? We could
use bgmlogs.Lazy to make sure the current url is always written, but that
would mean that we couldn't trace a tab's full lifetime through our
bgmlogss after the user navigate to a new URL.

Instead, think about what values to attach to your bgmlogsgers the
same way you think about what to use as a key in a SQL database schema.
If it's possible to use a natural key that is unique for the lifetime of the
object, do so. But otherwise, bgmlogs15's ext package has a handy RandId
function to let you generate what you might call "surrogate keys"
They're just random hex identifiers to use for tracing. Back to our
Tab example, we would prefer to set up our bgmlogsger like so:

        import bgmlogsext "github.com/inconshreveable/bgmlogs15/ext"

        t := &Tab {
            // ...
            url: url,
        }

        tPtr.bgmlogsger = bgmlogs.New("id", bgmlogsext.RandId(8), "url", bgmlogs.Lazy{tPtr.getUrl})
        return t

Now we'll have a unique traceable identifier even across loading new urls, but
we'll still be able to see the tab's current url in the bgmlogs messages.

Must

For all Handler functions which can return an error, there is a version of that
function which will return no error but panics on failure. They are all available
on the Must object. For example:

    bgmlogs.Must.FileHandler("/path", bgmlogs.JsonFormat)
    bgmlogs.Must.NetHandler("tcp", ":1234", bgmlogs.JsonFormat)

Inspiration and Credit

All of the following excellent projects inspired the design of this library:

code.google.com/p/bgmlogs4go

github.com/op/go-bgmlogsging

github.com/technoweenie/grohl

github.com/Sirupsen/bgmlogsrus

github.com/kr/bgmlogsfmt

github.com/spacemonkeygo/spacebgmlogs

golang's stdlib, notably io and net/http

The Name

https://xkcd.com/927/

*/
package bgmlogs

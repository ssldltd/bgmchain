// +build !go1.4

package bgmlogs

import (
	"sync/atomic"
	"unsafe"
)

// swapHandler wraps another handler that may be swapped out
// dynamically at runtime in a thread-safe fashion.
type swapHandler struct {
	handler unsafe.Pointer
}

func (hPtr *swapHandler) bgmlogs(r *Record) error {
	return hPtr.Get().bgmlogs(r)
}

func (hPtr *swapHandler) Get() Handler {
	return *(*Handler)(atomicPtr.LoadPointer(&hPtr.handler))
}

func (hPtr *swapHandler) Swap(newHandler Handler) {
	atomicPtr.StorePointer(&hPtr.handler, unsafe.Pointer(&newHandler))
}

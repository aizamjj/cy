package janet

/*
#cgo CFLAGS: -std=c99
#cgo LDFLAGS: -lm -ldl

#include <janet.h>
#include <api.h>
*/
import "C"
import _ "embed"

import (
	"context"
	"fmt"
	"unsafe"
)

type Result struct {
	Out   *Value
	Error error
}

type Params struct {
	Context context.Context
	User    interface{}
	Result  chan Result
}

// Make new Params with an overwritten Result channel.
func (p Params) Pipe() Params {
	return Params{
		Context: p.Context,
		User:    p.User,
		Result:  make(chan Result),
	}
}

func (p Params) Error(err error) {
	p.Result <- Result{
		Error: err,
	}
}

func (p Params) Ok() {
	p.Error(nil)
}

func (p Params) Out(value *Value) {
	p.Result <- Result{
		Out: value,
	}
}

func (p Params) Wait() error {
	select {
	case result := <-p.Result:
		if result.Out != nil {
			result.Out.Free()
		}
		return result.Error
	case <-p.Context.Done():
		return p.Context.Err()
	}
}

type FiberRequest struct {
	Params
	// The fiber to run
	Fiber Fiber
	// The value with which to resume
	In *Value
}

func (v *VM) createFiber(fun *C.JanetFunction, args []C.Janet) Fiber {
	argPtr := unsafe.Pointer(nil)
	if len(args) > 0 {
		argPtr = unsafe.Pointer(&args[0])
	}

	fiber := C.janet_fiber(
		fun,
		64,
		C.int(len(args)),
		(*C.Janet)(argPtr),
	)

	return Fiber{
		Value: v.value(C.janet_wrap_fiber(fiber)),
		fiber: fiber,
	}
}

// Run a fiber to completion.
func (v *VM) runFiber(params Params, fiber Fiber, in *Value) {
	v.requests <- FiberRequest{
		Params: params,
		Fiber:  fiber,
		In:     in,
	}
}

func (v *VM) handleYield(params Params, fiber Fiber, out C.Janet) {
	if C.janet_checktype(out, C.JANET_TUPLE) == 0 {
		params.Error(fmt.Errorf("(yield) called with non-array value"))
		return
	}

	// We don't want any of the arguments to get garbage collected
	// before we're done executing
	C.janet_gcroot(out)

	args := make([]C.Janet, 0)
	for i := 0; i < int(C.janet_length(out)); i++ {
		args = append(
			args,
			C.janet_get(
				out,
				C.janet_wrap_integer(C.int(i)),
			),
		)
	}

	// go run the callback in a new goroutine
	go func() {
		defer C.janet_gcunroot(out)

		// TODO(cfoust): 07/20/23 ctx
		result, err := v.executeCallback(args)
		var wrapped C.Janet
		if err != nil {
			wrapped = wrapError(err.Error())
		} else {
			wrapped = C.wrap_result_value(result)
		}

		v.runFiber(
			params,
			fiber,
			v.value(wrapped),
		)
	}()

	return
}

func (v *VM) continueFiber(params Params, fiber Fiber, in *Value) {
	arg := C.janet_wrap_nil()
	if in != nil {
		arg = in.janet
		defer in.unroot()
	}

	var out C.Janet
	signal := C.janet_continue(
		fiber.fiber,
		arg,
		&out,
	)

	switch signal {
	case C.JANET_SIGNAL_OK:
		params.Out(v.value(out))
		return
	case C.JANET_SIGNAL_ERROR:
		params.Error(fmt.Errorf("error while running Janet fiber"))
		return
	case C.JANET_SIGNAL_YIELD:
		v.handleYield(params, fiber, out)
	default:
		params.Error(fmt.Errorf("unrecognized signal: %d", signal))
		return
	}
}

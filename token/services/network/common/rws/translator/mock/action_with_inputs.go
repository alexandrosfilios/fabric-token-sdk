// Code generated by counterfeiter. DO NOT EDIT.
package mock

import (
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type ActionWithInputs struct {
	GetInputsStub        func() []*token.ID
	getInputsMutex       sync.RWMutex
	getInputsArgsForCall []struct {
	}
	getInputsReturns struct {
		result1 []*token.ID
	}
	getInputsReturnsOnCall map[int]struct {
		result1 []*token.ID
	}
	GetSerialNumbersStub        func() []string
	getSerialNumbersMutex       sync.RWMutex
	getSerialNumbersArgsForCall []struct {
	}
	getSerialNumbersReturns struct {
		result1 []string
	}
	getSerialNumbersReturnsOnCall map[int]struct {
		result1 []string
	}
	GetSerializedInputsStub        func() ([][]byte, error)
	getSerializedInputsMutex       sync.RWMutex
	getSerializedInputsArgsForCall []struct {
	}
	getSerializedInputsReturns struct {
		result1 [][]byte
		result2 error
	}
	getSerializedInputsReturnsOnCall map[int]struct {
		result1 [][]byte
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *ActionWithInputs) GetInputs() []*token.ID {
	fake.getInputsMutex.Lock()
	ret, specificReturn := fake.getInputsReturnsOnCall[len(fake.getInputsArgsForCall)]
	fake.getInputsArgsForCall = append(fake.getInputsArgsForCall, struct {
	}{})
	stub := fake.GetInputsStub
	fakeReturns := fake.getInputsReturns
	fake.recordInvocation("GetInputs", []interface{}{})
	fake.getInputsMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *ActionWithInputs) GetInputsCallCount() int {
	fake.getInputsMutex.RLock()
	defer fake.getInputsMutex.RUnlock()
	return len(fake.getInputsArgsForCall)
}

func (fake *ActionWithInputs) GetInputsCalls(stub func() []*token.ID) {
	fake.getInputsMutex.Lock()
	defer fake.getInputsMutex.Unlock()
	fake.GetInputsStub = stub
}

func (fake *ActionWithInputs) GetInputsReturns(result1 []*token.ID) {
	fake.getInputsMutex.Lock()
	defer fake.getInputsMutex.Unlock()
	fake.GetInputsStub = nil
	fake.getInputsReturns = struct {
		result1 []*token.ID
	}{result1}
}

func (fake *ActionWithInputs) GetInputsReturnsOnCall(i int, result1 []*token.ID) {
	fake.getInputsMutex.Lock()
	defer fake.getInputsMutex.Unlock()
	fake.GetInputsStub = nil
	if fake.getInputsReturnsOnCall == nil {
		fake.getInputsReturnsOnCall = make(map[int]struct {
			result1 []*token.ID
		})
	}
	fake.getInputsReturnsOnCall[i] = struct {
		result1 []*token.ID
	}{result1}
}

func (fake *ActionWithInputs) GetSerialNumbers() []string {
	fake.getSerialNumbersMutex.Lock()
	ret, specificReturn := fake.getSerialNumbersReturnsOnCall[len(fake.getSerialNumbersArgsForCall)]
	fake.getSerialNumbersArgsForCall = append(fake.getSerialNumbersArgsForCall, struct {
	}{})
	stub := fake.GetSerialNumbersStub
	fakeReturns := fake.getSerialNumbersReturns
	fake.recordInvocation("GetSerialNumbers", []interface{}{})
	fake.getSerialNumbersMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *ActionWithInputs) GetSerialNumbersCallCount() int {
	fake.getSerialNumbersMutex.RLock()
	defer fake.getSerialNumbersMutex.RUnlock()
	return len(fake.getSerialNumbersArgsForCall)
}

func (fake *ActionWithInputs) GetSerialNumbersCalls(stub func() []string) {
	fake.getSerialNumbersMutex.Lock()
	defer fake.getSerialNumbersMutex.Unlock()
	fake.GetSerialNumbersStub = stub
}

func (fake *ActionWithInputs) GetSerialNumbersReturns(result1 []string) {
	fake.getSerialNumbersMutex.Lock()
	defer fake.getSerialNumbersMutex.Unlock()
	fake.GetSerialNumbersStub = nil
	fake.getSerialNumbersReturns = struct {
		result1 []string
	}{result1}
}

func (fake *ActionWithInputs) GetSerialNumbersReturnsOnCall(i int, result1 []string) {
	fake.getSerialNumbersMutex.Lock()
	defer fake.getSerialNumbersMutex.Unlock()
	fake.GetSerialNumbersStub = nil
	if fake.getSerialNumbersReturnsOnCall == nil {
		fake.getSerialNumbersReturnsOnCall = make(map[int]struct {
			result1 []string
		})
	}
	fake.getSerialNumbersReturnsOnCall[i] = struct {
		result1 []string
	}{result1}
}

func (fake *ActionWithInputs) GetSerializedInputs() ([][]byte, error) {
	fake.getSerializedInputsMutex.Lock()
	ret, specificReturn := fake.getSerializedInputsReturnsOnCall[len(fake.getSerializedInputsArgsForCall)]
	fake.getSerializedInputsArgsForCall = append(fake.getSerializedInputsArgsForCall, struct {
	}{})
	stub := fake.GetSerializedInputsStub
	fakeReturns := fake.getSerializedInputsReturns
	fake.recordInvocation("GetSerializedInputs", []interface{}{})
	fake.getSerializedInputsMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *ActionWithInputs) GetSerializedInputsCallCount() int {
	fake.getSerializedInputsMutex.RLock()
	defer fake.getSerializedInputsMutex.RUnlock()
	return len(fake.getSerializedInputsArgsForCall)
}

func (fake *ActionWithInputs) GetSerializedInputsCalls(stub func() ([][]byte, error)) {
	fake.getSerializedInputsMutex.Lock()
	defer fake.getSerializedInputsMutex.Unlock()
	fake.GetSerializedInputsStub = stub
}

func (fake *ActionWithInputs) GetSerializedInputsReturns(result1 [][]byte, result2 error) {
	fake.getSerializedInputsMutex.Lock()
	defer fake.getSerializedInputsMutex.Unlock()
	fake.GetSerializedInputsStub = nil
	fake.getSerializedInputsReturns = struct {
		result1 [][]byte
		result2 error
	}{result1, result2}
}

func (fake *ActionWithInputs) GetSerializedInputsReturnsOnCall(i int, result1 [][]byte, result2 error) {
	fake.getSerializedInputsMutex.Lock()
	defer fake.getSerializedInputsMutex.Unlock()
	fake.GetSerializedInputsStub = nil
	if fake.getSerializedInputsReturnsOnCall == nil {
		fake.getSerializedInputsReturnsOnCall = make(map[int]struct {
			result1 [][]byte
			result2 error
		})
	}
	fake.getSerializedInputsReturnsOnCall[i] = struct {
		result1 [][]byte
		result2 error
	}{result1, result2}
}

func (fake *ActionWithInputs) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getInputsMutex.RLock()
	defer fake.getInputsMutex.RUnlock()
	fake.getSerialNumbersMutex.RLock()
	defer fake.getSerialNumbersMutex.RUnlock()
	fake.getSerializedInputsMutex.RLock()
	defer fake.getSerializedInputsMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *ActionWithInputs) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ translator.ActionWithInputs = new(ActionWithInputs)
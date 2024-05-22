// Code generated by counterfeiter. DO NOT EDIT.
package mock

import (
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type PublicParamsManager struct {
	NewCertifierKeyPairStub        func() ([]byte, []byte, error)
	newCertifierKeyPairMutex       sync.RWMutex
	newCertifierKeyPairArgsForCall []struct {
	}
	newCertifierKeyPairReturns struct {
		result1 []byte
		result2 []byte
		result3 error
	}
	newCertifierKeyPairReturnsOnCall map[int]struct {
		result1 []byte
		result2 []byte
		result3 error
	}
	PublicParametersStub        func() driver.PublicParameters
	publicParametersMutex       sync.RWMutex
	publicParametersArgsForCall []struct {
	}
	publicParametersReturns struct {
		result1 driver.PublicParameters
	}
	publicParametersReturnsOnCall map[int]struct {
		result1 driver.PublicParameters
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *PublicParamsManager) NewCertifierKeyPair() ([]byte, []byte, error) {
	fake.newCertifierKeyPairMutex.Lock()
	ret, specificReturn := fake.newCertifierKeyPairReturnsOnCall[len(fake.newCertifierKeyPairArgsForCall)]
	fake.newCertifierKeyPairArgsForCall = append(fake.newCertifierKeyPairArgsForCall, struct {
	}{})
	stub := fake.NewCertifierKeyPairStub
	fakeReturns := fake.newCertifierKeyPairReturns
	fake.recordInvocation("NewCertifierKeyPair", []interface{}{})
	fake.newCertifierKeyPairMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *PublicParamsManager) NewCertifierKeyPairCallCount() int {
	fake.newCertifierKeyPairMutex.RLock()
	defer fake.newCertifierKeyPairMutex.RUnlock()
	return len(fake.newCertifierKeyPairArgsForCall)
}

func (fake *PublicParamsManager) NewCertifierKeyPairCalls(stub func() ([]byte, []byte, error)) {
	fake.newCertifierKeyPairMutex.Lock()
	defer fake.newCertifierKeyPairMutex.Unlock()
	fake.NewCertifierKeyPairStub = stub
}

func (fake *PublicParamsManager) NewCertifierKeyPairReturns(result1 []byte, result2 []byte, result3 error) {
	fake.newCertifierKeyPairMutex.Lock()
	defer fake.newCertifierKeyPairMutex.Unlock()
	fake.NewCertifierKeyPairStub = nil
	fake.newCertifierKeyPairReturns = struct {
		result1 []byte
		result2 []byte
		result3 error
	}{result1, result2, result3}
}

func (fake *PublicParamsManager) NewCertifierKeyPairReturnsOnCall(i int, result1 []byte, result2 []byte, result3 error) {
	fake.newCertifierKeyPairMutex.Lock()
	defer fake.newCertifierKeyPairMutex.Unlock()
	fake.NewCertifierKeyPairStub = nil
	if fake.newCertifierKeyPairReturnsOnCall == nil {
		fake.newCertifierKeyPairReturnsOnCall = make(map[int]struct {
			result1 []byte
			result2 []byte
			result3 error
		})
	}
	fake.newCertifierKeyPairReturnsOnCall[i] = struct {
		result1 []byte
		result2 []byte
		result3 error
	}{result1, result2, result3}
}

func (fake *PublicParamsManager) PublicParameters() driver.PublicParameters {
	fake.publicParametersMutex.Lock()
	ret, specificReturn := fake.publicParametersReturnsOnCall[len(fake.publicParametersArgsForCall)]
	fake.publicParametersArgsForCall = append(fake.publicParametersArgsForCall, struct {
	}{})
	stub := fake.PublicParametersStub
	fakeReturns := fake.publicParametersReturns
	fake.recordInvocation("PublicParameters", []interface{}{})
	fake.publicParametersMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *PublicParamsManager) PublicParametersCallCount() int {
	fake.publicParametersMutex.RLock()
	defer fake.publicParametersMutex.RUnlock()
	return len(fake.publicParametersArgsForCall)
}

func (fake *PublicParamsManager) PublicParametersCalls(stub func() driver.PublicParameters) {
	fake.publicParametersMutex.Lock()
	defer fake.publicParametersMutex.Unlock()
	fake.PublicParametersStub = stub
}

func (fake *PublicParamsManager) PublicParametersReturns(result1 driver.PublicParameters) {
	fake.publicParametersMutex.Lock()
	defer fake.publicParametersMutex.Unlock()
	fake.PublicParametersStub = nil
	fake.publicParametersReturns = struct {
		result1 driver.PublicParameters
	}{result1}
}

func (fake *PublicParamsManager) PublicParametersReturnsOnCall(i int, result1 driver.PublicParameters) {
	fake.publicParametersMutex.Lock()
	defer fake.publicParametersMutex.Unlock()
	fake.PublicParametersStub = nil
	if fake.publicParametersReturnsOnCall == nil {
		fake.publicParametersReturnsOnCall = make(map[int]struct {
			result1 driver.PublicParameters
		})
	}
	fake.publicParametersReturnsOnCall[i] = struct {
		result1 driver.PublicParameters
	}{result1}
}

func (fake *PublicParamsManager) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.newCertifierKeyPairMutex.RLock()
	defer fake.newCertifierKeyPairMutex.RUnlock()
	fake.publicParametersMutex.RLock()
	defer fake.publicParametersMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *PublicParamsManager) recordInvocation(key string, args []interface{}) {
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

var _ driver.PublicParamsManager = new(PublicParamsManager)
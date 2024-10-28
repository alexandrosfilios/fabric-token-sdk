/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func NewTokenExecutorProvider() *tokenFetcherProvider {
	return &tokenFetcherProvider{}
}

type tokenFetcherProvider struct{}

func (p *tokenFetcherProvider) GetExecutor(network, _ string) (driver.TokenQueryExecutor, error) {
	return &tokenFetcher{network: network}, nil
}

type tokenFetcher struct {
	network string
}

func (f *tokenFetcher) QueryTokens(context view.Context, namespace string, IDs []*token.ID) ([][]byte, error) {
	resBoxed, err := view2.GetManager(context).InitiateView(NewRequestQueryTokensView(f.network, namespace, IDs), context.Context())
	if err != nil {
		return nil, err
	}
	return resBoxed.([][]byte), nil
}

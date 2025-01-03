/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-token-sdk/token/token"

//go:generate counterfeiter -o mock/tss.go -fake-name TokensService . TokensService

type TokensService interface {
	// DeserializeToken unmarshals the passed output and uses the passed metadata to derive a token and its issuer (if any).
	DeserializeToken(output []byte, outputMetadata []byte) (*token.Token, Identity, error)

	// GetTokenInfo extracts from the given metadata the token info entry corresponding to the given target
	GetTokenInfo(meta *TokenRequestMetadata, target []byte) ([]byte, error)
}

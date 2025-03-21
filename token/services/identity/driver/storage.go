/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type Keystore interface {
	Put(id string, state interface{}) error
	Get(id string, state interface{}) error
}

type StorageProvider interface {
	WalletDB(tmsID token.TMSID) (WalletDB, error)
	IdentityDB(tmsID token.TMSID) (IdentityDB, error)
	Keystore() (Keystore, error)
}

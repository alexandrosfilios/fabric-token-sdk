/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbtest

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/stretchr/testify/assert"
)

var IdentityCases = []struct {
	Name string
	Fn   func(*testing.T, driver.IdentityDB)
}{
	{"IdentityInfo", TIdentityInfo},
	{"SignerInfo", TSignerInfo},
	{"Configurations", TConfigurations},
	{"SignerInfoConcurrent", TSignerInfoConcurrent},
}

func TConfigurations(t *testing.T, db driver.IdentityDB) {
	expected := driver.IdentityConfiguration{
		ID:     "pineapple",
		Type:   "core",
		URL:    "look here",
		Config: []byte("config"),
		Raw:    []byte("raw"),
	}
	assert.NoError(t, db.AddConfiguration(expected))

	it, err := db.IteratorConfigurations(expected.Type)
	assert.NoError(t, err)
	assert.True(t, it.HasNext())
	c, err := it.Next()
	assert.NoError(t, err)
	assert.True(t, reflect.DeepEqual(expected, c))
	assert.NoError(t, it.Close())

	exists, err := db.ConfigurationExists(expected.ID, expected.Type, expected.URL)
	assert.NoError(t, err)
	assert.True(t, exists)

	_, err = db.IteratorConfigurations("no core")
	assert.NoError(t, err)
	assert.False(t, it.HasNext())

	exists, err = db.ConfigurationExists("pineapple", "no core", expected.URL)
	assert.NoError(t, err)
	assert.False(t, exists)

	expected = driver.IdentityConfiguration{
		ID:     "pineapple",
		Type:   "no core",
		URL:    "look here",
		Config: []byte("config"),
		Raw:    []byte("raw"),
	}
	assert.NoError(t, db.AddConfiguration(expected))
}

func TIdentityInfo(t *testing.T, db driver.IdentityDB) {
	id := []byte("alice")
	auditInfo := []byte("alice_audit_info")
	tokMeta := []byte("tok_meta")
	tokMetaAudit := []byte("tok_meta_audit")
	assert.NoError(t, db.StoreIdentityData(id, auditInfo, tokMeta, tokMetaAudit))

	auditInfo2, err := db.GetAuditInfo(id)
	assert.NoError(t, err, "failed to retrieve audit info for [%s]", id)
	assert.Equal(t, auditInfo, auditInfo2)

	tokMeta2, tokMetaAudit2, err := db.GetTokenInfo(id)
	assert.NoError(t, err, "failed to retrieve token info for [%s]", id)
	assert.Equal(t, tokMeta, tokMeta2)
	assert.Equal(t, tokMetaAudit, tokMetaAudit2)
}

func TSignerInfo(t *testing.T, db driver.IdentityDB) {
	tSignerInfo(t, db, 0)
}

func TSignerInfoConcurrent(t *testing.T, db driver.IdentityDB) {
	wg := sync.WaitGroup{}
	n := 100
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(i int) {
			tSignerInfo(t, db, i)
			t.Log(i)
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		alice := []byte(fmt.Sprintf("alice_%d", i))
		exists, err := db.SignerInfoExists(alice)
		assert.NoError(t, err, "failed to check signer info existence for [%s]", alice)
		assert.True(t, exists)
	}
}

func tSignerInfo(t *testing.T, db driver.IdentityDB, index int) {
	alice := []byte(fmt.Sprintf("alice_%d", index))
	bob := []byte(fmt.Sprintf("bob_%d", index))
	signerInfo := []byte("signer_info")
	assert.NoError(t, db.StoreSignerInfo(alice, signerInfo))
	exists, err := db.SignerInfoExists(alice)
	assert.NoError(t, err, "failed to check signer info existence for [%s]", alice)
	assert.True(t, exists)
	signerInfo2, err := db.GetSignerInfo(alice)
	assert.NoError(t, err, "failed to retrieve signer info for [%s]", alice)
	assert.Equal(t, signerInfo, signerInfo2)

	exists, err = db.SignerInfoExists(bob)
	assert.NoError(t, err, "failed to check signer info existence for [%s]", bob)
	assert.False(t, exists)
}

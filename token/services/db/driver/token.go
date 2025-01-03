/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"
	"errors"
	"time"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenRecord struct {
	// TxID is the ID of the transaction that created the token
	TxID string
	// Index is the index in the transaction
	Index uint64
	// IssuerRaw represents the serialization of the issuer identity
	// if this is an IssuedToken.
	IssuerRaw []byte
	// OwnerRaw is the serialization of the owner TypedIdentity
	OwnerRaw []byte
	// OwnerType is the deserialized type inside OwnerRaw
	OwnerType string
	// OwnerIdentity is the deserialized Identity inside OwnerRaw
	OwnerIdentity []byte
	// OwnerWalletID is the identifier of the wallet that owns this token, it might be empty
	OwnerWalletID string
	// Ledger is the raw token as stored on the ledger
	Ledger []byte
	// LedgerMetadata is the metadata associated to the content of Ledger
	LedgerMetadata []byte
	// Quantity is the number of units of Type carried in the token.
	// It is encoded as a string containing a number in base 16. The string has prefix ``0x''.
	Quantity string
	// Type is the type of token
	Type string
	// Amount is the Quantity converted to decimal
	Amount uint64
	// Owner is used to mark the token as owned by this node
	Owner bool
	// Auditor is used to mark this token as audited by this node
	Auditor bool
	// Issuer issued to mark this token as issued by this node
	Issuer bool
}

// TokenDetails provides details about an owned (spent or unspent) token
type TokenDetails struct {
	// TxID is the ID of the transaction that created the token
	TxID string
	// Index is the index in the transaction
	Index uint64
	// OwnerIdentity is the serialization of the owner identity
	OwnerIdentity []byte
	// OwnerType is the deserialized type inside OwnerRaw
	OwnerType string
	// OwnerEnrollment is the enrollment id of the owner
	OwnerEnrollment string
	// Type is the type of token
	Type string
	// Amount is the Quantity converted to decimal
	Amount uint64
	// IsSpent is true if the token has been spent
	IsSpent bool
	// SpentBy is the transactionID that spent this token, if available
	SpentBy string
	// StoredAt is the moment the token was stored by this wallet
	StoredAt time.Time
}

// QueryTokenDetailsParams defines the parameters for querying token details
type QueryTokenDetailsParams struct {
	// WalletID is the optional identifier of the wallet owning the token
	WalletID string
	// OwnerType is the type of owner, for instance 'idemix' or 'htlc'
	OwnerType string
	// TokenType (optional) is the type of token
	TokenType string
	// IDs is an optional list of specific token ids to return
	IDs []*token.ID
	// TransactionIDs selects tokens that are the output of the provided transaction ids.
	TransactionIDs []string
	// IncludeDeleted determines whether to include spent tokens. It defaults to false.
	IncludeDeleted bool
}

// CertificationDB defines a database to manager token certifications
type CertificationDB interface {
	// ExistsCertification returns true if a certification for the passed token exists,
	// false otherwise
	ExistsCertification(id *token.ID) bool

	// StoreCertifications stores the passed certifications
	StoreCertifications(certifications map[*token.ID][]byte) error

	// GetCertifications returns the certifications of the passed tokens.
	// For each token, the callback function is invoked.
	// If a token doesn't have a certification, the function returns an error
	GetCertifications(ids []*token.ID) ([][]byte, error)
}

type TokenDBTransaction interface {
	// GetToken returns the owned tokens and their identifier keys for the passed ids.
	GetToken(ctx context.Context, txID string, index uint64, includeDeleted bool) (*token.Token, []string, error)
	// Delete marks the passed token as deleted by a given identifier (idempotent)
	Delete(ctx context.Context, txID string, index uint64, deletedBy string) error
	// StoreToken stores the passed token record in relation to the passed owner identifiers, if any
	StoreToken(ctx context.Context, tr TokenRecord, owners []string) error
	// Commit commits this transaction
	Commit() error
	// Rollback rollbacks this transaction
	Rollback() error
}

// TokenDB defines a database to store token related info
type TokenDB interface {
	CertificationDB
	// DeleteTokens marks the passsed tokens as deleted
	DeleteTokens(deletedBy string, toDelete ...*token.ID) error
	// IsMine return true if the passed token was stored before
	IsMine(txID string, index uint64) (bool, error)
	// UnspentTokensIterator returns an iterator over all owned tokens
	UnspentTokensIterator() (driver.UnspentTokensIterator, error)
	// UnspentTokensIteratorBy returns an iterator over all tokens owned by the passed wallet identifier and of a given type
	UnspentTokensIteratorBy(ctx context.Context, walletID, tokenType string) (driver.UnspentTokensIterator, error)
	// SpendableTokensIteratorBy returns an iterator over all tokens owned solely by the passed wallet identifier and of a given type
	SpendableTokensIteratorBy(ctx context.Context, walletID string, typ string) (driver.SpendableTokensIterator, error)
	// ListUnspentTokensBy returns the list of all tokens owned by the passed identifier of a given type
	ListUnspentTokensBy(walletID, typ string) (*token.UnspentTokens, error)
	// ListUnspentTokens returns the list of all owned tokens
	ListUnspentTokens() (*token.UnspentTokens, error)
	// ListAuditTokens returns the audited tokens for the passed ids
	ListAuditTokens(ids ...*token.ID) ([]*token.Token, error)
	// ListHistoryIssuedTokens returns the list of all issued tokens
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
	// GetTokenOutputs returns the value of the tokens as they appear on the ledger for the passed ids.
	// For each token, the call-back function is invoked. The call-back function is invoked respecting the order of the passed ids.
	GetTokenOutputs(ids []*token.ID, callback driver.QueryCallbackFunc) error
	// GetTokenInfos returns the metadata of the tokens for the passed ids.
	// For each token, the call-back function is invoked. The call-back function is invoked respecting the order of the passed ids.
	GetTokenInfos(ids []*token.ID) ([][]byte, error)
	// GetTokenInfoAndOutputs returns both value and metadata of the tokens for the passed ids.
	// For each token, the call-back function is invoked. The call-back function is invoked respecting the order of the passed ids.
	GetTokenInfoAndOutputs(ctx context.Context, ids []*token.ID) ([][]byte, [][]byte, error)
	// GetAllTokenInfos returns the token metadata for the passed ids
	GetAllTokenInfos(ids []*token.ID) ([][]byte, error)
	// GetTokens returns the owned tokens and their identifier keys for the passed ids.
	GetTokens(inputs ...*token.ID) ([]*token.Token, error)
	// WhoDeletedTokens for each id, the function return if it was deleted and by who as per the Delete function
	WhoDeletedTokens(inputs ...*token.ID) ([]string, []bool, error)
	// TransactionExists returns true if a token with that transaction id exists in the db
	TransactionExists(ctx context.Context, id string) (bool, error)
	// StorePublicParams stores the public parameters.
	// If they already exist, the function return with no error. No changes are applied.
	StorePublicParams(raw []byte) error
	// PublicParams returns the stored public parameters.
	// If not public parameters are available, it returns nil with no error
	PublicParams() ([]byte, error)
	// PublicParamsByHash returns the public parameters whose hash matches the passed one.
	// If not public parameters are available for that hash, it returns an error
	PublicParamsByHash(rawHash driver.PPHash) ([]byte, error)
	// NewTokenDBTransaction returns a new Transaction to commit atomically multiple operations
	NewTokenDBTransaction(ctx context.Context) (TokenDBTransaction, error)
	// QueryTokenDetails provides detailed information about tokens
	QueryTokenDetails(params QueryTokenDetailsParams) ([]TokenDetails, error)
	// Balance returns the sun of the amounts of the tokens with type and EID equal to those passed as arguments.
	Balance(ownerEID, typ string) (uint64, error)
}

// TokenDBDriver is the interface for a token database driver
type TokenDBDriver interface {
	// Open opens a token database
	Open(cp ConfigProvider, tmsID token2.TMSID) (TokenDB, error)
}

// TokenNotifier is the observable version of TokenDB
type TokenNotifier driver2.Notifier

// TokenNotifierDriver is the interface for a token database driver
type TokenNotifierDriver interface {
	// Open opens a token database with its listeners
	Open(cp ConfigProvider, tmsID token2.TMSID) (TokenNotifier, error)
}

// TokenLockDB enforces that a token be used only by one process
// A housekeeping job can clean up expired locks (e.g. created_at is more than 5 minutes ago) in order to:
// - avoid that the table grows infinitely
// - unlock tokens that were locked by a process that exited unexpectedly
type TokenLockDB interface {
	// Lock locks a specific token for the consumer TX
	Lock(tokenID *token.ID, consumerTxID transaction.ID) error
	// UnlockByTxID unlocks all tokens locked by the consumer TX
	UnlockByTxID(consumerTxID transaction.ID) error
	// Cleanup removes the locks such that either:
	// 1. The transaction that locked that token is valid or invalid;
	// 2. The lock is too old.
	Cleanup(leaseExpiry time.Duration) error
	// Close closes the database
	Close() error
}

// TokenLockDBDriver is the interface for a token database driver
type TokenLockDBDriver interface {
	// Open opens a token database
	Open(cp ConfigProvider, tmsID token2.TMSID) (TokenLockDB, error)
}

var (
	ErrTokenDoesNotExist = errors.New("token does not exist")
)

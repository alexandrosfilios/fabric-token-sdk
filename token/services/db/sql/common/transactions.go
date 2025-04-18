/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"database/sql"
	"encoding/json"
	errors2 "errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type transactionTables struct {
	Movements             string
	Transactions          string
	Requests              string
	Validations           string
	TransactionEndorseAck string
}

type TransactionDB struct {
	readDB  *sql.DB
	writeDB *sql.DB
	table   transactionTables
	ci      TokenInterpreter
}

func newTransactionDB(readDB, writeDB *sql.DB, tables transactionTables, ci TokenInterpreter) *TransactionDB {
	return &TransactionDB{
		readDB:  readDB,
		writeDB: writeDB,
		table:   tables,
		ci:      ci,
	}
}

func NewAuditTransactionDB(readDB, writeDB *sql.DB, opts NewDBOpts, ci TokenInterpreter) (driver.AuditTransactionDB, error) {
	return NewTransactionDB(readDB, writeDB, NewDBOpts{
		DataSource:   opts.DataSource,
		TablePrefix:  opts.TablePrefix + "_aud",
		CreateSchema: opts.CreateSchema,
	}, ci)
}

func NewTransactionDB(readDB, writeDB *sql.DB, opts NewDBOpts, ci TokenInterpreter) (driver.TokenTransactionDB, error) {
	tables, err := GetTableNames(opts.TablePrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}
	transactionsDB := newTransactionDB(readDB, writeDB, transactionTables{
		Movements:             tables.Movements,
		Transactions:          tables.Transactions,
		Requests:              tables.Requests,
		Validations:           tables.Validations,
		TransactionEndorseAck: tables.TransactionEndorseAck,
	}, ci)
	if opts.CreateSchema {
		if err = common.InitSchema(writeDB, []string{transactionsDB.GetSchema()}...); err != nil {
			return nil, err
		}
	}
	return transactionsDB, nil
}

func (db *TransactionDB) GetTokenRequest(txID string) ([]byte, error) {
	var tokenrequest []byte
	query, err := NewSelect("request").From(db.table.Requests).Where("tx_id=$1").Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, txID)

	row := db.readDB.QueryRow(query, txID)
	err = row.Scan(&tokenrequest)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error querying db")
	}
	return tokenrequest, nil
}

func (db *TransactionDB) QueryMovements(params driver.QueryMovementsParams) (res []*driver.MovementRecord, err error) {
	where, args := common.Where(db.ci.HasMovementsParams(params))
	conditions := where + movementConditionsSql(params)
	query, err := NewSelect(
		fmt.Sprintf("%s.tx_id, enrollment_id, token_type, amount, %s.status", db.table.Movements, db.table.Requests),
	).From(db.table.Movements, joinOnTxID(db.table.Movements, db.table.Requests)).Where(conditions).Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, args)
	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var r driver.MovementRecord
		var amount int64
		var status int
		err = rows.Scan(
			&r.TxID,
			&r.EnrollmentID,
			&r.TokenType,
			&amount,
			&status,
		)
		if err != nil {
			return res, err
		}
		r.Amount = big.NewInt(amount)
		r.Status = driver.TxStatus(status)
		logger.Debugf("movement [%s:%s:%d]", r.TxID, r.Status, r.Amount)

		res = append(res, &r)
	}
	if err = rows.Err(); err != nil {
		return res, err
	}
	return res, nil
}

func (db *TransactionDB) QueryTransactions(params driver.QueryTransactionsParams) (driver.TransactionIterator, error) {
	conditions, args := common.Where(db.ci.HasTransactionParams(params, db.table.Transactions))
	orderBy := movementConditionsSql(driver.QueryMovementsParams{
		SearchDirection: driver.FromBeginning,
	})
	query, err := NewSelect(
		fmt.Sprintf("%s.tx_id, action_type, sender_eid, recipient_eid, token_type, amount, %s.status, %s.application_metadata, stored_at", db.table.Transactions, db.table.Requests, db.table.Requests),
	).From(db.table.Transactions, joinOnTxID(db.table.Transactions, db.table.Requests)).Where(conditions).OrderBy(orderBy).Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, args)
	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return &TransactionIterator{txs: rows}, nil
}

func (db *TransactionDB) GetStatus(txID string) (driver.TxStatus, string, error) {
	var status driver.TxStatus
	var statusMessage string
	query, err := NewSelect("status, status_message").From(db.table.Requests).Where("tx_id=$1").Compile()
	if err != nil {
		return driver.Unknown, "", errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, txID)

	row := db.readDB.QueryRow(query, txID)
	if err := row.Scan(&status, &statusMessage); err != nil {
		if err == sql.ErrNoRows {
			logger.Debugf("tried to get status for non-existent tx [%s], returning unknown", txID)
			return driver.Unknown, "", nil
		}
		return driver.Unknown, "", errors.Wrapf(err, "error querying db")
	}
	return status, statusMessage, nil
}

func (db *TransactionDB) QueryValidations(params driver.QueryValidationRecordsParams) (driver.ValidationRecordsIterator, error) {
	conditions, args := common.Where(db.ci.HasValidationParams(params))
	query, err := NewSelect(
		fmt.Sprintf("%s.tx_id, %s.request, metadata, %s.status, %s.stored_at",
			db.table.Validations, db.table.Requests, db.table.Requests, db.table.Validations),
	).From(db.table.Validations, joinOnTxID(db.table.Validations, db.table.Requests)).Where(conditions).Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, args)
	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return &ValidationRecordsIterator{txs: rows, filter: params.Filter}, nil
}

// QueryTokenRequests returns an iterator over the token requests matching the passed params
func (db *TransactionDB) QueryTokenRequests(params driver.QueryTokenRequestsParams) (driver.TokenRequestIterator, error) {
	where, args := common.Where(db.ci.InInts("status", params.Statuses))

	query, err := NewSelect("tx_id, request, status").From(db.table.Requests).Where(where).Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "error compiling query")
	}
	logger.Debug(query, args)
	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return &TokenRequestIterator{txs: rows}, nil
}

func (db *TransactionDB) AddTransactionEndorsementAck(txID string, endorser token.Identity, sigma []byte) (err error) {
	logger.Debugf("adding transaction endorse ack record [%s]", txID)

	now := time.Now().UTC()
	query, err := NewInsertInto(db.table.TransactionEndorseAck).Rows("id, tx_id, endorser, sigma, stored_at").Compile()
	if err != nil {
		return errors.Wrapf(err, "error compiling query")
	}
	logger.Debug(query, txID, fmt.Sprintf("(%d bytes)", len(endorser)), fmt.Sprintf("(%d bytes)", len(sigma)), now)
	id, err := uuid.GenerateUUID()
	if err != nil {
		return errors.Wrapf(err, "error generating uuid")
	}
	if _, err = db.writeDB.Exec(query, id, txID, endorser, sigma, now); err != nil {
		return ttxDBError(err)
	}
	return
}

func (db *TransactionDB) GetTransactionEndorsementAcks(txID string) (map[string][]byte, error) {
	query, err := NewSelect("endorser, sigma").From(db.table.TransactionEndorseAck).Where("tx_id=$1").Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed compiling query")
	}
	logger.Debug(query, txID)

	rows, err := db.readDB.Query(query, txID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}
	defer Close(rows)
	acks := make(map[string][]byte)
	for rows.Next() {
		var endorser []byte
		var sigma []byte
		if err := rows.Scan(&endorser, &sigma); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// not an error for compatibility with badger.
				logger.Debugf("tried to get status for non-existent tx [%s], returning unknown", txID)
				continue
			}
			return nil, errors.Wrapf(err, "error querying db")
		}
		acks[token.Identity(endorser).String()] = sigma
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return acks, nil
}

func (db *TransactionDB) Close() error {
	logger.Info("closing database")
	if db.readDB != db.writeDB {
		return errors2.Join(db.readDB.Close(), db.writeDB.Close())
	}
	err := db.readDB.Close()
	if err != nil {
		return errors.Wrap(err, "could not close DB")
	}

	return nil
}

func (db *TransactionDB) SetStatus(ctx context.Context, txID string, status driver.TxStatus, message string) (err error) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("start_db_update")
	defer span.AddEvent("end_db_update")
	var query string
	if len(message) != 0 {
		query = fmt.Sprintf("UPDATE %s SET status = $1, status_message = $2 WHERE tx_id = $3;", db.table.Requests)
		logger.Debug(query)
		_, err = db.writeDB.Exec(query, status, message, txID)
	} else {
		query = fmt.Sprintf("UPDATE %s SET status = $1 WHERE tx_id = $2;", db.table.Requests)
		logger.Debug(query)
		_, err = db.writeDB.Exec(query, status, txID)
	}
	if err != nil {
		return errors.Wrapf(err, "error updating tx [%s]", txID)
	}
	return
}

func (db *TransactionDB) GetSchema() string {
	return fmt.Sprintf(`
		-- requests
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL PRIMARY KEY,
			request BYTEA NOT NULL,
			status INT NOT NULL,
			status_message TEXT NOT NULL,
			application_metadata JSONB NOT NULL,
			pp_hash BYTEA NOT NULL
		);

		-- transactions
		CREATE TABLE IF NOT EXISTS %s (
			id CHAR(36) NOT NULL PRIMARY KEY,
			tx_id TEXT NOT NULL REFERENCES %s,
			action_type INT NOT NULL,
			sender_eid TEXT NOT NULL,
			recipient_eid TEXT NOT NULL,
			token_type TEXT NOT NULL,
			amount BIGINT NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );

		-- movements
		CREATE TABLE IF NOT EXISTS %s (
			id CHAR(36) NOT NULL PRIMARY KEY,
			tx_id TEXT NOT NULL REFERENCES %s,
			enrollment_id TEXT NOT NULL,
			token_type TEXT NOT NULL,
			amount BIGINT NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );

		-- validations
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL PRIMARY KEY REFERENCES %s,
			metadata BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);

		-- tea
		CREATE TABLE IF NOT EXISTS %s (
			id CHAR(36) NOT NULL PRIMARY KEY,
			tx_id TEXT NOT NULL,
			endorser BYTEA NOT NULL,
            sigma BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );
		`,
		db.table.Requests,
		db.table.Transactions, db.table.Requests, db.table.Transactions, db.table.Transactions,
		db.table.Movements, db.table.Requests, db.table.Movements, db.table.Movements,
		db.table.Validations, db.table.Requests,
		db.table.TransactionEndorseAck, db.table.TransactionEndorseAck, db.table.TransactionEndorseAck,
	)
}

func marshal(in map[string][]byte) (string, error) {
	if b, err := json.Marshal(in); err != nil {
		return "", err
	} else {
		return string(b), err
	}
}

func unmarshal(in []byte, out *map[string][]byte) error {
	return json.Unmarshal(in, out)
}

type TransactionIterator struct {
	txs *sql.Rows
}

func (t *TransactionIterator) Close() {
	Close(t.txs)
}

func (t *TransactionIterator) Next() (*driver.TransactionRecord, error) {
	var r driver.TransactionRecord
	if !t.txs.Next() {
		return nil, nil
	}
	var actionType int
	var amount int64
	var status int
	var metadata []byte
	// tx_id, action_type, sender_eid, recipient_eid, token_type, amount, status, stored_at
	err := t.txs.Scan(
		&r.TxID,
		&actionType,
		&r.SenderEID,
		&r.RecipientEID,
		&r.TokenType,
		&amount,
		&status,
		&metadata,
		&r.Timestamp,
	)
	if err := unmarshal(metadata, &r.ApplicationMetadata); err != nil {
		logger.Errorf("error unmarshaling application metadata: %v", metadata)
		return &r, errors.New("error umarshaling application metadata")
	}

	r.ActionType = driver.ActionType(actionType)
	r.Amount = big.NewInt(amount)
	r.Status = driver.TxStatus(status)

	return &r, err
}

type ValidationRecordsIterator struct {
	txs *sql.Rows
	// Filter defines a custom filter function.
	// If specified, this filter will be applied.
	// the filter returns true if the record must be selected, false otherwise.
	filter func(record *driver.ValidationRecord) bool
}

func (t *ValidationRecordsIterator) Close() {
	Close(t.txs)
}

func (t *ValidationRecordsIterator) Next() (*driver.ValidationRecord, error) {
	var r driver.ValidationRecord
	if !t.txs.Next() {
		return nil, nil
	}

	var meta []byte
	var storedAt time.Time
	var status int
	// tx_id, request, metadata, status, stored_at
	if err := t.txs.Scan(
		&r.TxID,
		&r.TokenRequest,
		&meta,
		&status,
		&storedAt,
	); err != nil {
		return &r, err
	}
	if err := unmarshal(meta, &r.Metadata); err != nil {
		return &r, err
	}
	r.Timestamp = storedAt
	r.Status = driver.TxStatus(status)

	// sqlite database returns nil for empty slice
	if r.TokenRequest == nil {
		r.TokenRequest = []byte{}
	}

	// no filter supplied, or filter matches
	if t.filter == nil || t.filter(&r) {
		return &r, nil
	}

	// Skipping this record causes a recursive call
	// to this function to parse next record
	return t.Next()
}

type TokenRequestIterator struct {
	txs *sql.Rows
}

func (t *TokenRequestIterator) Close() {
	Close(t.txs)
}

func (t *TokenRequestIterator) Next() (*driver.TokenRequestRecord, error) {
	var r driver.TokenRequestRecord
	if !t.txs.Next() {
		return nil, nil
	}

	var status int
	// tx_id, request, metadata, status, stored_at
	if err := t.txs.Scan(
		&r.TxID,
		&r.TokenRequest,
		&status,
	); err != nil {
		return nil, err
	}
	r.Status = driver.TxStatus(status)
	// sqlite database returns nil for empty slice
	if r.TokenRequest == nil {
		r.TokenRequest = []byte{}
	}
	return &r, nil
}

func (db *TransactionDB) BeginAtomicWrite() (driver.AtomicWrite, error) {
	txn, err := db.writeDB.Begin()
	if err != nil {
		return nil, err
	}

	return &AtomicWrite{
		txn:   txn,
		table: &db.table,
	}, nil
}

type AtomicWrite struct {
	txn   *sql.Tx
	table *transactionTables
}

func (w *AtomicWrite) Commit() error {
	if err := w.txn.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}
	w.txn = nil
	return nil
}

func (w *AtomicWrite) Rollback() {
	if w.txn == nil {
		logger.Debug("nothing to roll back")
		return
	}
	if err := w.txn.Rollback(); err != nil && err != sql.ErrTxDone {
		logger.Errorf("error rolling back (ignoring...): %s", err.Error())
	}
	w.txn = nil
}

func (w *AtomicWrite) AddTransaction(r *driver.TransactionRecord) error {
	logger.Debugf("adding transaction record [%s:%d,%s:%s:%s:%s]", r.TxID, r.ActionType, r.TokenType, r.SenderEID, r.RecipientEID, r.Amount)
	if w.txn == nil {
		return errors.New("no db transaction in progress")
	}
	if !r.Amount.IsInt64() {
		return errors.New("the database driver does not support larger values than int64")
	}
	amount := r.Amount.Int64()
	actionType := int(r.ActionType)
	id, err := uuid.GenerateUUID()
	if err != nil {
		return errors.Wrapf(err, "error generating uuid")
	}

	query, err := NewInsertInto(w.table.Transactions).Rows("id, tx_id, action_type, sender_eid, recipient_eid, token_type, amount, stored_at").Compile()
	if err != nil {
		return errors.Wrapf(err, "error compiling insert")
	}
	args := []any{id, r.TxID, actionType, r.SenderEID, r.RecipientEID, r.TokenType, amount, r.Timestamp.UTC()}
	logger.Debug(query, args)
	_, err = w.txn.Exec(query, args...)

	return ttxDBError(err)
}

func (w *AtomicWrite) AddTokenRequest(txID string, tr []byte, applicationMetadata map[string][]byte, ppHash driver2.PPHash) error {
	logger.Debugf("adding token request [%s]", txID)
	if w.txn == nil {
		return errors.New("no db transaction in progress")
	}
	if applicationMetadata == nil {
		applicationMetadata = make(map[string][]byte)
	}
	j, err := marshal(applicationMetadata)
	if err != nil {
		return errors.New("error marshaling application metadata")
	}

	query, err := NewInsertInto(w.table.Requests).Rows("tx_id, request, status, status_message, application_metadata, pp_hash").Compile()
	if err != nil {
		return errors.Wrapf(err, "error compiling query")
	}
	logger.Debug(query, txID, fmt.Sprintf("(%d bytes)", len(tr)), len(applicationMetadata), len(ppHash))

	_, err = w.txn.Exec(query, txID, tr, driver.Pending, "", j, ppHash)
	return ttxDBError(err)
}

func (w *AtomicWrite) AddMovement(r *driver.MovementRecord) error {
	logger.Debugf("adding movement record [%s:%s:%s:%d:%s]", r.TxID, r.EnrollmentID, r.TokenType, r.Amount.Int64(), r.Status)
	if w.txn == nil {
		return errors.New("no db transaction in progress")
	}
	if !r.Amount.IsInt64() {
		return errors.New("the database driver does not support larger values than int64")
	}
	amount := r.Amount.Int64()

	id, err := uuid.GenerateUUID()
	if err != nil {
		return errors.Wrapf(err, "error generating uuid")
	}
	now := time.Now().UTC()

	query, err := NewInsertInto(w.table.Movements).Rows("id, tx_id, enrollment_id, token_type, amount, stored_at").Compile()
	if err != nil {
		return errors.Wrapf(err, "error compiling query")
	}
	args := []any{id, r.TxID, r.EnrollmentID, r.TokenType, amount, now}
	logger.Debug(query, args)
	_, err = w.txn.Exec(query, args...)

	return ttxDBError(err)
}

func (w *AtomicWrite) AddValidationRecord(txID string, meta map[string][]byte) error {
	logger.Debugf("adding validation record [%s]", txID)
	if w.txn == nil {
		return errors.New("no db transaction in progress")
	}
	md, err := marshal(meta)
	if err != nil {
		return errors.New("can't marshal metadata")
	}
	now := time.Now().UTC()

	query, err := NewInsertInto(w.table.Validations).Rows("tx_id, metadata, stored_at").Compile()
	if err != nil {
		return errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, txID, len(md), now)

	_, err = w.txn.Exec(query, txID, md, now)
	return ttxDBError(err)
}

func ttxDBError(err error) error {
	if err == nil {
		return nil
	}
	logger.Error(err)
	e := strings.ToLower(err.Error())
	if strings.Contains(e, "foreign key constraint") {

		return driver.ErrTokenRequestDoesNotExist
	}
	return err
}

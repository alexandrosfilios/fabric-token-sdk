/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// Metadata contains the metadata of a Token Request
type Metadata struct {
	TokenService         driver.TokensService
	WalletService        driver.WalletService
	TokenRequestMetadata *driver.TokenRequestMetadata
	Logger               logging.Logger
}

// SpentTokenID returns the token IDs of the tokens that were spent by the Token Request this metadata is associated with.
func (m *Metadata) SpentTokenID() []*token.ID {
	var res []*token.ID
	for _, issue := range m.TokenRequestMetadata.Issues {
		res = append(res, issue.TokenIDs...)
	}
	for _, transfer := range m.TokenRequestMetadata.Transfers {
		res = append(res, transfer.TokenIDs...)
	}
	return res
}

// FilterBy returns a new Metadata containing only the metadata that matches the given enrollment IDs.
// For Issue actions, for each issue:
// - The sender;
// - The returned metadata will contain only the outputs whose owner has the given enrollment IDs.
// For Transfer actions, for each action:
// - The list of token IDs will be empty;
// - The returned metadata will contain only the outputs whose owner has the given enrollment IDs;
// - The senders are included if and only if there is at least one output whose owner has the given enrollment IDs.
// Application metadata is always included
func (m *Metadata) FilterBy(eIDs ...string) (*Metadata, error) {
	res := &Metadata{
		TokenService:         m.TokenService,
		WalletService:        m.WalletService,
		TokenRequestMetadata: &driver.TokenRequestMetadata{},
		Logger:               m.Logger,
	}

	// filter issues
	for _, issue := range m.TokenRequestMetadata.Issues {
		issueRes := driver.IssueMetadata{
			Issuer:       issue.Issuer,
			TokenIDs:     issue.TokenIDs,
			ExtraSigners: issue.ExtraSigners,
		}

		for i, receivedAuditInfo := range issue.ReceiversAuditInfos {
			// If the receiver has the given enrollment ID, add it
			recipientEID, err := m.WalletService.GetEnrollmentID(issue.Receivers[i], receivedAuditInfo)
			if err != nil {
				return nil, errors.Wrap(err, "failed getting enrollment ID")
			}
			var outputMetadata []byte
			receivers := issue.Receivers[i]
			var receiverAuditInfos []byte

			if search(eIDs, recipientEID) != -1 {
				outputMetadata = issue.OutputsMetadata[i]
				receiverAuditInfos = issue.ReceiversAuditInfos[i]
			} else {
				m.Logger.Debugf("skipping issue for [%s]", recipientEID)
			}

			issueRes.OutputsMetadata = append(issueRes.OutputsMetadata, outputMetadata)
			issueRes.Receivers = append(issueRes.Receivers, receivers)
			issueRes.ReceiversAuditInfos = append(issueRes.ReceiversAuditInfos, receiverAuditInfos)
		}

		res.TokenRequestMetadata.Issues = append(res.TokenRequestMetadata.Issues, issueRes)
	}

	// filter transfers
	for _, transfer := range m.TokenRequestMetadata.Transfers {
		transferRes := driver.TransferMetadata{
			ExtraSigners: transfer.ExtraSigners,
		}

		// Filter outputs
		// if the receiver has the given enrollment ID, add it. Otherwise, add empty entries
		skip := true
		for i, auditInfo := range transfer.ReceiverAuditInfos {
			recipientEID, err := m.WalletService.GetEnrollmentID(transfer.Receivers[i], auditInfo)
			if err != nil {
				return nil, errors.Wrap(err, "failed getting enrollment ID")
			}
			var outputMetadata []byte
			var outputAuditInfo []byte
			receiver := transfer.Receivers[i]
			var receiverAuditInfos []byte

			if search(eIDs, recipientEID) != -1 {
				m.Logger.Debugf("keeping transfer for [%s]", recipientEID)

				outputMetadata = transfer.OutputsMetadata[i]
				outputAuditInfo = transfer.OutputsAuditInfo[i]

				receiverAuditInfos = transfer.ReceiverAuditInfos[i]
				skip = false
			} else {
				m.Logger.Debugf("skipping transfer for [%s]", recipientEID)
			}

			transferRes.OutputsMetadata = append(transferRes.OutputsMetadata, outputMetadata)
			transferRes.OutputsAuditInfo = append(transferRes.OutputsAuditInfo, outputAuditInfo)

			transferRes.Receivers = append(transferRes.Receivers, receiver)
			transferRes.ReceiverAuditInfos = append(transferRes.ReceiverAuditInfos, receiverAuditInfos)
		}

		// if skip = true, it means that this transfer does not contain any output for the given enrollment IDs.
		// Therefore, no metadata should be given to the passed enrollment IDs.
		// if skip = false, it means that this transfer contains at least one output for the given enrollment IDs.
		// Append the senders to the transfer metadata.
		if !skip {
			for i, sender := range transfer.Senders {
				transferRes.Senders = append(transferRes.Senders, sender)
				transferRes.SenderAuditInfos = append(transferRes.SenderAuditInfos, transfer.SenderAuditInfos[i])
			}
		}

		m.Logger.Debugf("keeping transfer with [%d] out of [%d] outputs", len(transferRes.OutputsMetadata), len(transfer.OutputsMetadata))
		res.TokenRequestMetadata.Transfers = append(res.TokenRequestMetadata.Transfers, transferRes)
	}

	// application
	res.TokenRequestMetadata.Application = m.TokenRequestMetadata.Application

	m.Logger.Debugf("filtered metadata for [% x] from [%d:%d] to [%d:%d]",
		eIDs,
		len(m.TokenRequestMetadata.Issues), len(m.TokenRequestMetadata.Transfers),
		len(res.TokenRequestMetadata.Issues), len(res.TokenRequestMetadata.Transfers))

	return res, nil
}

// Issue returns the i-th issue metadata, if present
func (m *Metadata) Issue(i int) (*IssueMetadata, error) {
	if i >= len(m.TokenRequestMetadata.Issues) {
		return nil, errors.Errorf("index [%d] out of range [0:%d]", i, len(m.TokenRequestMetadata.Issues))
	}
	return &IssueMetadata{IssueMetadata: &m.TokenRequestMetadata.Issues[i]}, nil
}

// Transfer returns the i-th transfer metadata, if present
func (m *Metadata) Transfer(i int) (*TransferMetadata, error) {
	if i >= len(m.TokenRequestMetadata.Transfers) {
		return nil, errors.Errorf("index [%d] out of range [0:%d]", i, len(m.TokenRequestMetadata.Transfers))
	}
	return &TransferMetadata{TransferMetadata: &m.TokenRequestMetadata.Transfers[i]}, nil
}

// IssueMetadata contains the metadata of an issue action
type IssueMetadata struct {
	*driver.IssueMetadata
}

// Match returns true if the given action matches this metadata
func (m *IssueMetadata) Match(action *IssueAction) error {
	if action == nil {
		return errors.New("can't match issue metadata to issue action: nil issue action")
	}

	// validate action
	if err := action.Validate(); err != nil {
		return errors.Wrap(err, "failed validating issue action")
	}

	// check inputs
	if len(m.TokenIDs) != action.NumInputs() {
		return errors.Errorf("expected [%d] inputs but got [%d]", len(m.TokenIDs), action.NumInputs())
	}

	// check outputs
	if len(m.OutputsMetadata) != action.NumOutputs() {
		return errors.Errorf("expected [%d] outputs but got [%d]", len(m.OutputsMetadata), action.NumOutputs())
	}
	if len(m.Receivers) != len(m.ReceiversAuditInfos) {
		return errors.Errorf("expected [%d] receiver audit infos but got [%d]", len(m.OutputsMetadata), len(m.ReceiversAuditInfos))
	}
	extraSigner := action.a.ExtraSigners()
	if len(m.ExtraSigners) != len(extraSigner) {
		return errors.Errorf("expected [%d] extra signers but got [%d]", len(extraSigner), len(m.ExtraSigners))
	}
	// check that the extra signers are the same
	for i, signer := range extraSigner {
		if !signer.Equal(m.ExtraSigners[i]) {
			return errors.Errorf("expected extra signer [%s] but got [%s]", signer, m.ExtraSigners[i])
		}
	}
	return nil
}

// IsOutputAbsent returns true if the given output's metadata is absent
func (m *IssueMetadata) IsOutputAbsent(j int) bool {
	return len(m.OutputsMetadata[j]) == 0
}

// TransferMetadata contains the metadata of a transfer action
type TransferMetadata struct {
	*driver.TransferMetadata
}

// Match returns true if the given action matches this metadata
func (m *TransferMetadata) Match(action *TransferAction) error {
	if action == nil {
		return errors.New("can't match transfer metadata to transfer action: nil issue action")
	}

	// validate action
	if err := action.Validate(); err != nil {
		return errors.Wrap(err, "failed validating issue action")
	}

	if len(m.TokenIDs) != 0 && len(m.Senders) != len(m.TokenIDs) {
		return errors.Errorf("expected [%d] token IDs and senders but got [%d]", len(m.TokenIDs), len(m.Senders))
	}
	if len(m.Senders) != len(m.SenderAuditInfos) {
		return errors.Errorf("expected [%d] senders and sender audit infos but got [%d]", len(m.Senders), len(m.SenderAuditInfos))
	}
	if len(m.OutputsMetadata) != action.NumOutputs() {
		return errors.Errorf("expected [%d] outputs but got [%d]", len(m.OutputsMetadata), action.NumOutputs())
	}
	if len(m.OutputsAuditInfo) != action.NumOutputs() {
		return errors.Errorf("expected [%d] output audit info but got [%d]", action.NumOutputs(), len(m.OutputsAuditInfo))
	}
	if len(m.Receivers) != len(m.ReceiverAuditInfos) {
		return errors.Errorf("expected [%d] receiver audit infos but got [%d]", len(m.Receivers), len(m.ReceiverAuditInfos))
	}
	extraSigner := action.a.ExtraSigners()
	if len(m.ExtraSigners) != len(extraSigner) {
		return errors.Errorf("expected [%d] extra signers but got [%d]", len(extraSigner), len(m.ExtraSigners))
	}
	// check that the extra signers are the same
	for i, signer := range extraSigner {
		if !signer.Equal(m.ExtraSigners[i]) {
			return errors.Errorf("expected extra signer [%s] but got [%s]", signer, m.ExtraSigners[i])
		}
	}
	return nil
}

// IsOutputAbsent returns true if the given output's metadata is absent
func (m *TransferMetadata) IsOutputAbsent(j int) bool {
	if j >= len(m.OutputsMetadata) {
		return true
	}
	return len(m.OutputsMetadata[j]) == 0
}

// IsInputAbsent returns true if the given input's metadata is absent
func (m *TransferMetadata) IsInputAbsent(j int) bool {
	if j >= len(m.Senders) {
		return true
	}
	return m.Senders[j].IsNone()
}

func search(s []string, e string) int {
	for i, v := range s {
		if v == e {
			return i
		}
	}
	return -1
}

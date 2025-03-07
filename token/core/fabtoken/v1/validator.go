/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

type ValidateTransferFunc = common.ValidateTransferFunc[*PublicParams, *Output, *TransferAction, *IssueAction, driver.Deserializer]

type ValidateIssueFunc = common.ValidateIssueFunc[*PublicParams, *Output, *TransferAction, *IssueAction, driver.Deserializer]

type ActionDeserializer struct{}

func (a *ActionDeserializer) DeserializeActions(tr *driver.TokenRequest) ([]*IssueAction, []*TransferAction, error) {
	issueActions := make([]*IssueAction, len(tr.Issues))
	for i := 0; i < len(tr.Issues); i++ {
		ia := &IssueAction{}
		if err := ia.Deserialize(tr.Issues[i]); err != nil {
			return nil, nil, err
		}
		issueActions[i] = ia
	}

	transferActions := make([]*TransferAction, len(tr.Transfers))
	for i := 0; i < len(tr.Transfers); i++ {
		ta := &TransferAction{}
		if err := ta.Deserialize(tr.Transfers[i]); err != nil {
			return nil, nil, err
		}
		transferActions[i] = ta
	}

	return issueActions, transferActions, nil
}

type Context = common.Context[*PublicParams, *Output, *TransferAction, *IssueAction, driver.Deserializer]

type Validator = common.Validator[*PublicParams, *Output, *TransferAction, *IssueAction, driver.Deserializer]

func NewValidator(logger logging.Logger, pp *PublicParams, deserializer driver.Deserializer, extraValidators ...ValidateTransferFunc) *Validator {
	transferValidators := []ValidateTransferFunc{
		TransferActionValidate,
		TransferSignatureValidate,
		TransferBalanceValidate,
		TransferHTLCValidate,
	}
	transferValidators = append(transferValidators, extraValidators...)

	issueValidators := []ValidateIssueFunc{
		IssueValidate,
	}

	return common.NewValidator[*PublicParams, *Output, *TransferAction, *IssueAction, driver.Deserializer](
		logger,
		pp,
		deserializer,
		&ActionDeserializer{},
		transferValidators,
		issueValidators,
	)
}

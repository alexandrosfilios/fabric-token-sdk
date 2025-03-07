/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package audit

import (
	"context"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	htlc2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

//go:generate counterfeiter -o mock/signing_identity.go -fake-name SigningIdentity . SigningIdentity

// SigningIdentity models a signing identity
type SigningIdentity interface {
	driver.SigningIdentity
}

// Deserializer deserialize audit information
type Deserializer interface {
	// GetOwnerMatcher returns the owner matcher for the given audit info
	GetOwnerMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error)
}

// InspectTokenOwnerFunc models a function to inspect the owner field
type InspectTokenOwnerFunc = func(des Deserializer, token *AuditableToken, index int) error

// GetAuditInfoForIssuesFunc models a function to get auditable tokens from issue actions
type GetAuditInfoForIssuesFunc = func(issues [][]byte, metadata []*driver.IssueMetadata) ([][]*AuditableToken, error)

// GetAuditInfoForTransfersFunc models a function to get auditable tokens from transfer actions
type GetAuditInfoForTransfersFunc = func(transfers [][]byte, metadata []*driver.TransferMetadata, inputs [][]*token.Token) ([][]*AuditableToken, [][]*AuditableToken, error)

// AuditableToken contains a zkat token and the information that allows
// an auditor to learn its content.
type AuditableToken struct {
	Token *token.Token
	Data  *TokenDataOpening
	Owner *OwnerOpening
}

func NewAuditableToken(token *token.Token, ownerInfo []byte, tokenType token2.Type, value *math.Zr, bf *math.Zr) (*AuditableToken, error) {
	return &AuditableToken{
		Token: token,
		Owner: &OwnerOpening{
			OwnerInfo: ownerInfo,
		},
		Data: &TokenDataOpening{
			TokenType: tokenType,
			Value:     value,
			BF:        bf,
		},
	}, nil
}

// TokenDataOpening contains the opening of the TokenData.
// TokenData is a Pedersen commitment to token type and Value.
type TokenDataOpening struct {
	TokenType token2.Type
	Value     *math.Zr
	BF        *math.Zr
}

// OwnerOpening contains the information that allows the auditor to identify the owner.
type OwnerOpening struct {
	OwnerInfo []byte
}

// Auditor inspects zkat tokens and their owners.
type Auditor struct {
	Logger logging.Logger
	tracer trace.Tracer
	// Owner Identity Deserializer
	Des Deserializer
	// Auditor's signing identity
	Signer SigningIdentity
	// Pedersen generators used to compute TokenData
	PedersenParams []*math.G1
	// Elliptic curve
	Curve *math.Curve

	// InspectTokenOwnerFunc is a function that inspects the owner field
	InspectTokenOwnerFunc        InspectTokenOwnerFunc
	GetAuditInfoForIssuesFunc    GetAuditInfoForIssuesFunc
	GetAuditInfoForTransfersFunc GetAuditInfoForTransfersFunc
}

func NewAuditor(logger logging.Logger, tracer trace.Tracer, des Deserializer, pp []*math.G1, signer SigningIdentity, c *math.Curve) *Auditor {
	a := &Auditor{
		Logger:         logger,
		tracer:         tracer,
		Des:            des,
		PedersenParams: pp,
		Signer:         signer,
		Curve:          c,
	}
	a.InspectTokenOwnerFunc = InspectTokenOwner
	a.GetAuditInfoForIssuesFunc = GetAuditInfoForIssues
	a.GetAuditInfoForTransfersFunc = GetAuditInfoForTransfers
	return a
}

// Endorse is called to sign a valid token request
func (a *Auditor) Endorse(tokenRequest *driver.TokenRequest, txID string) ([]byte, error) {
	if tokenRequest == nil {
		return nil, errors.Errorf("audit of tx [%s] failed: : token request is nil", txID)
	}
	// Marshal tokenRequest
	bytes, err := tokenRequest.MarshalToMessageToSign([]byte(txID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling token request [%s]", txID)
	}
	// Sign
	a.Logger.Debugf("Endorse [%s][%s]", hash.Hashable(bytes).String(), txID)
	if a.Signer == nil {
		return nil, errors.Errorf("audit of tx [%s] failed: signer is nil", txID)
	}
	return a.Signer.Sign(bytes)
}

// Check validates TokenRequest against TokenRequestMetadata
func (a *Auditor) Check(
	ctx context.Context,
	tokenRequest *driver.TokenRequest,
	tokenRequestMetadata *driver.TokenRequestMetadata,
	inputTokens [][]*token.Token,
	txID string,
) error {
	// TODO: inputTokens should be checked against the actions
	span := trace.SpanFromContext(ctx)
	span.AddEvent("start_check")
	defer span.AddEvent("end_check")
	// De-obfuscate issue requests
	span.AddEvent("get_issue_audit_info")
	outputsFromIssue, err := a.GetAuditInfoForIssuesFunc(tokenRequest.Issues, tokenRequestMetadata.Issues)
	if err != nil {
		return errors.Wrapf(err, "failed getting audit info for issues for [%s]", txID)
	}
	// check validity of issue requests
	span.AddEvent("check_issue_requests")
	err = a.CheckIssueRequests(outputsFromIssue, txID)
	if err != nil {
		return errors.Wrapf(err, "failed checking issues for [%s]", txID)
	}
	// De-obfuscate transfer requests
	span.AddEvent("get_transfer_audit_info")
	auditableInputs, outputsFromTransfer, err := a.GetAuditInfoForTransfersFunc(tokenRequest.Transfers, tokenRequestMetadata.Transfers, inputTokens)
	if err != nil {
		return errors.Wrapf(err, "failed getting audit info for transfers for [%s]", txID)
	}
	// check validity of transfer requests
	span.AddEvent("check_transfer_requests")
	if err := a.CheckTransferRequests(auditableInputs, outputsFromTransfer, txID); err != nil {
		return errors.Wrapf(err, "failed checking transfers [%s]", txID)
	}

	return nil
}

// CheckTransferRequests verifies that the commitments in transfer inputs and outputs match the information provided in the clear.
func (a *Auditor) CheckTransferRequests(inputs [][]*AuditableToken, outputsFromTransfer [][]*AuditableToken, txID string) error {

	for k, transferred := range outputsFromTransfer {
		err := a.InspectOutputs(transferred)
		if err != nil {
			return errors.Wrapf(err, "audit of %d th transfer in tx [%s] failed", k, txID)
		}
	}

	for k, i := range inputs {
		err := a.InspectInputs(i)
		if err != nil {
			return errors.Wrapf(err, "audit of %d th transfer in tx [%s] failed", k, txID)
		}
	}

	return nil
}

// CheckIssueRequests verifies that the commitments in issue outputs match the information provided in the clear.
func (a *Auditor) CheckIssueRequests(outputsFromIssue [][]*AuditableToken, txID string) error {
	// Inspect
	for k, issued := range outputsFromIssue {
		err := a.InspectOutputs(issued)
		if err != nil {
			return errors.Wrapf(err, "audit of %d th issue in tx [%s] failed", k, txID)
		}
	}
	return nil
}

// InspectOutputs verifies that the commitments in an array of outputs matches the information provided in the clear.
func (a *Auditor) InspectOutputs(tokens []*AuditableToken) error {
	for i, t := range tokens {
		err := a.InspectOutput(t, i)
		if err != nil {
			return errors.Wrapf(err, "failed inspecting output [%d]", i)
		}
	}

	return nil
}

// InspectOutput verifies that the commitments in an output token of a given index
// match the information provided in the clear.
func (a *Auditor) InspectOutput(output *AuditableToken, index int) error {
	if len(a.PedersenParams) != 3 {
		return errors.Errorf("length of Pedersen basis != 3")
	}
	if output == nil || output.Data == nil {
		return errors.Errorf("invalid output at index [%d]", index)
	}
	tokenComm := commit([]*math.Zr{a.Curve.HashToZr([]byte(output.Data.TokenType)), output.Data.Value, output.Data.BF}, a.PedersenParams, a.Curve)
	if output.Token == nil || output.Token.Data == nil {
		return errors.Errorf("invalid output at index [%d]", index)
	}
	if !tokenComm.Equals(output.Token.Data) {
		return errors.Errorf("output at index [%d] does not match the provided opening", index)
	}

	if !output.Token.IsRedeem() { // this is not a redeemed output
		if err := a.InspectTokenOwnerFunc(a.Des, output, index); err != nil {
			return errors.Wrapf(err, "failed inspecting output at index [%d]", index)
		}
	}

	return nil
}

// InspectInputs verifies that the commitments in an array of inputs matches the information provided in the clear.
func (a *Auditor) InspectInputs(inputs []*AuditableToken) error {
	for i, input := range inputs {
		if input == nil || input.Token == nil {
			return errors.Errorf("invalid input at index [%d]", i)
		}

		if !input.Token.IsRedeem() {
			if err := a.InspectTokenOwnerFunc(a.Des, input, i); err != nil {
				return errors.Wrapf(err, "failed inspecting input at index [%d]", i)
			}
		}
	}
	return nil
}

// InspectTokenOwner verifies that the audit info matches the token owner
func InspectTokenOwner(des Deserializer, token *AuditableToken, index int) error {
	if token.Token.IsRedeem() {
		return errors.Errorf("token at index [%d] is a redeem token, cannot inspect ownership", index)
	}
	if len(token.Owner.OwnerInfo) == 0 {
		return errors.Errorf("failed to inspect owner at index [%d]: owner info is nil", index)
	}
	ro, err := identity.UnmarshalTypedIdentity(token.Token.GetOwner())
	if err != nil {
		return errors.Errorf("owner at index [%d] cannot be unwrapped", index)
	}
	switch ro.Type {
	case x509.IdentityType:
		matcher, err := des.GetOwnerMatcher(token.Token.Owner, token.Owner.OwnerInfo)
		if err != nil {
			return errors.Wrapf(err, "failed to get owner matcher for output [%d]", index)
		}
		if err := matcher.Match(ro.Identity); err != nil {
			return errors.Wrapf(err, "owner at index [%d] does not match the provided opening", index)
		}
		return nil
	case msp.IdemixIdentity:
		matcher, err := des.GetOwnerMatcher(token.Token.Owner, token.Owner.OwnerInfo)
		if err != nil {
			return errors.Wrapf(err, "failed to get owner matcher for output [%d]", index)
		}
		if err := matcher.Match(ro.Identity); err != nil {
			return errors.Wrapf(err, "owner at index [%d] does not match the provided opening", index)
		}
		return nil
	case htlc2.ScriptType:
		return inspectTokenOwnerOfScript(des, token, index)
	default:
		return errors.Errorf("identity type [%s] not recognized", ro.Type)
	}
}

func inspectTokenOwnerOfScript(des Deserializer, token *AuditableToken, index int) error {
	owner, err := identity.UnmarshalTypedIdentity(token.Token.GetOwner())
	if err != nil {
		return errors.Errorf("input owner at index [%d] cannot be unmarshalled", index)
	}
	scriptInf := &htlc.ScriptInfo{}
	if err := json.Unmarshal(token.Owner.OwnerInfo, scriptInf); err != nil {
		return errors.Wrapf(err, "failed to unmarshal script info")
	}
	scriptSender, scriptRecipient, err := htlc.GetScriptSenderAndRecipient(owner)
	if err != nil {
		return errors.Wrap(err, "failed getting script sender and recipient")
	}

	sender, err := des.GetOwnerMatcher(scriptSender, scriptInf.Sender)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal audit info from script sender [%s]", string(scriptInf.Sender))
	}
	ro, err := identity.UnmarshalTypedIdentity(scriptSender)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve raw owner from sender in script")
	}
	if err := sender.Match(ro.Identity); err != nil {
		return errors.Wrapf(err, "token at index [%d] does not match the provided opening [%s]", index, string(scriptInf.Sender))
	}

	recipient, err := des.GetOwnerMatcher(scriptRecipient, scriptInf.Recipient)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal audit info from script recipient [%s]", string(scriptInf.Recipient))
	}
	ro, err = identity.UnmarshalTypedIdentity(scriptRecipient)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve raw owner from recipien in script")
	}
	if err := recipient.Match(ro.Identity); err != nil {
		return errors.Wrapf(err, "token at index [%d] does not match the provided opening [%s]", index, string(scriptInf.Recipient))
	}

	return nil
}

// GetAuditInfoForIssues returns an array of AuditableToken for each issue action
// It takes a deserializer, an array of serialized issue actions and an array of issue metadata.
func GetAuditInfoForIssues(issues [][]byte, issueMetadata []*driver.IssueMetadata) ([][]*AuditableToken, error) {
	if len(issues) != len(issueMetadata) {
		return nil, errors.Errorf("number of issues does not match number of provided metadata")
	}
	outputs := make([][]*AuditableToken, len(issues))
	for k, md := range issueMetadata {
		ia := &issue.Action{}
		err := ia.Deserialize(issues[k])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to deserialize issue action at index [%d]", k)
		}

		if len(ia.Outputs) != len(md.Outputs) {
			return nil, errors.Errorf("number of output does not match number of provided metadata")
		}
		outputs[k] = make([]*AuditableToken, len(md.Outputs))
		for i, o := range md.Outputs {
			if o == nil {
				return nil, errors.Errorf("output at index [%d] is nil", i)
			}
			metadata := &token.Metadata{}
			err = metadata.Deserialize(o.OutputMetadata)
			if err != nil {
				return nil, err
			}
			if ia.Outputs[i] == nil {
				return nil, errors.Errorf("output token at index [%d] is nil", i)
			}
			if ia.Outputs[i].IsRedeem() {
				return nil, errors.Errorf("issue cannot redeem tokens")
			}
			if len(o.Receivers) == 0 {
				return nil, errors.Errorf("issue must have at least one receiver")
			}
			// TODO: check that o.Receivers contains not-nil elements
			outputs[k][i], err = NewAuditableToken(
				ia.Outputs[i],
				o.Receivers[0].AuditInfo,
				metadata.Type,
				metadata.Value,
				metadata.BlindingFactor,
			)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to create auditable token at index [%d]", i)
			}
		}
	}
	return outputs, nil
}

// GetAuditInfoForTransfers returns an array of AuditableToken for each transfer action.
// It takes a deserializer, an array of serialized transfer actions and an array of transfer metadata.
func GetAuditInfoForTransfers(transfers [][]byte, metadata []*driver.TransferMetadata, inputs [][]*token.Token) ([][]*AuditableToken, [][]*AuditableToken, error) {
	if len(transfers) != len(metadata) {
		return nil, nil, errors.Errorf("number of transfers does not match the number of provided metadata")
	}
	if len(inputs) != len(metadata) {
		return nil, nil, errors.Errorf("number of inputs does not match the number of provided metadata")
	}
	auditableInputs := make([][]*AuditableToken, len(inputs))
	outputs := make([][]*AuditableToken, len(transfers))
	for k, transferMetadata := range metadata {
		if len(transferMetadata.Inputs) != len(inputs[k]) {
			return nil, nil, errors.Errorf("number of inputs does not match the number of senders [%d]!=[%d]", len(transferMetadata.Inputs), len(inputs[k]))
		}
		auditableInputs[k] = make([]*AuditableToken, len(transferMetadata.Inputs))
		for i := 0; i < len(transferMetadata.Inputs); i++ {
			var err error
			if inputs[k][i] == nil {
				return nil, nil, errors.Errorf("input[%d][%d] is nil", k, i)
			}
			auditableInputs[k][i], err = NewAuditableToken(inputs[k][i], transferMetadata.Inputs[i].Senders[0].AuditInfo, "", nil, nil)
			if err != nil {
				return nil, nil, err
			}
		}
		ta := &transfer.Action{}
		err := ta.Deserialize(transfers[k])
		if err != nil {
			return nil, nil, err
		}
		if len(ta.Outputs) != len(transferMetadata.Outputs) {
			return nil, nil, errors.Errorf("number of outputs does not match the number of output metadata [%d]!=[%d]", len(ta.Outputs), len(transferMetadata.Outputs))
		}
		outputs[k] = make([]*AuditableToken, len(ta.Outputs))
		for i := 0; i < len(ta.Outputs); i++ {
			if ta.Outputs[i] == nil {
				return nil, nil, errors.Errorf("output token at index [%d] is nil", i)
			}

			ti := &token.Metadata{}
			err = ti.Deserialize(transferMetadata.Outputs[i].OutputMetadata)
			if err != nil {
				return nil, nil, err
			}
			// TODO: we need to check also how many recipients the output contains, and check them all in isolation and compatibility
			outputs[k][i], err = NewAuditableToken(ta.Outputs[i], transferMetadata.Outputs[i].OutputAuditInfo, ti.Type, ti.Value, ti.BlindingFactor)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	return auditableInputs, outputs, nil
}

func commit(vector []*math.Zr, generators []*math.G1, c *math.Curve) *math.G1 {
	com := c.NewG1()
	for i := range vector {
		com.Add(generators[i].Mul(vector[i]))
	}
	return com
}

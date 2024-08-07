/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/common"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/config"
	idemix2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/msp"
	x5092 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509"
	"github.com/pkg/errors"
)

const (
	// OwnerMSPID is the default MSP ID for the owner wallet
	OwnerMSPID = "OwnerMSPID"
	// IssuerMSPID is the default MSP ID for the issuer wallet
	IssuerMSPID = "IssuerMSPID"
	// AuditorMSPID is the default MSP ID for the auditor wallet
	AuditorMSPID = "AuditorMSPID"
	// CertifierMSPID is the default MSP ID for the certifier wallet
	CertifierMSPID = "CertifierMSPID"
)

// RoleToMSPID maps the role to the MSP ID
var RoleToMSPID = map[driver.IdentityRole]string{
	driver.OwnerRole:     OwnerMSPID,
	driver.IssuerRole:    IssuerMSPID,
	driver.AuditorRole:   AuditorMSPID,
	driver.CertifierRole: CertifierMSPID,
}

// RoleFactory is the factory for creating wallets, idemix and x509
type RoleFactory struct {
	TMSID                  token.TMSID
	Config                 config2.Config
	FSCIdentity            driver.Identity
	NetworkDefaultIdentity driver.Identity
	IdentityProvider       common.IdentityProvider
	SignerService          common.SigService
	BinderService          common.BinderService
	StorageProvider        identity.StorageProvider
	DeserializerManager    driver2.DeserializerManager
	ignoreRemote           bool
}

// NewRoleFactory creates a new RoleFactory
func NewRoleFactory(
	TMSID token.TMSID,
	config config2.Config,
	fscIdentity driver.Identity,
	networkDefaultIdentity driver.Identity,
	identityProvider common.IdentityProvider,
	signerService common.SigService,
	binderService common.BinderService,
	storageProvider identity.StorageProvider,
	deserializerManager driver2.DeserializerManager,
	ignoreRemote bool,
) *RoleFactory {
	return &RoleFactory{
		TMSID:                  TMSID,
		Config:                 config,
		FSCIdentity:            fscIdentity,
		NetworkDefaultIdentity: networkDefaultIdentity,
		IdentityProvider:       identityProvider,
		SignerService:          signerService,
		BinderService:          binderService,
		StorageProvider:        storageProvider,
		DeserializerManager:    deserializerManager,
		ignoreRemote:           ignoreRemote,
	}
}

// NewIdemix creates a new Idemix-based role
func (f *RoleFactory) NewIdemix(role driver.IdentityRole, cacheSize int, issuerPublicKey []byte, curveID math3.CurveID) (identity.Role, error) {
	identities, err := f.IdentitiesForRole(role)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identities for role [%d]", role)
	}

	identityDB, err := f.StorageProvider.OpenIdentityDB(f.TMSID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallet path storage")
	}
	backend, err := f.StorageProvider.NewKeystore()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get new keystore backend")
	}
	keyStore, err := msp.NewKeyStore(curveID, backend)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to instantiate bccsp key store")
	}
	lm := idemix2.NewLocalMembership(
		issuerPublicKey,
		curveID,
		f.Config,
		f.NetworkDefaultIdentity,
		f.SignerService,
		f.DeserializerManager,
		identityDB,
		keyStore,
		RoleToMSPID[role],
		cacheSize,
		identities,
		f.ignoreRemote,
	)
	if err := lm.Load(); err != nil {
		return nil, errors.WithMessage(err, "failed to load identities")
	}
	return &BindingRole{
		Role:             idemix2.NewRole(role, f.TMSID.Network, f.FSCIdentity, lm),
		IdentityType:     IdemixIdentity,
		RootIdentity:     f.FSCIdentity,
		IdentityProvider: f.IdentityProvider,
		BinderService:    f.BinderService,
	}, nil
}

// NewX509 creates a new X509-based role
func (f *RoleFactory) NewX509(role driver.IdentityRole) (identity.Role, error) {
	return f.newX509WithType(role, "", false)
}

func (f *RoleFactory) NewWrappedX509(role driver.IdentityRole, ignoreRemote bool) (identity.Role, error) {
	return f.newX509WithType(role, X509Identity, ignoreRemote)
}

func (f *RoleFactory) newX509WithType(role driver.IdentityRole, identityType string, ignoreRemote bool) (identity.Role, error) {
	identities, err := f.IdentitiesForRole(role)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identities for role [%d]", role)
	}
	identityDB, err := f.StorageProvider.OpenIdentityDB(f.TMSID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallet path storage")
	}
	lm := x5092.NewLocalMembership(
		f.Config,
		f.NetworkDefaultIdentity,
		f.SignerService,
		f.BinderService,
		f.DeserializerManager,
		identityDB,
		RoleToMSPID[role],
		ignoreRemote,
	)
	if err := lm.Load(identities); err != nil {
		return nil, errors.WithMessage(err, "failed to load identities")
	}
	return &BindingRole{
		Role:             x5092.NewRole(role, f.TMSID.Network, f.FSCIdentity, lm),
		IdentityType:     identityType,
		RootIdentity:     f.FSCIdentity,
		IdentityProvider: f.IdentityProvider,
		BinderService:    f.BinderService,
	}, nil
}

// IdentitiesForRole returns the configured identities for the passed role
func (f *RoleFactory) IdentitiesForRole(role driver.IdentityRole) ([]*config.Identity, error) {
	return f.Config.IdentitiesForRole(role)
}

type BindingRole struct {
	identity.Role
	IdentityType identity.Type

	RootIdentity     driver.Identity
	IdentityProvider common.IdentityProvider
	BinderService    common.BinderService
}

func (r *BindingRole) GetIdentityInfo(id string) (driver.IdentityInfo, error) {
	info, err := r.Role.GetIdentityInfo(id)
	if err != nil {
		return nil, err
	}
	return &Info{
		IdentityInfo:     info,
		IdentityType:     r.IdentityType,
		RootIdentity:     r.RootIdentity,
		IdentityProvider: r.IdentityProvider,
		BinderService:    r.BinderService,
	}, nil
}

// Info wraps a driver.IdentityInfo to further register the audit info,
// and binds the new identity to the default FSC node identity
type Info struct {
	driver.IdentityInfo
	IdentityType identity.Type

	RootIdentity     driver.Identity
	IdentityProvider common.IdentityProvider
	BinderService    common.BinderService
}

func (i *Info) ID() string {
	return i.IdentityInfo.ID()
}

func (i *Info) EnrollmentID() string {
	return i.IdentityInfo.EnrollmentID()
}

func (i *Info) Get() (driver.Identity, []byte, error) {
	// get the identity
	id, ai, err := i.IdentityInfo.Get()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get root identity")
	}
	// register the audit info
	if err := i.IdentityProvider.RegisterAuditInfo(id, ai); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to register audit info for identity [%s]", id)
	}
	// bind the identity to the default FSC node identity
	if i.BinderService != nil {
		if err := i.BinderService.Bind(i.RootIdentity, id, false); err != nil {
			return nil, nil, errors.Wrapf(err, "failed to bind identity [%s] to [%s]", id, i.RootIdentity)
		}
	}
	// wrap the backend identity, and bind it
	if len(i.IdentityType) != 0 {
		typedIdentity, err := identity.WrapWithType(i.IdentityType, id)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to wrap identity [%s]", i.IdentityType)
		}
		if i.BinderService != nil {
			if err := i.BinderService.Bind(id, typedIdentity, true); err != nil {
				return nil, nil, errors.Wrapf(err, "failed to bind identity [%s] to [%s]", typedIdentity, id)
			}
			if err := i.BinderService.Bind(i.RootIdentity, typedIdentity, false); err != nil {
				return nil, nil, errors.Wrapf(err, "failed to bind identity [%s] to [%s]", typedIdentity, i.RootIdentity)
			}
		} else {
			// register at the list the audit info
			if err := i.IdentityProvider.RegisterAuditInfo(typedIdentity, ai); err != nil {
				return nil, nil, errors.Wrapf(err, "failed to register audit info for identity [%s]", id)
			}
		}
		id = typedIdentity
	}
	return id, ai, nil
}

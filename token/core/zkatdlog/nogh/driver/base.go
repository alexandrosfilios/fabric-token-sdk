/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/membership"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix"
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/role"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/wallet"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

type base struct{}

func (d *base) PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
	pp, err := crypto.NewPublicParamsFromBytes(params, crypto.DLogPublicParameters)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal public parameters")
	}
	return pp, nil
}

func (d *base) DefaultValidator(pp driver.PublicParameters) (driver.Validator, error) {
	deserializer, err := NewDeserializer(pp.(*crypto.PublicParams))
	if err != nil {
		return nil, errors.Errorf("failed to create token service deserializer: %v", err)
	}
	logger := logging.DriverLoggerFromPP("token-sdk.driver.zkatdlog", pp.Identifier())
	return validator.New(logger, pp.(*crypto.PublicParams), deserializer), nil
}

func (d *base) newWalletService(
	tmsConfig driver.Config,
	binder idriver.NetworkBinderService,
	storageProvider identity.StorageProvider,
	qe driver.QueryEngine,
	logger logging.Logger,
	fscIdentity view.Identity,
	networkDefaultIdentity view.Identity,
	publicParams driver.PublicParameters,
	ignoreRemote bool,
) (*wallet.Service, error) {
	pp := publicParams.(*crypto.PublicParams)
	roles := wallet.NewRoles()
	deserializerManager := sig.NewMultiplexDeserializer()
	tmsID := tmsConfig.ID()
	identityDB, err := storageProvider.OpenIdentityDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open identity db for tms [%s]", tmsID)
	}
	sigService := sig.NewService(deserializerManager, identityDB)
	ip := identity.NewProvider(logger.Named("identity"), identityDB, sigService, binder, NewEIDRHDeserializer())
	identityConfig, err := config.NewIdentityConfig(tmsConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create identity config")
	}

	// Prepare roles
	roleFactory := role.NewFactory(logger, tmsID, identityConfig, fscIdentity, networkDefaultIdentity, ip, ip, ip, storageProvider, deserializerManager)
	// owner role
	// we have one key manager for fabtoken and one for each idemix issuer public key
	kmps := make([]membership.KeyManagerProvider, 0, len(pp.IdemixIssuerPublicKeys)+1)
	for _, key := range pp.IdemixIssuerPublicKeys {
		backend, err := storageProvider.NewKeystore()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get new keystore backend")
		}
		keyStore, err := msp2.NewKeyStore(key.Curve, backend)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to instantiate bccsp key store")
		}
		kmp := idemix.NewKeyManagerProvider(
			key.PublicKey,
			key.Curve,
			msp.RoleToMSPID[identity.OwnerRole],
			keyStore,
			sigService,
			identityConfig,
			identityConfig.DefaultCacheSize(),
			ignoreRemote,
		)
		kmps = append(kmps, kmp)
	}
	kmps = append(kmps, x509.NewKeyManagerProvider(identityConfig, msp.RoleToMSPID[identity.OwnerRole], ip, ignoreRemote))

	role, err := roleFactory.NewRole(identity.OwnerRole, true, nil, kmps...)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create owner role")
	}
	roles.Register(identity.OwnerRole, role)
	role, err = roleFactory.NewRole(identity.IssuerRole, false, pp.Issuers(), x509.NewKeyManagerProvider(identityConfig, msp.RoleToMSPID[identity.IssuerRole], ip, ignoreRemote))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create issuer role")
	}
	roles.Register(identity.IssuerRole, role)
	role, err = roleFactory.NewRole(identity.AuditorRole, false, pp.Auditors(), x509.NewKeyManagerProvider(identityConfig, msp.RoleToMSPID[identity.AuditorRole], ip, ignoreRemote))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create auditor role")
	}
	roles.Register(identity.AuditorRole, role)
	role, err = roleFactory.NewRole(identity.CertifierRole, false, nil, x509.NewKeyManagerProvider(identityConfig, msp.RoleToMSPID[identity.CertifierRole], ip, ignoreRemote))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create certifier role")
	}
	roles.Register(identity.CertifierRole, role)

	// wallet service
	walletDB, err := storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	deserializer, err := NewDeserializer(pp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to instantiate the deserializer")
	}
	return wallet.NewService(
		logger,
		ip,
		deserializer,
		wallet.NewFactory(logger, ip, qe, identityConfig, deserializer),
		roles.ToWalletRegistries(logger, walletDB),
	), nil
}

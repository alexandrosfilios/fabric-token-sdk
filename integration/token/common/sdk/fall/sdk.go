/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fall

import (
	"errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	auditdb "github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb/db/sql"
	identitydb "github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	tokendb "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
	tokenlockdb "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb/db/sql"
	ttxdb "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
	"go.uber.org/dig"
)

type SDK struct {
	dig2.SDK
}

func NewSDK(registry node.Registry) *SDK {
	return &SDK{SDK: tokensdk.NewSDK(registry)}
}

func NewFrom(sdk dig2.SDK) *SDK {
	return &SDK{SDK: sdk}
}

func (p *SDK) Install() error {
	err := errors.Join(
		p.Container().Provide(fabric.NewDriver, dig.Group("network-drivers")),
		p.Container().Provide(tokenlockdb.NewDriver, dig.Group("tokenlockdb-drivers")),
		p.Container().Provide(auditdb.NewDriver, dig.Group("auditdb-drivers")),
		p.Container().Provide(tokendb.NewDriver, dig.Group("tokendb-drivers")),
		p.Container().Provide(ttxdb.NewDriver, dig.Group("ttxdb-drivers")),
		p.Container().Provide(identitydb.NewDriver, dig.Group("identitydb-drivers")),
		p.Container().Provide(tokensdk.NewDBDrivers),
		p.Container().Provide(fabtoken.NewDriver, dig.Group("token-drivers")),
		p.Container().Provide(dlog.NewDriver, dig.Group("token-drivers")),
	)
	if err != nil {
		return err
	}

	return p.SDK.Install()
}

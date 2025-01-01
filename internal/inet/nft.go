package inet

/*
* Equivialent of the following command line.
*
* nft flush ruleset
* nft add table inet filter
* nft add chain inet filter input '{type filter hook input priority 0; policy accept; }'
* nft add chain inet filter forward '{type filter hook forward priority 0; policy accept; }'
* nft add chain inet filter output '{type filter hook output priority 0; policy accept; }'
*
* nft add table inet nat
* nft add chain inet nat prerouting '{ type nat hook prerouting priority -100; }'
* nft add chain inet nat postrouting '{ type nat hook postrouting priority 100; }'
* nft add rule inet nat postrouting oifname "ens33" masquerade
* nft add rule inet nat postrouting iifname "ens33" masquerade
 */

import (
	"log/slog"
	"os"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
)

type NftUtil struct {
	linkName    string // TODO: Support more than one interface.
	nat         *nftables.Table
	prerouting  *nftables.Chain
	postrouting *nftables.Chain

	filter        *nftables.Table
	filterInput   *nftables.Chain
	filterOutput  *nftables.Chain
	filterForward *nftables.Chain
}

func NewNftUtil(linkName string) *NftUtil {

	nfu := NftUtil{
		linkName: linkName,
	}
	return &nfu
}

/*
* Force fit for C IFNAMSIZ
 */
func (nfu *NftUtil) ifname(n string) []byte {
	b := make([]byte, 16)
	copy(b, []byte(n+"\x00"))
	return b
}

/*
* Apply our forwarding rules.
 */
func (nfu *NftUtil) EnableForwarding() {

	c, err := nftables.New(nftables.AsLasting())
	if err != nil {
		slog.Error("failed opening nftables", "error", err)
		os.Exit(1)
	}
	c.FlushRuleset()

	nfu.nat = c.AddTable(&nftables.Table{
		Family: nftables.TableFamilyINet,
		Name:   "nat",
	})

	nfu.prerouting = c.AddChain(&nftables.Chain{
		Name:     "prerouting",
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriorityFilter,
		Table:    nfu.nat,
		Type:     nftables.ChainTypeNAT,
	})

	nfu.postrouting = c.AddChain(&nftables.Chain{
		Name:     "postrouting",
		Hooknum:  nftables.ChainHookPostrouting,
		Priority: nftables.ChainPriorityNATSource,
		Table:    nfu.nat,
		Type:     nftables.ChainTypeNAT,
	})

	// TODO: Upgrade for multiple interfaces rather than just one.
	c.AddRule(&nftables.Rule{
		Table: nfu.nat,
		Chain: nfu.postrouting,
		Exprs: []expr.Any{
			// meta load oifname => reg 1
			&expr.Meta{Key: expr.MetaKeyOIFNAME, Register: 1},
			// cmp eq reg 1 0x696c7075 0x00306b6e 0x00000000 0x00000000
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     nfu.ifname(nfu.linkName),
			},
			// masq
			&expr.Masq{},
		},
	})

	c.AddRule(&nftables.Rule{
		Table: nfu.nat,
		Chain: nfu.postrouting,
		Exprs: []expr.Any{
			// meta load iifname => reg 1
			&expr.Meta{Key: expr.MetaKeyIIFNAME, Register: 1},
			// cmp eq reg 1 0x696c7075 0x00306b6e 0x00000000 0x00000000
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     nfu.ifname(nfu.linkName),
			},
			// masq
			&expr.Masq{},
		},
	})

	nfu.filter = c.AddTable(&nftables.Table{
		Family: nftables.TableFamilyINet,
		Name:   "filter",
	})

	nfu.filterInput = c.AddChain(&nftables.Chain{
		Name:     "input",
		Hooknum:  nftables.ChainHookInput,
		Priority: nftables.ChainPriorityFilter,
		Table:    nfu.filter,
		Type:     nftables.ChainTypeFilter,
	})

	nfu.filterForward = c.AddChain(&nftables.Chain{
		Name:     "forward",
		Hooknum:  nftables.ChainHookForward,
		Priority: nftables.ChainPriorityFilter,
		Table:    nfu.filter,
		Type:     nftables.ChainTypeFilter,
	})

	nfu.filterOutput = c.AddChain(&nftables.Chain{
		Name:     "output",
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityFilter,
		Table:    nfu.filter,
		Type:     nftables.ChainTypeFilter,
	})

	err = c.Flush()
	if err != nil {
		slog.Error("failed flushing")
	}
	err = c.CloseLasting()
	if err != nil {
		slog.Error("failed CloseLasting")
	}
}

/*
* Remove tables and null out object.
 */
func (nfu *NftUtil) DisableForwarding() {

	c, err := nftables.New(nftables.AsLasting())
	if err != nil {
		slog.Error("failed opening nftables", "error", err)
		os.Exit(1)
	}

	c.DelTable(nfu.nat)
	c.DelTable(nfu.filter)

	nfu.nat = nil
	nfu.prerouting = nil
	nfu.postrouting = nil

	nfu.filter = nil
	nfu.filterInput = nil
	nfu.filterOutput = nil
	nfu.filterForward = nil
}

package p2p

import (
	"context"
	"fmt"
	"time"

	datastore "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	host "github.com/libp2p/go-libp2p/core/host"
	peer "github.com/libp2p/go-libp2p/core/peer"
	routing "github.com/libp2p/go-libp2p/core/routing"

	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	routingdiscovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

// ----------------- mDNS (LAN) -----------------

type mdnsNotifee struct {
	h host.Host
}

func (n *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == n.h.ID() {
		return
	}
	fmt.Println("[mDNS] Peer found:", pi.ID, "— dialing...")
	_ = n.h.Connect(context.Background(), pi)
}

func StartMDNS(ctx context.Context, h host.Host, serviceTag string) (func(), error) {
	ser := mdns.NewMdnsService(h, serviceTag, &mdnsNotifee{h: h})
	return func() { ser.Close() }, nil
}

// ----------------- DHT (global / WAN) -----------------

func SetupDHT(ctx context.Context, h host.Host) (routing.Routing, *dht.IpfsDHT, error) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	kdht, err := dht.New(ctx, h, dht.Datastore(ds), dht.Mode(dht.ModeAuto))
	if err != nil {
		return nil, nil, err
	}
	if err := kdht.Bootstrap(ctx); err != nil {
		return nil, nil, err
	}
	return kdht, kdht, nil
}

func AdvertiseAndFind(ctx context.Context, h host.Host, r routing.Routing, rendezvous string) error {
	rd := routingdiscovery.NewRoutingDiscovery(r)

	ttl, err := rd.Advertise(ctx, rendezvous)
	if err != nil {
		return err
	}
	fmt.Println("[DHT] Advertised rendezvous:", rendezvous, " ttl:", ttl)

	// discovery loop
	go func() {
		for {
			peerCh, err := rd.FindPeers(ctx, rendezvous)
			if err != nil {
				time.Sleep(2 * time.Second)
				continue
			}
			for pi := range peerCh {
				if pi.ID == h.ID() {
					continue
				}
				fmt.Println("[DHT] Discovered peer:", pi.ID, "— dialing...")
				_ = h.Connect(context.Background(), pi)
			}
			time.Sleep(10 * time.Second)
		}
	}()

	return nil
}

package p2p

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	ma "github.com/multiformats/go-multiaddr"
)

type FilePathGetter func(ctx context.Context, cid string) (string, error)

type Node struct {
	Host       host.Host
	DHT        *dht.IpfsDHT
	protocolID protocol.ID

	getFilePath FilePathGetter
}

func NewNode(ctx context.Context, listenAddrs []string, bootstrapPeers []string, enableMDNS bool, protocolID string, getFilePath FilePathGetter) (*Node, error) {
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(listenAddrs...),
	}
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("libp2p new: %w", err)
	}

	kdht, err := dht.New(ctx, h, dht.Mode(dht.ModeAuto))
	if err != nil {
		_ = h.Close()
		return nil, fmt.Errorf("dht new: %w", err)
	}
	if err := kdht.Bootstrap(ctx); err != nil {
		_ = kdht.Close()
		_ = h.Close()
		return nil, fmt.Errorf("dht bootstrap: %w", err)
	}

	n := &Node{
		Host:        h,
		DHT:         kdht,
		protocolID:  protocol.ID(protocolID),
		getFilePath: getFilePath,
	}

	h.SetStreamHandler(n.protocolID, n.handleFileRequest)

	for _, s := range bootstrapPeers {
		ai, err := addrInfoFromString(s)
		if err != nil {
			continue
		}
		ctxConn, cancel := context.WithTimeout(ctx, 5*time.Second)
		_ = h.Connect(ctxConn, *ai)
		cancel()
	}

	if enableMDNS {
		svc := mdns.NewMdnsService(h, "p2p-fileshare-mdns", &mdnsNotifee{h: h})
		if svc != nil {
			_ = svc.Start()
		}
	}

	return n, nil
}

func (n *Node) Close() error {
	if n.DHT != nil {
		_ = n.DHT.Close()
	}
	if n.Host != nil {
		return n.Host.Close()
	}
	return nil
}

func (n *Node) PeerID() string {
	return n.Host.ID().String()
}

func (n *Node) Addrs() []string {
	addrs := n.Host.Addrs()
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, fmt.Sprintf("%s/p2p/%s", a.String(), n.PeerID()))
	}
	return out
}

func (n *Node) Provide(ctx context.Context, cidStr string) error {
	c, err := cid.Parse(cidStr)
	if err != nil {
		return fmt.Errorf("parse cid: %w", err)
	}
	if err := n.DHT.Provide(ctx, c, true); err != nil {
		return fmt.Errorf("dht provide: %w", err)
	}
	return nil
}

func (n *Node) FindProviders(ctx context.Context, cidStr string, limit int) ([]peer.AddrInfo, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	c, err := cid.Parse(cidStr)
	if err != nil {
		return nil, fmt.Errorf("parse cid: %w", err)
	}

	ch := n.DHT.FindProvidersAsync(ctx, c, limit)
	out := make([]peer.AddrInfo, 0, limit)
	for ai := range ch {
		out = append(out, ai)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (n *Node) Fetch(ctx context.Context, from peer.AddrInfo, cidStr string) (io.ReadCloser, int64, error) {
	if err := n.Host.Connect(ctx, from); err != nil {
		return nil, 0, fmt.Errorf("connect: %w", err)
	}

	s, err := n.Host.NewStream(ctx, from.ID, n.protocolID)
	if err != nil {
		return nil, 0, fmt.Errorf("new stream: %w", err)
	}

	if _, err := io.WriteString(s, cidStr+"\n"); err != nil {
		_ = s.Reset()
		return nil, 0, fmt.Errorf("write request: %w", err)
	}

	br := bufio.NewReader(s)
	line, err := br.ReadString('\n')
	if err != nil {
		_ = s.Reset()
		return nil, 0, fmt.Errorf("read response: %w", err)
	}
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "ERR ") {
		_ = s.Reset()
		return nil, 0, fmt.Errorf("remote error: %s", strings.TrimSpace(strings.TrimPrefix(line, "ERR ")))
	}
	if !strings.HasPrefix(line, "OK ") {
		_ = s.Reset()
		return nil, 0, fmt.Errorf("bad response: %q", line)
	}
	sizeStr := strings.TrimSpace(strings.TrimPrefix(line, "OK "))
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		_ = s.Reset()
		return nil, 0, fmt.Errorf("bad size: %q", sizeStr)
	}

	return &streamReadCloser{r: br, s: s}, size, nil
}

type streamReadCloser struct {
	r io.Reader
	s network.Stream
}

func (s *streamReadCloser) Read(p []byte) (int, error) { return s.r.Read(p) }
func (s *streamReadCloser) Close() error               { return s.s.Close() }

func (n *Node) handleFileRequest(s network.Stream) {
	defer func() { _ = s.Close() }()
	br := bufio.NewReader(s)
	line, err := br.ReadString('\n')
	if err != nil {
		_ = s.Reset()
		return
	}
	cidStr := strings.TrimSpace(line)
	if cidStr == "" || n.getFilePath == nil {
		_, _ = io.WriteString(s, "ERR invalid\n")
		return
	}
	if _, err := cid.Parse(cidStr); err != nil {
		_, _ = io.WriteString(s, "ERR bad_cid\n")
		return
	}

	path, err := n.getFilePath(context.Background(), cidStr)
	if err != nil {
		_, _ = io.WriteString(s, "ERR notfound\n")
		return
	}

	f, err := os.Open(path)
	if err != nil {
		_, _ = io.WriteString(s, "ERR notfound\n")
		return
	}
	defer func() { _ = f.Close() }()

	st, err := f.Stat()
	if err != nil {
		_, _ = io.WriteString(s, "ERR stat\n")
		return
	}
	_, _ = io.WriteString(s, fmt.Sprintf("OK %d\n", st.Size()))
	_, _ = io.Copy(s, f)
}

type mdnsNotifee struct{ h host.Host }

func (n *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	_ = n.h.Connect(context.Background(), pi)
}

func addrInfoFromString(s string) (*peer.AddrInfo, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("empty")
	}
	m, err := ma.NewMultiaddr(s)
	if err != nil {
		return nil, err
	}
	ai, err := peer.AddrInfoFromP2pAddr(m)
	if err != nil {
		return nil, err
	}
	return ai, nil
}

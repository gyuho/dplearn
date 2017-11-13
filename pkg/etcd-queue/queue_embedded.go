package etcdqueue

import (
	"context"
	"fmt"
	"net/url"

	"github.com/coreos/etcd/compactor"
	"github.com/coreos/etcd/embed"
	"github.com/coreos/etcd/etcdserver/api/v3client"
	"github.com/golang/glog"
)

// implements Queue interface with a single-node embedded etcd cluster.
type embeddedQueue struct {
	srv *embed.Etcd
	Queue
}

// NewEmbeddedQueue starts a new embedded etcd server.
// cport is the TCP port used for etcd client request serving.
// pport is for etcd peer traffic, and still needed even if it's a single-node cluster.
func NewEmbeddedQueue(ctx context.Context, cport, pport int, dataDir string) (Queue, error) {
	cfg := embed.NewConfig()
	cfg.ClusterState = embed.ClusterStateFlagNew

	cfg.Name = "etcd-queue"
	cfg.Dir = dataDir

	curl := url.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", cport)}
	cfg.ACUrls, cfg.LCUrls = []url.URL{curl}, []url.URL{curl}

	purl := url.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", pport)}
	cfg.APUrls, cfg.LPUrls = []url.URL{purl}, []url.URL{purl}

	cfg.InitialCluster = fmt.Sprintf("%s=%s", cfg.Name, cfg.APUrls[0].String())

	cfg.AutoCompactionMode = compactor.ModePeriodic
	cfg.AutoCompactionRetention = "1h" // every hour
	cfg.SnapCount = 1000               // single-node, keep minimum snapshot

	glog.Infof("starting %q with endpoint %q", cfg.Name, curl.String())
	srv, err := embed.StartEtcd(cfg)
	if err != nil {
		return nil, err
	}
	select {
	case <-srv.Server.ReadyNotify():
		err = nil
	case err = <-srv.Err():
	case <-srv.Server.StopNotify():
		err = fmt.Errorf("received from etcdserver.Server.StopNotify")
	case <-ctx.Done():
		err = ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	glog.Infof("started %q with endpoint %q", cfg.Name, curl.String())

	cli := v3client.New(srv.Server)

	// issue linearized read to ensure leader election
	glog.Infof("sending GET to endpoint %q", curl.String())
	_, err = cli.Get(ctx, "foo")
	glog.Infof("sent GET to endpoint %q (error: %v)", curl.String(), err)

	cctx, cancel := context.WithCancel(ctx)
	return &embeddedQueue{
		srv: srv,
		Queue: &queue{
			cli:        cli,
			rootCtx:    cctx,
			rootCancel: cancel,
		},
	}, err
}

func (qu *embeddedQueue) Stop() {
	glog.Info("stopping queue with an embedded etcd server")
	qu.Queue.Stop()
	qu.srv.Close()
	glog.Info("stopped queue with an embedded etcd server")
}

func (qu *embeddedQueue) ClientEndpoints() []string {
	eps := make([]string, 0, len(qu.srv.Config().LCUrls))
	for i := range qu.srv.Config().LCUrls {
		eps = append(eps, qu.srv.Config().LCUrls[i].String())
	}
	return eps
}

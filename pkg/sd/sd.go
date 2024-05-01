package sd

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/mtch3n/etcd-discovery/pkg/machineinfo"
	log "github.com/sirupsen/logrus"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"path"
	"time"
)

var esdRoot = "/esd"

type (
	ServiceDiscoveryConfig struct {
		Endpoints        []string
		EtcdTimeout      time.Duration
		KeepaliveTimeout time.Duration `default:"30"`
	}

	// todo: add opt using ServiceDiscoveryOptions
	//ServiceDiscoveryOptions interface{}

	ServiceDiscovery struct {
		client       *clientv3.Client
		leaseId      clientv3.LeaseID
		uuid         string
		leaseTimeout time.Duration
		servers      map[string]machineinfo.Discovery
	}
)

func NewServiceDiscovery(c ServiceDiscoveryConfig) (*ServiceDiscovery, error) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   c.Endpoints,
		DialTimeout: c.EtcdTimeout,
	})
	if err != nil {
		return nil, err
	}

	sd, err := newServiceDiscovery(etcdClient, c.KeepaliveTimeout)
	if err != nil {
		return nil, err
	}

	return sd, nil
}

func newServiceDiscovery(client *clientv3.Client, t time.Duration) (*ServiceDiscovery, error) {

	id := uuid.New().String()
	if id == "" {
		return nil, fmt.Errorf("failed to generate uuid")
	}
	log.Tracef("sd uuid: %s", id)

	return &ServiceDiscovery{
		client:       client,
		leaseTimeout: t,
		uuid:         id,
		servers:      make(map[string]machineinfo.Discovery),
	}, nil
}

type DiscoveryStatus struct {
	InstanceId string
	Key        string
	machineinfo.Discovery
}

func (s *ServiceDiscovery) Status(ctx context.Context) ([]DiscoveryStatus, error) {
	getResp, err := s.client.Get(ctx, esdRoot, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var status []DiscoveryStatus
	if getResp.Kvs != nil {
		for _, kv := range getResp.Kvs {
			status = append(status, DiscoveryStatus{
				InstanceId: extractInstanceId(string(kv.Key)),
				Key:        string(kv.Key),
				Discovery:  *machineinfo.Deserialize(kv.Value),
			})

		}
	}

	return status, nil
}

func extractInstanceId(k string) string {
	return path.Dir(k)
}

func (s *ServiceDiscovery) Register(ctx context.Context, info *machineinfo.Discovery) error {

	err := s.acquireLease(ctx)
	if err != nil {
		return err
	}

	esdEntry := getEsdEntry(info.CanonicalName(), s.uuid)
	_, err = s.client.Put(ctx, esdEntry, info.String(), clientv3.WithLease(s.leaseId))
	if err != nil {
		return err
	}

	respCh, err := s.client.KeepAlive(ctx, s.leaseId)
	go receiveKeepAliveMessage(ctx, respCh)

	return nil
}

func receiveKeepAliveMessage(ctx context.Context, respCh <-chan *clientv3.LeaseKeepAliveResponse) {
	for {
		select {
		case <-ctx.Done():
			return
		case resp, ok := <-respCh:
			if !ok {
				log.Tracef("receiveKeepAliveMessage chan closed")
				return
			}
			log.Tracef("received keep alive response: %v", resp)
		}
	}
}

func (s *ServiceDiscovery) acquireLease(ctx context.Context) error {
	grant, err := s.client.Grant(ctx, int64second(s.leaseTimeout))

	if err != nil {
		return err
	}

	if grant == nil {
		return fmt.Errorf("failed to grant lease")
	}

	s.leaseId = grant.ID
	return nil
}

func int64second(t time.Duration) int64 {
	return int64(t.Seconds())
}

func (s *ServiceDiscovery) Get() map[string]machineinfo.Discovery {
	return s.servers
}

var serverBufferSize = 50

type (
	discoveryKvs struct {
		Key   string
		Value *machineinfo.Discovery
	}
)

// StartDiscoveryService todo: refactor
func (s *ServiceDiscovery) StartDiscoveryService(ctx context.Context, selfInfo *machineinfo.Discovery) error {

	ch := make(chan discoveryKvs, serverBufferSize)
	defer close(ch)

	esdEntry := getEsdEntry(selfInfo.CanonicalName(), "")
	getResp, err := s.client.Get(ctx, esdEntry, clientv3.WithPrefix())
	log.Info("entry listed")

	if err != nil {
		return err
	}

	if getResp.Kvs != nil {
		go pushServers(getResp.Kvs, ch)
	}

	watchCh := s.client.Watch(ctx, esdEntry, clientv3.WithPrefix())
	log.Info("watch registered")

	go watchChanges(ctx, watchCh, ch)
	log.Info("start watching changes")
	for s.update(ch) {
	}

	return nil
}

func (s *ServiceDiscovery) update(ch <-chan discoveryKvs) bool {
	kvs, ok := <-ch
	if !ok {
		return false
	}

	if kvs.Value == nil {
		delete(s.servers, kvs.Key)
		log.Tracef("server deleted: %s", kvs.Key)
	} else {
		s.servers[kvs.Key] = *kvs.Value
		log.Tracef("server added: %s", kvs.Key)
	}
	return true
}

func watchChanges(ctx context.Context, watchCh clientv3.WatchChan, ch chan<- discoveryKvs) {
	for {
		select {
		case <-ctx.Done():
			return
		case watchResp, ok := <-watchCh:
			if !ok {
				log.Warn("watch channel closed")
				return
			}
			eventFilter(watchResp.Events, ch)
		}
	}
}

func eventFilter(events []*clientv3.Event, ch chan<- discoveryKvs) {
	for _, event := range events {
		switch event.Type {
		case mvccpb.PUT:
			k, v := string(event.Kv.Key), machineinfo.Deserialize(event.Kv.Value)
			ch <- discoveryKvs{k, v}
		case mvccpb.DELETE:
			ch <- discoveryKvs{string(event.Kv.Key), nil}
		}
	}
}

func pushServers(kvs []*mvccpb.KeyValue, ch chan<- discoveryKvs) {
	for _, kv := range kvs {
		k, v := string(kv.Key), machineinfo.Deserialize(kv.Value)
		ch <- discoveryKvs{k, v}
	}
}

func (s *ServiceDiscovery) Close() error {
	return s.client.Close()
}

func getEsdEntry(cname, instanceId string) string {
	return genEsdPath(cname, "/instances", instanceId)
}

func genEsdPath(paths ...string) string {
	esdPath := path.Join(paths...)
	return path.Join(esdRoot, esdPath)
}

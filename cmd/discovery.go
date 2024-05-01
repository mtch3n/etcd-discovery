package cmd

import (
	"context"
	"fmt"
	"github.com/mtch3n/etcd-discovery/pkg/machineinfo"
	"github.com/mtch3n/etcd-discovery/pkg/sd"
	"github.com/mtch3n/etcd-discovery/pkg/signals"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net/http"
	"time"
)

type DiscoveryOption struct {
	Endpoints         []string
	ReconcileInterval time.Duration
	TestInterval      time.Duration

	//WaitTimeout       time.Duration

	name          string
	serverIp      string
	serverPort    string
	discoveryOnly bool
}

func NewCmdDiscovery() *cobra.Command {
	o := &DiscoveryOption{}
	cmd := &cobra.Command{
		Use:   "discovery",
		Short: "discovery services and run tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run()
		},
	}

	flags := cmd.Flags()
	cmd.Flags().StringSliceVar(&o.Endpoints, "endpoints", []string{"localhost:2379"}, "etcd endpoints")
	flags.DurationVarP(&o.ReconcileInterval, "reconcile", "r", 3*time.Second, "Reconcile interval")
	flags.DurationVarP(&o.TestInterval, "test", "t", 3*time.Second, "Test interval")
	flags.StringVarP(&o.name, "name", "n", "test", "Service name")
	flags.BoolVarP(&o.discoveryOnly, "discovery", "q", false, "Discovery only(no connectivity test)")
	//flags.DurationVarP(&o.WaitTimeout, "timeout", "w", 10*time.Second, "Wait timeout")

	return cmd
}

func (o *DiscoveryOption) Run() error {
	ctx := signals.SetupSignalContext()

	ch := make(chan machineinfo.Discovery)
	defer close(ch)

	go o.startDiscovery(ctx, ch)
	var httpClient = &http.Client{
		Timeout: 3 * time.Second,
	}

	ticker := time.NewTicker(o.TestInterval)
	defer ticker.Stop()

	// add waitgroup

	for {
		select {
		case <-ctx.Done():
			return nil
		case machineInfo := <-ch:
			log.Infof("recieved and update data: %v", machineInfo)
			o.serverIp = machineInfo.IP
			o.serverPort = machineInfo.Port
		case <-ticker.C:
			if o.discoveryOnly {
				continue
			}
			if doConnectivityTest(httpClient, o.serverIp, o.serverPort) {
				log.Info("connectivity test passed")
			} else {
				log.Warn("connectivity test failed")
			}
		}
	}
}

func (o *DiscoveryOption) startDiscovery(ctx context.Context, ch chan<- machineinfo.Discovery) {

	discovery, err := sd.NewServiceDiscovery(sd.ServiceDiscoveryConfig{
		Endpoints:   o.Endpoints,
		EtcdTimeout: 3 * time.Second,
	})
	defer discovery.Close()

	if err != nil {
		log.Fatalf("failed to init discover service: %v", err)
	}

	//todo: refactor
	go func() {
		err := discovery.StartDiscoveryService(ctx, machineinfo.ServiceDiscoveryInfoFromEnv())
		if err != nil {
			log.Fatalf("failed to start discovery service: %v", err)
		}
	}()
	log.Debug("service discovery started")

	ticker := time.NewTicker(o.ReconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			svcInfo := discovery.Get()
			if len(svcInfo) > 0 {
				ch <- firstServer(svcInfo)
			}
		}
	}
}

func firstServer(m map[string]machineinfo.Discovery) machineinfo.Discovery {
	if len(m) > 0 {
		for _, v := range m {
			return v
		}
	}
	return machineinfo.Discovery{}
}

func doConnectivityTest(client *http.Client, host, port string) bool {

	url := fmt.Sprintf("http://%s:%s", host, port)
	resp, err := client.Get(url)

	if err != nil {
		log.Debugf("failed to connect to %s: %v", url, err)
		return false
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return true
	default:
		log.Debugf("unexpected status code: %d", resp.StatusCode)
		return false
	}
}

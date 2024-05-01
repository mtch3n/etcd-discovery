package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/mtch3n/etcd-discovery/pkg/machineinfo"
	"github.com/mtch3n/etcd-discovery/pkg/sd"
	"github.com/mtch3n/etcd-discovery/pkg/signals"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net/http"
	"time"
)

type RegistryOptions struct {
	Endpoints   []string
	MaxLifetime time.Duration
	StartDelay  time.Duration
	port        string
	name        string
}

func NewCmdRegistry() *cobra.Command {
	o := &RegistryOptions{}

	cmd := &cobra.Command{
		Use:   "register",
		Short: "register services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run()
		},
	}

	cmd.Flags().StringSliceVar(&o.Endpoints, "endpoints", []string{"localhost:2379"}, "etcd endpoints")
	cmd.Flags().DurationVar(&o.StartDelay, "start-delay", 0, "start delay")
	cmd.Flags().DurationVar(&o.MaxLifetime, "max-lifetime", 0, "max lifetime")
	cmd.Flags().StringVar(&o.port, "port", "8080", "service port")
	cmd.Flags().StringVar(&o.name, "name", "test", "service name")

	return cmd
}

func (o *RegistryOptions) Run() error {

	if o.StartDelay != 0 {
		time.Sleep(o.StartDelay)
	}

	if o.MaxLifetime != 0 {
		go shutdownTimer(o.MaxLifetime)
	}

	ctx := signals.SetupSignalContext()

	s, _ := sd.NewServiceDiscovery(sd.ServiceDiscoveryConfig{
		Endpoints:   o.Endpoints,
		EtcdTimeout: 3 * time.Second,
	})

	if err := s.Register(ctx, machineinfo.ServiceDiscoveryInfoFromEnv()); err != nil {
		return err
	}

	o.serveHttp(ctx)

	<-ctx.Done()
	return nil
}

func shutdownTimer(after time.Duration) {
	time.Sleep(after)
	signals.RequestShutdown()
}

func (o *RegistryOptions) serveHttp(ctx context.Context) {

	srv := &http.Server{
		Addr: fmt.Sprintf(":%s", o.port),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	go func() {
		log.Infof("http server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}()

	<-ctx.Done()
	srv.Shutdown(ctx)
}

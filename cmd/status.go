package cmd

import (
	"github.com/mtch3n/etcd-discovery/pkg/sd"
	"github.com/mtch3n/etcd-discovery/pkg/signals"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"os"
	"time"
)

type StatusOption struct {
	Endpoints []string
}

func NewCmdStatus() *cobra.Command {
	o := &StatusOption{}
	cmd := &cobra.Command{
		Use: "status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run()
		},
	}

	cmd.Flags().StringSliceVar(&o.Endpoints, "endpoints", []string{"localhost:2379"}, "etcd endpoints")

	return cmd
}

func (o *StatusOption) Run() error {
	ctx := signals.SetupSignalContext()

	discovery, err := sd.NewServiceDiscovery(sd.ServiceDiscoveryConfig{
		Endpoints:   o.Endpoints,
		EtcdTimeout: 3 * time.Second,
	})
	defer discovery.Close()

	sts, err := discovery.Status(ctx)
	if err != nil {
		return err
	}

	showTable(parseRows(sts))
	return nil
}

func parseRows(data []sd.DiscoveryStatus) [][]string {

	var rows [][]string

	for _, v := range data {
		rows = append(rows, []string{v.InstanceId, v.Key, v.Name, v.Hostname, v.IP, v.Port})
	}

	return rows
}

func showTable(data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Instance ID", "DB Key", "Name", "hostname", "IP", "Port"})
	table.AppendBulk(data)
	table.Render()
}

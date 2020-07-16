// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package collect

import (
	"context"

	"github.com/newrelic/nri-vsphere/internal/performance"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/newrelic/nri-vsphere/internal/load"
	"github.com/vmware/govmomi/vim25/mo"
)

// Datastores collects data of all datastores
func Datastores(config *load.Config) {
	ctx := context.Background()
	m := config.ViewManager

	for i, dc := range config.Datacenters {
		cv, err := m.CreateContainerView(ctx, dc.Datacenter.Reference(), []string{"Datastore"}, true)
		if err != nil {
			config.Logrus.WithError(err).Error("failed to create Datastore container view")
			continue
		}
		defer func() {
			err := cv.Destroy(ctx)
			if err != nil {
				config.Logrus.WithError(err).Error("error while cleaning up datastores container view")
			}
		}()

		var datastores []mo.Datastore
		// Reference: https://code.vmware.com/apis/42/vsphere/doc/vim.Datastore.html
		err = cv.Retrieve(ctx, []string{"Datastore"}, []string{"name", "summary", "overallStatus", "vm", "host", "info"}, &datastores)
		if err != nil {
			config.Logrus.WithError(err).Error("failed to retrieve Datastore")
			continue
		}
		if err := collectTags(config, datastores, config.Datacenters[i]); err != nil {
			config.Logrus.WithError(err).Errorf("failed to retrieve tags:%v", err)
		}
		var refSlice []types.ManagedObjectReference

		for j := 0; j < len(datastores); j++ {
			config.Datacenters[i].Datastores[datastores[j].Self] = &datastores[j]
			refSlice = append(refSlice, datastores[j].Self)
		}

		if config.Args.EnableVspherePerfMetrics && dc.PerfCollector != nil {
			collectedData := dc.PerfCollector.Collect(refSlice, dc.PerfCollector.MetricDefinition.Datastore, performance.FiveMinutesInterval)
			dc.AddPerfMetrics(collectedData)
		}
	}
}

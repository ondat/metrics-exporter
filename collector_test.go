package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	configondatv1 "github.com/ondat/metrics-exporter/api/config.storageos.com/v1"
)

func TestGetEnabledMetricsCollectors(t *testing.T) {
	tests := []struct {
		name            string
		disable         []configondatv1.MetricsExporterCollector
		expectedEnabled []string
	}{
		{
			name:    "all enabled",
			disable: nil,
			expectedEnabled: []string{
				"diskstats",
				"filesystem",
			},
		},

		{
			name:    "disable filesystem",
			disable: []configondatv1.MetricsExporterCollector{configondatv1.MetricsExporterCollectorFileSystem},
			expectedEnabled: []string{
				"diskstats",
			},
		},

		{
			name:    "disable diskstats",
			disable: []configondatv1.MetricsExporterCollector{configondatv1.MetricsExporterCollectorDiskStats},
			expectedEnabled: []string{
				"filesystem",
			},
		},

		{
			name: "disable both",
			disable: []configondatv1.MetricsExporterCollector{
				configondatv1.MetricsExporterCollectorFileSystem,
				configondatv1.MetricsExporterCollectorDiskStats,
			},
			expectedEnabled: []string{},
		},

		{
			name: "disable both - reversed",
			disable: []configondatv1.MetricsExporterCollector{
				configondatv1.MetricsExporterCollectorDiskStats,
				configondatv1.MetricsExporterCollectorFileSystem,
			},
			expectedEnabled: []string{},
		},

		{
			name: "ignore bad value",
			disable: []configondatv1.MetricsExporterCollector{
				configondatv1.MetricsExporterCollector("bad"),
				configondatv1.MetricsExporterCollectorFileSystem,
			},
			expectedEnabled: []string{
				"diskstats",
			},
		},
	}

	for _, tt := range tests {
		var tt = tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loggerConfig := zap.NewProductionConfig()
			logger, _ := loggerConfig.Build()
			log := logger.Sugar()

			collectors := GetEnabledMetricsCollectors(log, tt.disable)
			names := make([]string, 0, len(collectors))
			for _, c := range collectors {
				names = append(names, c.Name())
			}
			require.ElementsMatch(t, tt.expectedEnabled, names)
		})
	}
}

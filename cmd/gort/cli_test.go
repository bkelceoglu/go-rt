package main

import (
	"strings"
	"testing"
)

func TestParseCLIArgsErgonomics(t *testing.T) {
	cases := []struct {
		args       []string
		wantKind   cliCommandKind
		wantTarget string
		wantConfig string
		wantDS     int
	}{
		{args: []string{"ls"}, wantKind: cmdListDatasets},
		{args: []string{"ls", "1"}, wantKind: cmdListDatasets, wantTarget: "1"},
		{args: []string{"ls", "advbench_harmful_goals"}, wantKind: cmdListDatasets, wantTarget: "advbench_harmful_goals"},
		{args: []string{"config.yaml", "ls"}, wantKind: cmdListDatasets},
		{args: []string{"--list-datasets"}, wantKind: cmdListDatasets},
		{args: []string{"plugins"}, wantKind: cmdListPlugins},
		{args: []string{"plugins", "1"}, wantKind: cmdListPlugins, wantTarget: "1"},
		{args: []string{"plugins", "1", "--ai"}, wantKind: cmdListPlugins, wantTarget: "1"},
		{args: []string{"plugins", "gort:plugin:aegis"}, wantKind: cmdListPlugins, wantTarget: "gort:plugin:aegis"},
		{args: []string{"ls", "plugins"}, wantKind: cmdListPlugins},
		{args: []string{"strategies"}, wantKind: cmdListStrategies},
		{args: []string{"strategies", "1"}, wantKind: cmdListStrategies, wantTarget: "1"},
		{args: []string{"strategies", "crescendo"}, wantKind: cmdListStrategies, wantTarget: "crescendo"},
		{args: []string{"ls", "strategies"}, wantKind: cmdListStrategies},
		{args: []string{"validate"}, wantKind: cmdValidate, wantConfig: "config.yaml"},
		{args: []string{"validate", "custom.yaml"}, wantKind: cmdValidate, wantConfig: "custom.yaml"},
		{args: []string{"pull"}, wantKind: cmdPullDatasets},
		{args: []string{"lakera_gandalf"}, wantKind: cmdRun, wantConfig: "config.yaml", wantDS: 1},
		{args: []string{"config.yaml", "lakera_gandalf"}, wantKind: cmdRun, wantConfig: "config.yaml", wantDS: 1},
		{args: []string{"all"}, wantKind: cmdRun, wantConfig: "config.yaml", wantDS: 1},
		{args: []string{"config.yaml"}, wantKind: cmdRun, wantConfig: "config.yaml", wantDS: 0},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			got := parseCLIArgs(tc.args)
			if got.Kind != tc.wantKind || got.TargetArg != tc.wantTarget {
				t.Fatalf("args=%v: want kind=%s target=%q, got kind=%s target=%q",
					tc.args, tc.wantKind, tc.wantTarget, got.Kind, got.TargetArg)
			}
			if tc.wantKind == cmdRun && (got.ConfigPath != tc.wantConfig || len(got.DatasetArgs) != tc.wantDS) {
				t.Fatalf("args=%v: got %+v", tc.args, got)
			}
		})
	}
}

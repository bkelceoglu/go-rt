package main

import "testing"

func TestParseCLIArgsErgonomics(t *testing.T) {
	cases := []struct {
		args       []string
		wantKind   cliCommandKind
		wantTarget string
		wantConfig string
		wantDS     int
	}{
		{[]string{"ls"}, cmdListDatasets, "", "", 0},
		{[]string{"ls", "1"}, cmdListDatasets, "1", "", 0},
		{[]string{"ls", "advbench_harmful_goals"}, cmdListDatasets, "advbench_harmful_goals", "", 0},
		{[]string{"config.yaml", "ls"}, cmdListDatasets, "", "", 0},
		{[]string{"--list-datasets"}, cmdListDatasets, "", "", 0},
		{[]string{"plugins"}, cmdListPlugins, "", "", 0},
		{[]string{"plugins", "1"}, cmdListPlugins, "1", "", 0},
		{[]string{"plugins", "1", "--ai"}, cmdListPlugins, "1", "", 0},
		{[]string{"plugins", "gort:plugin:aegis"}, cmdListPlugins, "gort:plugin:aegis", "", 0},
		{[]string{"ls", "plugins"}, cmdListPlugins, "", "", 0},
		{[]string{"strategies"}, cmdListStrategies, "", "", 0},
		{[]string{"strategies", "1"}, cmdListStrategies, "1", "", 0},
		{[]string{"strategies", "crescendo"}, cmdListStrategies, "crescendo", "", 0},
		{[]string{"ls", "strategies"}, cmdListStrategies, "", "", 0},
		{[]string{"pull"}, cmdPullDatasets, "", "", 0},
		{[]string{"lakera_gandalf"}, cmdRun, "", "config.yaml", 1},
		{[]string{"config.yaml", "lakera_gandalf"}, cmdRun, "", "config.yaml", 1},
		{[]string{"all"}, cmdRun, "", "config.yaml", 1},
		{[]string{"config.yaml"}, cmdRun, "", "config.yaml", 0},
	}
	for _, tc := range cases {
		got := parseCLIArgs(tc.args)
		if got.Kind != tc.wantKind || got.TargetArg != tc.wantTarget {
			t.Fatalf("args=%v: want kind=%s target=%q, got kind=%s target=%q",
				tc.args, tc.wantKind, tc.wantTarget, got.Kind, got.TargetArg)
		}
		if tc.wantKind == cmdRun && (got.ConfigPath != tc.wantConfig || len(got.DatasetArgs) != tc.wantDS) {
			t.Fatalf("args=%v: got %+v", tc.args, got)
		}
	}
}

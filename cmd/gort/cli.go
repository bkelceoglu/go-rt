package main

import "strings"

type cliCommandKind string

const (
	cmdHelp         cliCommandKind = "help"
	cmdVersion      cliCommandKind = "version"
	cmdListDatasets cliCommandKind = "list-datasets"
	cmdListPlugins  cliCommandKind = "list-plugins"
	cmdListResults  cliCommandKind = "list-results"
	cmdPullDatasets cliCommandKind = "pull-datasets"
	cmdWebUI        cliCommandKind = "web-ui"
	cmdRun          cliCommandKind = "run"
)

type parsedCLI struct {
	Kind        cliCommandKind
	ConfigPath  string
	TargetArg   string
	AIGenerate  bool
	Verbose     bool
	DatasetArgs []string
}

func parseCLIArgs(args []string) parsedCLI {
	if len(args) == 0 {
		return parsedCLI{Kind: cmdHelp}
	}
	if parsed, ok := detectSubcommand(args); ok {
		return parsed
	}
	return parseRunArgs(args)
}

func detectSubcommand(args []string) (parsedCLI, bool) {
	var kind cliCommandKind
	var extra []string
	cfgPath := "config.yaml"
	aiGen := false
	verbose := false
	for _, raw := range args {
		a := strings.TrimSpace(raw)
		lower := strings.ToLower(a)
		switch lower {
		case "help", "--help", "-h":
			return parsedCLI{Kind: cmdHelp}, true
		case "version", "--version":
			return parsedCLI{Kind: cmdVersion}, true
		case "--verbose", "-v":
			verbose = true
		case "--ai", "-a":
			aiGen = true
		case "results", "result", "runs", "history", "--results":
			kind = cmdListResults
		case "view", "ui", "web", "wizard", "--view", "--ui":
			kind = cmdWebUI
		case "plugins", "plugin", "--list-plugins", "--plugins", "-p":
			kind = cmdListPlugins
		case "ls", "list", "datasets", "dataset", "--list-datasets", "--datasets", "-l":
			if kind != cmdListPlugins && kind != cmdListResults {
				kind = cmdListDatasets
			}
		case "pull", "sync", "--pull-datasets", "--pull":
			return parsedCLI{Kind: cmdPullDatasets}, true
		default:
			if isYAMLPath(a) {
				cfgPath = a
			} else {
				extra = append(extra, a)
			}
		}
	}
	if kind == "" {
		return parsedCLI{}, false
	}
	target := ""
	if len(extra) > 0 {
		target = extra[0]
	}
	return parsedCLI{Kind: kind, TargetArg: target, ConfigPath: cfgPath, AIGenerate: aiGen, Verbose: verbose}, true
}

func parseRunArgs(args []string) parsedCLI {
	res := parsedCLI{Kind: cmdRun, ConfigPath: "config.yaml"}
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		lower := strings.ToLower(arg)
		if lower == "--verbose" || lower == "-v" {
			res.Verbose = true
			continue
		}
		if (arg == "-c" || arg == "--config") && i+1 < len(args) {
			res.ConfigPath = args[i+1]
			i++
			continue
		}
		if isYAMLPath(arg) {
			res.ConfigPath = arg
			continue
		}
		if arg != "run" {
			res.DatasetArgs = append(res.DatasetArgs, arg)
		}
	}
	return res
}

func isYAMLPath(arg string) bool {
	lower := strings.ToLower(arg)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

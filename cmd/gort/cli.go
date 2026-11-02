package main

import "strings"

type cliCommandKind string

const (
	cmdHelp           cliCommandKind = "help"
	cmdVersion        cliCommandKind = "version"
	cmdValidate       cliCommandKind = "validate"
	cmdListDatasets   cliCommandKind = "list-datasets"
	cmdListPlugins    cliCommandKind = "list-plugins"
	cmdListStrategies cliCommandKind = "list-strategies"
	cmdListResults    cliCommandKind = "list-results"
	cmdPullDatasets   cliCommandKind = "pull-datasets"
	cmdWebUI          cliCommandKind = "web-ui"
	cmdRun            cliCommandKind = "run"
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

type subcommandState struct {
	kind     cliCommandKind
	cfgPath  string
	target   string
	aiGen    bool
	verbose  bool
	earlyEnd bool
}

func detectSubcommand(args []string) (parsedCLI, bool) {
	st := subcommandState{cfgPath: "config.yaml"}
	for _, raw := range args {
		if handled := processSubcommandArg(strings.TrimSpace(raw), &st); handled && st.earlyEnd {
			return parsedCLI{Kind: st.kind}, true
		}
	}
	if st.kind == "" {
		return parsedCLI{}, false
	}
	return parsedCLI{Kind: st.kind, TargetArg: st.target, ConfigPath: st.cfgPath, AIGenerate: st.aiGen, Verbose: st.verbose}, true
}

func processSubcommandArg(a string, st *subcommandState) bool {
	lower := strings.ToLower(a)
	if matchEarlyCommand(lower, st) {
		return true
	}
	if matchSubcommandKind(lower, st) {
		return true
	}
	if isYAMLPath(a) {
		st.cfgPath = a
	} else if st.target == "" {
		st.target = a
	}
	return false
}

func matchEarlyCommand(lower string, st *subcommandState) bool {
	switch lower {
	case "help", "--help", "-h":
		st.kind = cmdHelp
		st.earlyEnd = true
		return true
	case "version", "--version":
		st.kind = cmdVersion
		st.earlyEnd = true
		return true
	case "pull", "sync", "--pull-datasets", "--pull":
		st.kind = cmdPullDatasets
		st.earlyEnd = true
		return true
	default:
		return false
	}
}

func matchSubcommandKind(lower string, st *subcommandState) bool {
	switch lower {
	case "--verbose", "-v":
		st.verbose = true
		return true
	case "--ai", "-a":
		st.aiGen = true
		return true
	case "results", "result", "runs", "history", "--results":
		st.kind = cmdListResults
		return true
	case "validate", "check", "--validate":
		st.kind = cmdValidate
		return true
	case "view", "ui", "web", "wizard", "--view", "--ui":
		st.kind = cmdWebUI
		return true
	case "plugins", "plugin", "--list-plugins", "--plugins", "-p":
		st.kind = cmdListPlugins
		return true
	case "strategies", "strategy", "--list-strategies", "--strategies", "-s":
		st.kind = cmdListStrategies
		return true
	case "ls", "list", "datasets", "dataset", "--list-datasets", "--datasets", "-l":
		if st.kind == "" {
			st.kind = cmdListDatasets
		}
		return true
	default:
		return false
	}
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

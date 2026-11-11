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

var earlyCommandMap = map[string]cliCommandKind{
	"help":            cmdHelp,
	"--help":          cmdHelp,
	"-h":              cmdHelp,
	"version":         cmdVersion,
	"--version":       cmdVersion,
	"pull":            cmdPullDatasets,
	"sync":            cmdPullDatasets,
	"--pull-datasets": cmdPullDatasets,
	"--pull":          cmdPullDatasets,
}

func matchEarlyCommand(lower string, st *subcommandState) bool {
	if kind, ok := earlyCommandMap[lower]; ok {
		st.kind = kind
		st.earlyEnd = true
		return true
	}
	return false
}

var subcommandKindMap = map[string]cliCommandKind{
	"results":           cmdListResults,
	"result":            cmdListResults,
	"runs":              cmdListResults,
	"history":           cmdListResults,
	"--results":         cmdListResults,
	"validate":          cmdValidate,
	"check":             cmdValidate,
	"--validate":        cmdValidate,
	"view":              cmdWebUI,
	"ui":                cmdWebUI,
	"web":               cmdWebUI,
	"wizard":            cmdWebUI,
	"--view":            cmdWebUI,
	"--ui":              cmdWebUI,
	"plugins":           cmdListPlugins,
	"plugin":            cmdListPlugins,
	"--list-plugins":    cmdListPlugins,
	"--plugins":         cmdListPlugins,
	"-p":                cmdListPlugins,
	"strategies":        cmdListStrategies,
	"strategy":          cmdListStrategies,
	"--list-strategies": cmdListStrategies,
	"--strategies":      cmdListStrategies,
	"-s":                cmdListStrategies,
}

var datasetAliasSet = map[string]bool{
	"ls": true, "list": true, "datasets": true, "dataset": true,
	"--list-datasets": true, "--datasets": true, "-l": true,
}

func matchSubcommandKind(lower string, st *subcommandState) bool {
	if matchSubcommandFlag(lower, st) {
		return true
	}
	if kind, ok := subcommandKindMap[lower]; ok {
		st.kind = kind
		return true
	}
	if datasetAliasSet[lower] {
		if st.kind == "" {
			st.kind = cmdListDatasets
		}
		return true
	}
	return false
}

func matchSubcommandFlag(lower string, st *subcommandState) bool {
	switch lower {
	case "--verbose", "-v":
		st.verbose = true
		return true
	case "--ai", "-a":
		st.aiGen = true
		return true
	default:
		return false
	}
}

func parseRunArgs(args []string) parsedCLI {
	res := parsedCLI{Kind: cmdRun, ConfigPath: "config.yaml", DatasetArgs: []string{}}
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if (arg == "-c" || arg == "--config") && i+1 < len(args) {
			res.ConfigPath = args[i+1]
			i++
			continue
		}
		applySingleRunArg(arg, &res)
	}
	return res
}

func applySingleRunArg(arg string, res *parsedCLI) {
	lower := strings.ToLower(arg)
	switch {
	case lower == "--verbose" || lower == "-v":
		res.Verbose = true
	case isYAMLPath(arg):
		res.ConfigPath = arg
	case arg != "run":
		res.DatasetArgs = append(res.DatasetArgs, arg)
	}
}

func isYAMLPath(arg string) bool {
	lower := strings.ToLower(arg)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

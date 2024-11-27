package main

var (
	flagRootName      = "root"
	flagRootShorthand = "r"
	flagRootValue     = "./proto"
	flagRootUsage     = "Set the path of the directory from which to start walking the filesystem tree in search of files matching --file-pattern."

	flagFileIncludePatternName      = "file-pattern"
	flagFileIncludePatternShorthand = "p"
	flagFileIncludePatternValue     = "*.proto"
	flagFileIncludePatternUsage     = "Set the pattern passed to filepath.Match(), used to include file names which match."

	flagModule          = "module"
	flagModuleShorthand = "m"
	flagModuleValue     = "*"
	flagModuleUsage     = "If present, only check message handlers of the given module."

	flagLogLevel          = "log-level"
	flagLogLevelShorthand = "l"
	flagLogLevelValue     = "info"
	flagLogLevelUsage     = "The logging level (debug|info|warn|error)"

	flagFixName      = "fix"
	flagFixShorthand = "f"
	flagFixValue     = false
	flagFixUsage     = "If present, protocheck will add the 'gogoproto.stable_marshaler_all' option to files which were discovered to be unstable."
)

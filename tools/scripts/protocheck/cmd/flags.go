package main

var (
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

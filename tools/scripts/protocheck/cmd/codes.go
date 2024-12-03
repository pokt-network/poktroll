package main

type ExitCode = int

const (
	CodeRootCmdErr = ExitCode(iota + 1)
	CodePathWalkErr
	CodeUnstableProtosFound
	CodeNonStatusGRPCErrorsFound
)

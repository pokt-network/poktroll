package e2e

import (
	"github.com/stretchr/testify/require"
)

func (s *suite) TheUserSuccessfullyStakesAWithUpoktForServiceFromTheAccount(actorType string, amount int64, serviceId, accName string) {
	s.TheUserStakesAWithUpoktForServiceFromTheAccount(actorType, amount, serviceId, accName)
	s.TheUserShouldBeAbleToSeeStandardOutputContaining("txhash:")
	s.TheUserShouldBeAbleToSeeStandardOutputContaining("code: 0")
	s.ThePocketdBinaryShouldExitWithoutError()
	s.TheUserShouldWaitForTheModuleMessageToBeSubmitted("application", "StakeApplication")
	s.TheForAccountIsStakedWithUpokt(actorType, accName, amount)
	s.TheAccountBalanceOfShouldBeUpoktThanBefore(accName, amount, "less")
}

func (s *suite) TheUserTransfersTheStakeFromAccountToAccount(actorType, fromAccName, toAccName string) {
	fromAddr, fromAddrIsFound := accNameToAddrMap[fromAccName]
	require.Truef(s, fromAddrIsFound, "account name %s not found in accNameToAddrMap", fromAccName)

	toAddr, toAddrIsFound := accNameToAddrMap[toAccName]
	require.Truef(s, toAddrIsFound, "account name %s not found in accNameToAddrMap", toAccName)

	args := []string{
		"tx",
		actorType,
		"transfer",
		fromAddr,
		toAddr,
		"--from",
		fromAccName,
		keyRingFlag,
		chainIdFlag,
		"-y",
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)

	s.pocketd.result = res
}

func (s *suite) ThisTestEnsuresTheForAccountIsNotStaked(actorType, accName string) {
	if _, ok := s.getStakedAmount(actorType, accName); ok {
		s.TheUserUnstakesAFromTheAccount(actorType, accName)
		s.TheUserShouldBeAbleToSeeStandardOutputContaining("txhash:")
		s.TheUserShouldBeAbleToSeeStandardOutputContaining("code: 0")
	}
}

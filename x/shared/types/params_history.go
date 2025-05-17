package types

type ParamsHistory []*ParamsUpdate

func (ph *ParamsHistory) GetCurrentParamsUpdate() *ParamsUpdate {
	var latestParamsUpdate *ParamsUpdate

	paramsUpdates := []*ParamsUpdate(*ph)
	for _, paramsUpdate := range paramsUpdates {
		if latestParamsUpdate == nil {
			latestParamsUpdate = paramsUpdate
			continue
		}

		if paramsUpdate.ActivationHeight > latestParamsUpdate.ActivationHeight {
			latestParamsUpdate = paramsUpdate
		}
	}

	return latestParamsUpdate
}

func (ph *ParamsHistory) GetCurrentParams() Params {
	currentParamsUpdate := ph.GetCurrentParamsUpdate()

	return currentParamsUpdate.Params
}

func (ph *ParamsHistory) GetParamsAtHeight(queryHeight int64) Params {
	paramsUpdateAtHeight := GetActiveParamsUpdate(*ph, queryHeight)

	return paramsUpdateAtHeight.Params
}

func (ph *ParamsHistory) GetSessionStartHeight(queryHeight int64) int64 {
	return GetSessionStartHeight(*ph, queryHeight)
}

func (ph *ParamsHistory) GetSessionEndHeight(queryHeight int64) int64 {
	return GetSessionEndHeight(*ph, queryHeight)
}

func (ph *ParamsHistory) GetSessionNumber(queryHeight int64) int64 {
	return GetSessionNumber(*ph, queryHeight)
}

func (ph *ParamsHistory) GetSessionGracePeriodEndHeight(queryHeight int64) int64 {
	return GetSessionGracePeriodEndHeight(*ph, queryHeight)
}

func (ph *ParamsHistory) IsGracePeriodElapsed(queryHeight int64, currentHeight int64) bool {
	return IsGracePeriodElapsed(*ph, queryHeight, currentHeight)
}

func (ph *ParamsHistory) GetClaimWindowOpenHeight(queryHeight int64) int64 {
	return GetClaimWindowOpenHeight(*ph, queryHeight)
}

func (ph *ParamsHistory) GetClaimWindowCloseHeight(queryHeight int64) int64 {
	return GetClaimWindowCloseHeight(*ph, queryHeight)
}

func (ph *ParamsHistory) GetProofWindowOpenHeight(queryHeight int64) int64 {
	return GetProofWindowOpenHeight(*ph, queryHeight)
}

func (ph *ParamsHistory) GetProofWindowCloseHeight(queryHeight int64) int64 {
	return GetProofWindowCloseHeight(*ph, queryHeight)
}

func (ph *ParamsHistory) GetNextSessionStartHeight(queryHeight int64) int64 {
	return GetNextSessionStartHeight(*ph, queryHeight)
}

func (ph *ParamsHistory) IsSessionEndHeight(queryHeight int64) bool {
	return IsSessionEndHeight(*ph, queryHeight)
}

func (ph *ParamsHistory) IsSessionStartHeight(queryHeight int64) bool {
	return IsSessionStartHeight(*ph, queryHeight)
}

func (ph *ParamsHistory) GetSessionEndToProofWindowCloseBlocks(queryHeight int64) int64 {
	activeParamsAtHeight := ph.GetParamsAtHeight(queryHeight)

	return GetSessionEndToProofWindowCloseBlocks(&activeParamsAtHeight)
}

func (ph *ParamsHistory) GetSettlementSessionEndHeight(queryHeight int64) int64 {
	return GetSettlementSessionEndHeight(*ph, queryHeight)
}

func (ph *ParamsHistory) GetCurrentNumBlocksPerSession() int64 {
	activeParams := ph.GetCurrentParams()

	return int64(activeParams.NumBlocksPerSession)
}

func (ph *ParamsHistory) GetNumBlocksPerSessionAtHeight(queryHeight int64) int64 {
	activeParamsAtHeight := ph.GetParamsAtHeight(queryHeight)

	return int64(activeParamsAtHeight.NumBlocksPerSession)
}

func InitialParamsHistory(sharedParams Params) ParamsHistory {
	paramsUpdate := &ParamsUpdate{
		Params:             sharedParams,
		ActivationHeight:   1,
		DeactivationHeight: 0,
	}

	return ParamsHistory{paramsUpdate}
}

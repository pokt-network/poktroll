package keeper

func (k Keeper) ResetCache() {
	k.cachedParams = nil
	clear(k.cachedServices)
	clear(k.cachedRelayMiningDifficulty)
}

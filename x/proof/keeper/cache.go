package keeper

func (k Keeper) ResetCache() {
	k.cachedParams = nil
	clear(k.cachedProofs)
	clear(k.cachedClaims)
	k.accountQuerier.ResetCache()
}

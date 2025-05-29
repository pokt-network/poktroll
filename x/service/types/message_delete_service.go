package types

func NewMsgDeleteService(creator string, ownerAddress string, serviceId string, lastSessionEndHeight int64) *MsgDeleteService {
	return &MsgDeleteService{
		Creator:              creator,
		OwnerAddress:         ownerAddress,
		ServiceId:            serviceId,
		LastSessionEndHeight: lastSessionEndHeight,
	}
}

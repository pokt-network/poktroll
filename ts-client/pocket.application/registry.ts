import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgUpdateParams } from "./types/pocket/application/tx";
import { MsgStakeApplication } from "./types/pocket/application/tx";
import { MsgUnstakeApplication } from "./types/pocket/application/tx";
import { MsgDelegateToGateway } from "./types/pocket/application/tx";
import { MsgUndelegateFromGateway } from "./types/pocket/application/tx";
import { MsgTransferApplication } from "./types/pocket/application/tx";
import { MsgUpdateParam } from "./types/pocket/application/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/pocket.application.MsgUpdateParams", MsgUpdateParams],
    ["/pocket.application.MsgStakeApplication", MsgStakeApplication],
    ["/pocket.application.MsgUnstakeApplication", MsgUnstakeApplication],
    ["/pocket.application.MsgDelegateToGateway", MsgDelegateToGateway],
    ["/pocket.application.MsgUndelegateFromGateway", MsgUndelegateFromGateway],
    ["/pocket.application.MsgTransferApplication", MsgTransferApplication],
    ["/pocket.application.MsgUpdateParam", MsgUpdateParam],
    
];

export { msgTypes }
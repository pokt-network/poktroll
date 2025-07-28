import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgUpdateParams } from "./types/pocket/gateway/tx";
import { MsgStakeGateway } from "./types/pocket/gateway/tx";
import { MsgUnstakeGateway } from "./types/pocket/gateway/tx";
import { MsgUpdateParam } from "./types/pocket/gateway/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/pocket.gateway.MsgUpdateParams", MsgUpdateParams],
    ["/pocket.gateway.MsgStakeGateway", MsgStakeGateway],
    ["/pocket.gateway.MsgUnstakeGateway", MsgUnstakeGateway],
    ["/pocket.gateway.MsgUpdateParam", MsgUpdateParam],
    
];

export { msgTypes }
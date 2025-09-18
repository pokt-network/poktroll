import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgUpdateParams } from "./types/pocket/service/tx";
import { MsgUpdateParam } from "./types/pocket/service/tx";
import { MsgAddService } from "./types/pocket/service/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/pocket.service.MsgUpdateParams", MsgUpdateParams],
    ["/pocket.service.MsgUpdateParam", MsgUpdateParam],
    ["/pocket.service.MsgAddService", MsgAddService],
    
];

export { msgTypes }
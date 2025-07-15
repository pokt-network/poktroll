import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgRegisterPayee } from "./types/ibc/applications/fee/v1/tx";
import { MsgRegisterCounterpartyPayee } from "./types/ibc/applications/fee/v1/tx";
import { MsgPayPacketFee } from "./types/ibc/applications/fee/v1/tx";
import { MsgPayPacketFeeAsync } from "./types/ibc/applications/fee/v1/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/ibc.applications.fee.v1.MsgRegisterPayee", MsgRegisterPayee],
    ["/ibc.applications.fee.v1.MsgRegisterCounterpartyPayee", MsgRegisterCounterpartyPayee],
    ["/ibc.applications.fee.v1.MsgPayPacketFee", MsgPayPacketFee],
    ["/ibc.applications.fee.v1.MsgPayPacketFeeAsync", MsgPayPacketFeeAsync],
    
];

export { msgTypes }
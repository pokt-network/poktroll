import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgUpdateParams } from "./types/pocket/proof/tx";
import { MsgCreateClaim } from "./types/pocket/proof/tx";
import { MsgSubmitProof } from "./types/pocket/proof/tx";
import { MsgUpdateParam } from "./types/pocket/proof/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/pocket.proof.MsgUpdateParams", MsgUpdateParams],
    ["/pocket.proof.MsgCreateClaim", MsgCreateClaim],
    ["/pocket.proof.MsgSubmitProof", MsgSubmitProof],
    ["/pocket.proof.MsgUpdateParam", MsgUpdateParam],
    
];

export { msgTypes }
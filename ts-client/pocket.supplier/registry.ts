import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgUpdateParams } from "./types/pocket/supplier/tx";
import { MsgStakeSupplier } from "./types/pocket/supplier/tx";
import { MsgUnstakeSupplier } from "./types/pocket/supplier/tx";
import { MsgUpdateParam } from "./types/pocket/supplier/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/pocket.supplier.MsgUpdateParams", MsgUpdateParams],
    ["/pocket.supplier.MsgStakeSupplier", MsgStakeSupplier],
    ["/pocket.supplier.MsgUnstakeSupplier", MsgUnstakeSupplier],
    ["/pocket.supplier.MsgUpdateParam", MsgUpdateParam],
    
];

export { msgTypes }
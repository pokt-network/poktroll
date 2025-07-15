import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgUpdateParams } from "./types/pocket/migration/tx";
import { MsgImportMorseClaimableAccounts } from "./types/pocket/migration/tx";
import { MsgClaimMorseAccount } from "./types/pocket/migration/tx";
import { MsgClaimMorseApplication } from "./types/pocket/migration/tx";
import { MsgClaimMorseSupplier } from "./types/pocket/migration/tx";
import { MsgRecoverMorseAccount } from "./types/pocket/migration/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/pocket.migration.MsgUpdateParams", MsgUpdateParams],
    ["/pocket.migration.MsgImportMorseClaimableAccounts", MsgImportMorseClaimableAccounts],
    ["/pocket.migration.MsgClaimMorseAccount", MsgClaimMorseAccount],
    ["/pocket.migration.MsgClaimMorseApplication", MsgClaimMorseApplication],
    ["/pocket.migration.MsgClaimMorseSupplier", MsgClaimMorseSupplier],
    ["/pocket.migration.MsgRecoverMorseAccount", MsgRecoverMorseAccount],
    
];

export { msgTypes }
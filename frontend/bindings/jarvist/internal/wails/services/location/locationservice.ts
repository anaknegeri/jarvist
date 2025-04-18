// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import { Call as $Call, CancellablePromise as $CancellablePromise, Create as $Create } from "@wailsio/runtime";

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import * as models$0 from "../../../common/models/models.js";

export function CreateLocation(input: models$0.LocationInput): $CancellablePromise<models$0.Location | null> {
    return $Call.ByID(2195808865, input).then(($result: any) => {
        return $$createType1($result);
    });
}

export function DeleteLocation(id: string): $CancellablePromise<void> {
    return $Call.ByID(253386362, id);
}

export function ListLocations(): $CancellablePromise<models$0.Location[]> {
    return $Call.ByID(3709680806).then(($result: any) => {
        return $$createType2($result);
    });
}

export function UpdateLocation(id: string, input: models$0.LocationInput): $CancellablePromise<models$0.Location | null> {
    return $Call.ByID(2389794152, id, input).then(($result: any) => {
        return $$createType1($result);
    });
}

// Private type creation functions
const $$createType0 = models$0.Location.createFrom;
const $$createType1 = $Create.Nullable($$createType0);
const $$createType2 = $Create.Array($$createType0);

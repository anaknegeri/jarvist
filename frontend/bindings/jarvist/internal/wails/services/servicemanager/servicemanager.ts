// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

/**
 * ServiceManager handles interactions with the Windows system service
 * @module
 */

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import { Call as $Call, CancellablePromise as $CancellablePromise, Create as $Create } from "@wailsio/runtime";

export function CheckAndInstallService(): $CancellablePromise<string> {
    return $Call.ByID(463061348);
}

export function EnsureServiceRunning(): $CancellablePromise<string> {
    return $Call.ByID(2815307231);
}

export function GetServiceDetails(): $CancellablePromise<{ [_: string]: any }> {
    return $Call.ByID(1469174418).then(($result: any) => {
        return $$createType0($result);
    });
}

export function GetServiceStatus(): $CancellablePromise<string> {
    return $Call.ByID(1234762062);
}

export function InstallService(): $CancellablePromise<string> {
    return $Call.ByID(1326803921);
}

export function IsServiceInstalled(): $CancellablePromise<boolean> {
    return $Call.ByID(562112920);
}

export function IsServiceRunning(): $CancellablePromise<boolean> {
    return $Call.ByID(1986086321);
}

/**
 * RestartService restarts the Windows service
 */
export function RestartService(): $CancellablePromise<string> {
    return $Call.ByID(1085184201);
}

export function StartService(): $CancellablePromise<string> {
    return $Call.ByID(1067066616);
}

export function StopService(): $CancellablePromise<string> {
    return $Call.ByID(3943680838);
}

export function UninstallService(): $CancellablePromise<string> {
    return $Call.ByID(187585130);
}

// Private type creation functions
const $$createType0 = $Create.Map($Create.Any, $Create.Any);

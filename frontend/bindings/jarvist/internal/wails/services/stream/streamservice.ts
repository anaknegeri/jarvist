// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import { Call as $Call, CancellablePromise as $CancellablePromise, Create as $Create } from "@wailsio/runtime";

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import * as application$0 from "../../../../../github.com/wailsapp/wails/v3/pkg/application/models.js";

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import * as $models from "./models.js";

export function Cleanup(): $CancellablePromise<void> {
    return $Call.ByID(3094871872);
}

export function GetConfig(): $CancellablePromise<$models.MJPEGStreamConfig> {
    return $Call.ByID(3267436918).then(($result: any) => {
        return $$createType0($result);
    });
}

export function GetStreamURL(): $CancellablePromise<string> {
    return $Call.ByID(1487283833);
}

export function InitService(app: application$0.App | null): $CancellablePromise<void> {
    return $Call.ByID(2538253653, app);
}

export function IsStreamRunning(): $CancellablePromise<boolean> {
    return $Call.ByID(3878313739);
}

export function OnShutdown(): $CancellablePromise<void> {
    return $Call.ByID(3924232763);
}

export function OnStartup(options: application$0.ServiceOptions): $CancellablePromise<void> {
    return $Call.ByID(3626171258, options);
}

export function StartStream(): $CancellablePromise<string> {
    return $Call.ByID(3951203712);
}

export function StopStream(): $CancellablePromise<string> {
    return $Call.ByID(2534692284);
}

export function UpdateConfig(config: $models.MJPEGStreamConfig): $CancellablePromise<void> {
    return $Call.ByID(2307795135, config);
}

// Private type creation functions
const $$createType0 = $models.MJPEGStreamConfig.createFrom;

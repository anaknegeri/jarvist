// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import { Create as $Create } from "@wailsio/runtime";

export class DeviceInfo {
    "Name": string;
    "OS": string;
    "Architecture": string;

    /** Creates a new DeviceInfo instance. */
    constructor($$source: Partial<DeviceInfo> = {}) {
        if (!("Name" in $$source)) {
            this["Name"] = "";
        }
        if (!("OS" in $$source)) {
            this["OS"] = "";
        }
        if (!("Architecture" in $$source)) {
            this["Architecture"] = "";
        }

        Object.assign(this, $$source);
    }

    /**
     * Creates a new DeviceInfo instance from a string or object.
     */
    static createFrom($$source: any = {}): DeviceInfo {
        let $$parsedSource = typeof $$source === 'string' ? JSON.parse($$source) : $$source;
        return new DeviceInfo($$parsedSource as Partial<DeviceInfo>);
    }
}

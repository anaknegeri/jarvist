// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

/**
 * LogService handles log-related operations
 * @module
 */

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import { Call as $Call, CancellablePromise as $CancellablePromise, Create as $Create } from "@wailsio/runtime";

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import * as time$0 from "../../../../../time/models.js";

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import * as $models from "./models.js";

/**
 * DownloadLogs copies log files from all directories to a specified destination
 */
export function DownloadLogs(destinationDir: string): $CancellablePromise<string[]> {
    return $Call.ByID(2044453966, destinationDir).then(($result: any) => {
        return $$createType0($result);
    });
}

/**
 * ExportLogsToCSV exports filtered logs to a CSV file
 */
export function ExportLogsToCSV(outputPath: string, level: string, startDate: time$0.Time, endDate: time$0.Time, searchTerm: string): $CancellablePromise<void> {
    return $Call.ByID(707694399, outputPath, level, startDate, endDate, searchTerm);
}

/**
 * FilterLogs allows filtering logs based on various criteria
 */
export function FilterLogs(level: string, startDate: time$0.Time, endDate: time$0.Time, searchTerm: string): $CancellablePromise<$models.LogEntry[]> {
    return $Call.ByID(1330620320, level, startDate, endDate, searchTerm).then(($result: any) => {
        return $$createType2($result);
    });
}

/**
 * GetLogFiles returns a list of available log files
 */
export function GetLogFiles(): $CancellablePromise<string[]> {
    return $Call.ByID(3712390938).then(($result: any) => {
        return $$createType0($result);
    });
}

/**
 * ReadLogs reads log files from all log directories
 */
export function ReadLogs(): $CancellablePromise<string[]> {
    return $Call.ByID(829077226).then(($result: any) => {
        return $$createType0($result);
    });
}

/**
 * SetupLogRotation manages log file rotation
 */
export function SetupLogRotation(maxFiles: number, maxSizeBytes: number): $CancellablePromise<void> {
    return $Call.ByID(186090782, maxFiles, maxSizeBytes);
}

// Private type creation functions
const $$createType0 = $Create.Array($Create.Any);
const $$createType1 = $models.LogEntry.createFrom;
const $$createType2 = $Create.Array($$createType1);

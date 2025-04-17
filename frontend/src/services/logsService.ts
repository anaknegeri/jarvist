export interface LogEntry {
  timestamp: string;
  level: string;
  component: string;
  message: string;
  details: string;
}

export function parseLogLine(line: string): LogEntry | null {
  const logRegex = /^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) - (\w+) -\s*(.+)$/;
  const match = line.match(logRegex);

  if (!match) return null;

  return {
    timestamp: match[1],
    level: match[2].toLowerCase(),
    component: "Scheduler",
    message: "Database Update",
    details: match[3],
  };
}

export function processLogLines(lines: string[]): LogEntry[] {
  return lines
    .map(parseLogLine)
    .filter((entry): entry is LogEntry => entry !== null);
}

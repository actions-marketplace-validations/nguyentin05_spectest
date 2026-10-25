import { configHandlers } from "./config";
import { generateHandlers } from "./generate";
import { healthHandlers } from "./health";
import { historyHandlers } from "./history";
import { logsHandlers } from "./logs";
import { markdownHandlers } from "./markdown";
import { probeHandlers } from "./probe";
import { suiteHandlers } from "./suite";
import { workspaceHandlers } from "./workspace";

export const handlers = [
  ...healthHandlers,
  ...logsHandlers,
  ...configHandlers,
  ...workspaceHandlers,
  ...suiteHandlers,
  ...historyHandlers,
  ...probeHandlers,
  ...generateHandlers,
  ...markdownHandlers,
];

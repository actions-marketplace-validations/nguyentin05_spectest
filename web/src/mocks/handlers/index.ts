import { configHandlers } from "./config";
import { healthHandlers } from "./health";
import { historyHandlers } from "./history";
import { logsHandlers } from "./logs";
import { markdownHandlers } from "./markdown";
import { suiteHandlers } from "./suite";
import { workspaceHandlers } from "./workspace";

export const handlers = [
  ...healthHandlers,
  ...logsHandlers,
  ...configHandlers,
  ...workspaceHandlers,
  ...suiteHandlers,
  ...historyHandlers,
  ...markdownHandlers,
];

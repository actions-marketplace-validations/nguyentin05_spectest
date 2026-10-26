import type {
  LLMConfig,
  MarkdownResponse,
  PlanResponse,
  PlanScenario,
  PlanSetup,
  PlanStats,
  PlanStep,
  PlanSeed,
} from "./types";

export type RawConfig = {
  config_file: string;
  default_provider: string;
  active_model: string;
  endpoint_url: string;
  api_key_masked: string;
  has_api_key: boolean;
};

export function toLLMConfig(raw: RawConfig): LLMConfig {
  return {
    provider: raw.default_provider,
    model: raw.active_model,
    endpoint: raw.endpoint_url,
    has_api_key: raw.has_api_key,
    api_key_masked: raw.api_key_masked,
  };
}

export type RawMarkdown = {
  path?: string;
  title: string;
  sections: string[];
  table_count: number;
  line_count: number;
  raw_content: string;
};

export function toMarkdownResponse(raw: RawMarkdown, fallbackPath: string): MarkdownResponse {
  const path = raw.path || fallbackPath;
  return {
    path,
    file_name: path.split(/[\\/]/).pop() ?? path,
    content: raw.raw_content,
  };
}

// Go omits optional YAML blocks (omitempty) and encodes nil slices as null,
// so hand-written suites routinely miss keys the UI expects to be present.
export type RawPlanScenario = Partial<Omit<PlanScenario, "steps">> & {
  steps?: PlanStep[] | null;
};

export type RawPlan = {
  path?: string;
  version?: number;
  servers?: string[] | null;
  apis?: string[] | null;
  analysis?: string;
  seed?: Record<string, PlanSeed> | null;
  setup?: Record<string, Partial<PlanSetup>> | null;
  scenarios?: RawPlanScenario[] | null;
  stats?: Partial<PlanStats> | null;
};

export function toPlanResponse(raw: RawPlan): PlanResponse {
  return {
    path: raw.path ?? "",
    version: raw.version ?? 0,
    servers: raw.servers ?? [],
    apis: raw.apis ?? [],
    analysis: raw.analysis ?? "",
    seed: raw.seed ?? {},
    setup: Object.fromEntries(
      Object.entries(raw.setup ?? {}).map(([id, setup]) => [
        id,
        { ...setup, steps: setup.steps ?? [] },
      ]),
    ),
    scenarios: (raw.scenarios ?? []).map((scenario) => ({
      id: scenario.id ?? "",
      name: scenario.name ?? "",
      setup: scenario.setup ?? [],
      steps: scenario.steps ?? [],
      locked: scenario.locked ?? false,
      skip: scenario.skip ?? "",
    })),
    stats: {
      total_scenarios: raw.stats?.total_scenarios ?? 0,
      total_setups: raw.stats?.total_setups ?? 0,
      total_steps: raw.stats?.total_steps ?? 0,
      locked_scenarios: raw.stats?.locked_scenarios ?? 0,
      skipped_scenarios: raw.stats?.skipped_scenarios ?? 0,
    },
  };
}

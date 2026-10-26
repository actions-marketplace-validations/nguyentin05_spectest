import type { LLMConfig, MarkdownResponse } from "./types";

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

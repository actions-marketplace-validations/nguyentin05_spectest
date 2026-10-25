import { useState, type FormEvent, type ReactNode } from "react";
import type { HealthcheckResponse, LLMConfig } from "../api/types";
import { api } from "../api/endpoints";
import { isMissingEndpoint } from "../api/errors";
import { useApi } from "../hooks/useApi";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { CapabilityNotice } from "../components/ui/CapabilityNotice";
import { Card } from "../components/ui/Card";
import { ErrorState } from "../components/ui/ErrorState";
import { Field } from "../components/ui/Field";
import { LoadingBlock } from "../components/ui/LoadingBlock";
import { useToast } from "../components/ui/Toast";

const PROVIDERS = [
  { value: "gemini", label: "Google Gemini" },
  { value: "openai", label: "OpenAI" },
  { value: "local", label: "Local server" },
  { value: "custom", label: "Tùy chỉnh" },
];

const KEY_REQUIRED = "Cần API key cho nhà cung cấp này";

function requiresKey(provider: string): boolean {
  return provider === "gemini" || provider === "openai";
}

function errorMessage(cause: unknown, fallback: string): string {
  return cause instanceof Error ? cause.message : fallback;
}

function SettingsForm({ config, onSaved }: { config: LLMConfig; onSaved: () => void }) {
  const { push } = useToast();
  const [provider, setProvider] = useState(config.provider);
  const [endpoint, setEndpoint] = useState(config.endpoint);
  const [model, setModel] = useState(config.model);
  const [apiKey, setApiKey] = useState("");
  const [keyProvider, setKeyProvider] = useState(config.provider);
  const [hasKey, setHasKey] = useState(config.has_api_key);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [ping, setPing] = useState<string | null>(null);

  // A stored key only counts for the provider it was saved with.
  const hasUsableKey = hasKey && provider === keyProvider;

  function validate(): string | null {
    if (!endpoint.trim() || !model.trim()) {
      return "Nhập đầy đủ endpoint và model.";
    }
    if (requiresKey(provider) && !hasUsableKey && !apiKey.trim()) {
      return KEY_REQUIRED;
    }
    return null;
  }

  function payload() {
    const key = apiKey.trim();
    return {
      provider,
      endpoint: endpoint.trim(),
      model: model.trim(),
      ...(key ? { api_key: key } : {}),
    };
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const invalid = validate();
    if (invalid) {
      setError(invalid);
      return;
    }

    setError(null);
    setSaving(true);
    try {
      await api.saveConfig(payload());
      if (apiKey.trim()) setHasKey(true);
      setKeyProvider(provider);
      setApiKey("");
      push("Đã lưu cấu hình", "success");
      onSaved();
    } catch (cause) {
      setError(errorMessage(cause, "Lưu cấu hình thất bại"));
    } finally {
      setSaving(false);
    }
  }

  async function testConnection() {
    const invalid = validate();
    if (invalid) {
      setError(invalid);
      return;
    }

    setError(null);
    setPing(null);
    setTesting(true);
    try {
      const result = await api.testConfig(payload());
      setPing(`Kết nối thành công · ${result.latency_ms} ms`);
    } catch (cause) {
      setError(errorMessage(cause, "Kiểm tra kết nối thất bại"));
    } finally {
      setTesting(false);
    }
  }

  return (
    <Card
      title="AI model"
      subtitle="Cấu hình nhà cung cấp dùng để sinh kịch bản kiểm thử."
      actions={
        <Badge tone={hasUsableKey ? "pass" : "skip"}>
          {hasUsableKey ? "Đã cấu hình" : "Chưa có key"}
        </Badge>
      }
    >
      <form className="st-settings__form" onSubmit={(event) => void save(event)}>
        <div className="st-settings__row">
          <Field id="settings-provider" label="Nhà cung cấp">
            <select value={provider} onChange={(event) => setProvider(event.target.value)}>
              {PROVIDERS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </Field>

          <Field id="settings-model" label="Model">
            <input
              value={model}
              onChange={(event) => setModel(event.target.value)}
              placeholder="gemini-1.5-flash"
            />
          </Field>
        </div>

        <Field id="settings-endpoint" label="Endpoint">
          <input
            value={endpoint}
            onChange={(event) => setEndpoint(event.target.value)}
            placeholder="https://generativelanguage.googleapis.com"
          />
        </Field>

        <Field
          id="settings-api-key"
          label="API key"
          hint="Để trống để giữ key hiện tại"
        >
          <input
            type="password"
            value={apiKey}
            onChange={(event) => setApiKey(event.target.value)}
            placeholder={config.api_key_masked || "Nhập API key"}
            autoComplete="off"
          />
        </Field>

        {error && (
          <p className="st-settings__message st-settings__message--error" role="alert">
            {error}
          </p>
        )}
        {ping && (
          <p className="st-settings__message st-settings__message--ok" role="status">
            {ping}
          </p>
        )}

        <div className="st-settings__actions">
          <Button type="submit" loading={saving}>
            Lưu cấu hình
          </Button>
          <Button
            variant="secondary"
            loading={testing}
            onClick={() => void testConnection()}
          >
            Kiểm tra kết nối
          </Button>
        </div>
      </form>
    </Card>
  );
}

function HealthcheckCard() {
  const [targetUrl, setTargetUrl] = useState("http://localhost:8080");
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<HealthcheckResponse | null>(null);

  async function check() {
    if (!targetUrl.trim()) {
      setError("Nhập URL máy chủ đích.");
      return;
    }

    setError(null);
    setResult(null);
    setChecking(true);
    try {
      setResult(
        await api.healthcheck({ target_url: targetUrl.trim(), timeout_seconds: 5 }),
      );
    } catch (cause) {
      setError(errorMessage(cause, "Kiểm tra máy chủ thất bại"));
    } finally {
      setChecking(false);
    }
  }

  return (
    <Card
      title="Kiểm tra máy chủ đích"
      subtitle="Pre-flight tới API backend trước khi chạy kịch bản."
    >
      <div className="st-settings__healthcheck">
        <Field id="settings-target-url" label="URL máy chủ đích">
          <input
            value={targetUrl}
            onChange={(event) => setTargetUrl(event.target.value)}
            placeholder="http://localhost:8080"
          />
        </Field>
        <Button variant="secondary" loading={checking} onClick={() => void check()}>
          Kiểm tra máy chủ
        </Button>
      </div>

      {error && (
        <p className="st-settings__message st-settings__message--error" role="alert">
          {error}
        </p>
      )}
      {result && (
        <div className="st-settings__health">
          <Badge tone={result.ready ? "pass" : "fail"}>
            {result.ready ? "Sẵn sàng" : "Không phản hồi"}
          </Badge>
          <span>{result.latency_ms} ms</span>
          <p>{result.message}</p>
        </div>
      )}
    </Card>
  );
}

export function SettingsPage() {
  const { data, error, loading, reload } = useApi(api.getConfig);

  let content: ReactNode = null;

  if (loading) {
    content = <LoadingBlock label="Đang tải cấu hình…" rows={3} />;
  } else if (error) {
    content = isMissingEndpoint(error) ? (
      <CapabilityNotice feature="cấu hình LLM" error={error} />
    ) : (
      <ErrorState error={error} onRetry={reload} />
    );
  } else if (data) {
    content = (
      <>
        <SettingsForm config={data} onSaved={reload} />
        <HealthcheckCard />
      </>
    );
  }

  return (
    <section className="st-page">
      <header className="st-page__head">
        <h1>Cài đặt</h1>
        <p className="st-page__subtitle">
          Cấu hình nhà cung cấp AI và kiểm tra kết nối máy chủ đích.
        </p>
      </header>
      {content}
    </section>
  );
}

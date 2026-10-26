import { useState, type DragEvent, type FormEvent } from "react";
import { api } from "../../api/endpoints";
import { useToast } from "../ui/Toast";
import { Button } from "../ui/Button";
import { Card } from "../ui/Card";
import { Field } from "../ui/Field";
import { Tabs } from "../ui/Tabs";

const MAX_UPLOAD_BYTES = 10 * 1024 * 1024;
// shortcut: 10 MB limit is client-side only, backend must enforce its own
const FILE_TYPES = [".yaml", ".yml", ".json", ".md"];
const TYPE_ERROR = "Chỉ nhận tệp .yaml, .yml, .json hoặc .md";
const SIZE_ERROR = "Tệp tối đa 10 MB";

function validateFile(name: string, size: number): string | null {
  const lower = name.toLowerCase();
  if (!FILE_TYPES.some((extension) => lower.endsWith(extension))) return TYPE_ERROR;
  if (size > MAX_UPLOAD_BYTES) return SIZE_ERROR;
  return null;
}

function errorMessage(cause: unknown): string {
  return cause instanceof Error ? cause.message : "Tải tệp thất bại";
}

function FileUploadForm({ onUploaded }: { onUploaded: () => void }) {
  const { push } = useToast();
  const [error, setError] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);

  async function upload(file: File) {
    const invalid = validateFile(file.name, file.size);
    if (invalid) {
      setError(invalid);
      return;
    }

    setError(null);
    setUploading(true);
    try {
      const result = await api.uploadFile(file);
      push(`Đã tải lên ${result.file_name}`, "success");
      onUploaded();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setUploading(false);
    }
  }

  function handleDrop(event: DragEvent<HTMLLabelElement>) {
    event.preventDefault();
    const file = event.dataTransfer.files[0];
    if (file) void upload(file);
  }

  return (
    <div className="st-upload__form">
      <label
        className="st-upload__dropzone"
        onDragOver={(event) => event.preventDefault()}
        onDrop={handleDrop}
      >
        <span className="st-upload__drop-title">Kéo thả tệp vào đây</span>
        <span className="st-upload__hint">
          hoặc chọn từ máy · .yaml, .yml, .json, .md · tối đa 10 MB
        </span>
        {/* No accept attr: unsupported files must reach the validator so the reason is visible. */}
        <input
          id="workspace-upload-file"
          className="st-visually-hidden"
          type="file"
          aria-label="Chọn tệp để tải lên"
          disabled={uploading}
          onChange={(event) => {
            const file = event.target.files?.[0];
            event.target.value = "";
            if (file) void upload(file);
          }}
        />
      </label>
      {uploading && <p className="st-upload__hint">Đang tải lên…</p>}
      {error && (
        <p className="st-upload__error" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}

function TextUploadForm({ onUploaded }: { onUploaded: () => void }) {
  const { push } = useToast();
  const [name, setName] = useState("");
  const [content, setContent] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const trimmed = name.trim();
    if (!trimmed) {
      setError("Nhập tên tệp.");
      return;
    }

    const invalid = validateFile(trimmed, new TextEncoder().encode(content).length);
    if (invalid) {
      setError(invalid);
      return;
    }
    if (!content.trim()) {
      setError("Nhập nội dung tệp.");
      return;
    }

    setError(null);
    setSaving(true);
    try {
      const result = await api.uploadText({ file_name: trimmed, content });
      push(`Đã tải lên ${result.file_name}`, "success");
      setName("");
      setContent("");
      onUploaded();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setSaving(false);
    }
  }

  return (
    <form className="st-upload__form" onSubmit={(event) => void submit(event)}>
      <Field
        id="workspace-upload-name"
        label="Tên tệp"
        hint="Ví dụ: openapi-petstore.yaml"
      >
        <input
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="openapi-petstore.yaml"
        />
      </Field>

      <Field id="workspace-upload-content" label="Nội dung">
        <textarea value={content} onChange={(event) => setContent(event.target.value)} />
      </Field>

      {error && (
        <p className="st-upload__error" role="alert">
          {error}
        </p>
      )}

      <div>
        <Button type="submit" loading={saving}>
          Tải lên nội dung
        </Button>
      </div>
    </form>
  );
}

export function UploadPanel({ onUploaded }: { onUploaded: () => void }) {
  return (
    <Card
      id="upload"
      title="Tải tệp lên"
      subtitle="Thêm đặc tả API, yêu cầu hoặc kịch bản vào workspace."
    >
      <Tabs
        label="Cách tải tệp"
        items={[
          {
            id: "file",
            label: "Tải tệp",
            content: <FileUploadForm onUploaded={onUploaded} />,
          },
          {
            id: "text",
            label: "Dán nội dung",
            content: <TextUploadForm onUploaded={onUploaded} />,
          },
        ]}
      />
    </Card>
  );
}

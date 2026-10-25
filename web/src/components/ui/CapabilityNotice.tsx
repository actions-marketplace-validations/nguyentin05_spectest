import { isMissingEndpoint } from "../../api/errors";
import { isMock } from "../../lib/mode";

type CapabilityNoticeProps = {
  feature: string;
  error?: unknown;
};

export function CapabilityNotice({ feature, error }: CapabilityNoticeProps) {
  if (error !== undefined && !isMissingEndpoint(error)) return null;

  const message =
    error !== undefined
      ? `Backend chưa hỗ trợ ${feature}. Giao diện đang chạy ở chế độ mẫu.`
      : isMock
        ? "Tính năng này đang dùng dữ liệu mẫu."
        : `Backend chưa hỗ trợ ${feature}.`;

  return (
    <p className="st-notice" role="status">
      {message}
    </p>
  );
}

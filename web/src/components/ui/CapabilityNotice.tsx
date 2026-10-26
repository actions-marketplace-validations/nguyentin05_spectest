import { isMissingEndpoint } from "../../api/errors";

type CapabilityNoticeProps = {
  feature: string;
  error?: unknown;
};

export function CapabilityNotice({ feature, error }: CapabilityNoticeProps) {
  if (error !== undefined && !isMissingEndpoint(error)) return null;

  const message = `Backend chưa hỗ trợ ${feature}.`;

  return (
    <p className="st-notice" role="status">
      {message}
    </p>
  );
}

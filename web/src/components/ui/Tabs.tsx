import { useId, useRef, useState, type KeyboardEvent, type ReactNode } from "react";

export type TabItem = {
  id: string;
  label: string;
  content: ReactNode;
};

type TabsProps = {
  items: TabItem[];
  value?: string;
  defaultValue?: string;
  onChange?: (id: string) => void;
  label?: string;
};

export function Tabs({ items, value, defaultValue, onChange, label }: TabsProps) {
  const baseId = useId();
  const [internal, setInternal] = useState(defaultValue ?? items[0]?.id ?? "");
  const active = value ?? internal;
  const activeItem = items.find((item) => item.id === active) ?? items[0];
  const refs = useRef<(HTMLButtonElement | null)[]>([]);

  function select(id: string) {
    if (value === undefined) setInternal(id);
    onChange?.(id);
  }

  function handleKeyDown(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    const last = items.length - 1;
    let next = index;

    if (event.key === "ArrowRight") next = index === last ? 0 : index + 1;
    else if (event.key === "ArrowLeft") next = index === 0 ? last : index - 1;
    else if (event.key === "Home") next = 0;
    else if (event.key === "End") next = last;
    else return;

    event.preventDefault();
    const nextItem = items[next];
    if (!nextItem) return;
    select(nextItem.id);
    refs.current[next]?.focus();
  }

  return (
    <div className="st-tabs">
      <div className="st-tabs__list" role="tablist" aria-label={label}>
        {items.map((item, index) => (
          <button
            key={item.id}
            ref={(element) => {
              refs.current[index] = element;
            }}
            type="button"
            role="tab"
            id={`${baseId}-tab-${item.id}`}
            aria-selected={activeItem?.id === item.id}
            aria-controls={`${baseId}-panel-${item.id}`}
            tabIndex={activeItem?.id === item.id ? 0 : -1}
            onClick={() => select(item.id)}
            onKeyDown={(event) => handleKeyDown(event, index)}
          >
            {item.label}
          </button>
        ))}
      </div>
      {activeItem && (
        <div
          className="st-tabs__panel"
          role="tabpanel"
          id={`${baseId}-panel-${activeItem.id}`}
          aria-labelledby={`${baseId}-tab-${activeItem.id}`}
        >
          {activeItem.content}
        </div>
      )}
    </div>
  );
}

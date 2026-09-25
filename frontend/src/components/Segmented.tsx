interface Option<T extends string> {
  value: T;
  label: string;
}

interface Props<T extends string> {
  options: Option<T>[];
  value: T;
  onChange: (value: T) => void;
  ariaLabel: string;
}

/** Filtro segmentado: el activo lleva fondo menta y texto oscuro (RF-F-05, RF-F-06). */
export function Segmented<T extends string>({ options, value, onChange, ariaLabel }: Props<T>) {
  return (
    <div role="group" aria-label={ariaLabel} className="inline-flex h-9 items-center gap-1 rounded border border-line bg-panel p-1">
      {options.map((o) => {
        const active = o.value === value;
        return (
          <button
            key={o.value}
            type="button"
            aria-pressed={active}
            onClick={() => onChange(o.value)}
            className={`h-7 rounded px-3 text-xs font-semibold transition-colors duration-150 ${
              active ? 'bg-mint text-app' : 'text-ink-2 hover:bg-hover hover:text-ink'
            }`}
          >
            {o.label}
          </button>
        );
      })}
    </div>
  );
}

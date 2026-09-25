import type { ReactNode } from 'react';

interface Props {
  title?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  bodyClassName?: string;
  titleAs?: 'h2' | 'h3' | 'div';
}

export function Panel({ title, actions, children, className = '', bodyClassName = '', titleAs = 'h2' }: Props) {
  const Title = titleAs;
  return (
    <section className={`panel flex flex-col ${className}`}>
      {(title || actions) && (
        <header className="flex min-h-[44px] items-center justify-between gap-3 border-b border-line px-4 py-2">
          {title && <Title className="panel-title">{title}</Title>}
          {actions && <div className="flex items-center gap-2">{actions}</div>}
        </header>
      )}
      <div className={`flex-1 ${bodyClassName}`}>{children}</div>
    </section>
  );
}

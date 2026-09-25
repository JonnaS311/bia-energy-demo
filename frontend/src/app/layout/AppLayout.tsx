import { useEffect, useState, type ReactNode } from 'react';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { AnalysisModal } from '@/components/AnalysisModal';

const MIN_WIDTH = 1280;

function useNarrowViewport(): boolean {
  const [narrow, setNarrow] = useState(() => typeof window !== 'undefined' && window.innerWidth < MIN_WIDTH);
  useEffect(() => {
    const onResize = () => setNarrow(window.innerWidth < MIN_WIDTH);
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, []);
  return narrow;
}

export function AppLayout({ title, children }: { title: string; children: ReactNode }) {
  const narrow = useNarrowViewport();
  useEffect(() => {
    document.title = `${title} · Energy Management`;
  }, [title]);
  return (
    <div className="min-h-screen bg-canvas">
      {narrow && (
        <div className="fixed inset-x-0 top-0 z-50 bg-amber px-4 py-1.5 text-center text-xs font-medium text-app" role="status">
          Esta aplicación está diseñada para pantallas de al menos 1280 px de ancho
        </div>
      )}
      <Sidebar />
      <div className="pl-[240px]">
        <Header title={title} />
        <main className="mx-auto max-w-content p-8">{children}</main>
      </div>
      <AnalysisModal />
    </div>
  );
}

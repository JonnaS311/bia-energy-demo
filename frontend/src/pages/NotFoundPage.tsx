import { Link } from 'react-router-dom';
import { AppLayout } from '@/app/layout/AppLayout';

export function NotFoundPage() {
  return (
    <AppLayout title="Página no encontrada">
      <div className="panel flex flex-col items-center gap-3 px-6 py-16 text-center">
        <p className="text-md font-semibold text-ink">Página no encontrada</p>
        <Link to="/" className="text-sm font-semibold text-mint hover:text-mint-bright">
          Volver al dashboard
        </Link>
      </div>
    </AppLayout>
  );
}

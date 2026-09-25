// RF-F-14 — ErrorState.
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ApiError } from '@/api/client';
import { ErrorState } from './States';

describe('ErrorState', () => {
  it('error de red muestra el texto de conexión', () => {
    render(<ErrorState error={new ApiError(0, 'NETWORK_ERROR', 'x')} text="No se pudieron cargar los medidores" onRetry={() => {}} />);
    expect(screen.getByText('No se pudo conectar con el servidor')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Reintentar' })).toBeInTheDocument();
  });

  it('UNAUTHORIZED no renderiza nada', () => {
    const { container } = render(<ErrorState error={new ApiError(401, 'UNAUTHORIZED', 'x')} text="No se pudieron cargar los medidores" />);
    expect(container).toBeEmptyDOMElement();
  });

  it('otro error muestra el texto de la página', () => {
    render(<ErrorState error={new ApiError(500, 'INTERNAL_ERROR', 'x')} text="No se pudieron cargar los medidores" />);
    expect(screen.getByText('No se pudieron cargar los medidores')).toBeInTheDocument();
  });
});

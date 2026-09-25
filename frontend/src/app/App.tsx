import { BrowserRouter } from 'react-router-dom';
import { RootProviders } from './providers';
import { AppRoutes } from './router';

export function App() {
  return (
    <RootProviders>
      <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <AppRoutes />
      </BrowserRouter>
    </RootProviders>
  );
}

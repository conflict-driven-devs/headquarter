import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../lib/api';
import { useAuth } from '../lib/auth';
import { Field, ErrorAlert } from '../lib/components/ui';

interface SetupStatusResponse {
  state: 'no_db' | 'no_admin' | 'ready';
}

export default function Setup() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();
  const { login } = useAuth();

  useEffect(() => {
    api.get<SetupStatusResponse>('/api/setup/status')
      .then((data) => {
        if (data.state === 'ready') {
          navigate('/', { replace: true });
        }
      })
      .catch(() => setError('Errore di connessione al server'));
  }, [navigate]);

  const handleAdminSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    const formData = new FormData(e.currentTarget);
    const data = {
      name: formData.get('name'),
      surname: formData.get('surname'),
      email: formData.get('email'),
      password: formData.get('password'),
    };

    try {
      await api.post('/api/setup/admin', data);
      await login(String(data.email || ''), String(data.password || ''));
      navigate('/', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Errore sconosciuto');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="relative min-h-screen w-full bg-[#050505] flex items-center justify-center overflow-hidden font-sans">
      <div className="absolute inset-0 z-0 opacity-[0.03] bg-grid" />
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-125 h-125 bg-[#FF6900]/10 blur-[100px] rounded-full pointer-events-none" />

      <div className="relative z-10 w-full max-w-2xl mx-4 p-8 md:p-10 bg-[#0A0A0A]/80 backdrop-blur-2xl saturate-150 border border-white/10 rounded-2xl shadow-2xl shadow-black/80 flex flex-col gap-6">
        <div className="text-center space-y-2">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-linear-to-br from-[#FF6900] to-[#FF4422] shadow-lg shadow-[#FF6900]/20 mb-4">
            <svg className="w-6 h-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </div>
          <h2 className="text-2xl font-bold text-white tracking-tight">
            Creazione Amministratore
          </h2>
          <p className="text-sm text-white/40">
            Crea l'account amministratore iniziale
          </p>
        </div>

        <div className="flex items-center justify-center gap-2">
          <div className="w-2 h-2 rounded-full bg-[#FF6900]" />
        </div>

        {error && <ErrorAlert message={error} />}

        <form onSubmit={handleAdminSubmit} className="flex flex-col gap-4">
          <div className="grid grid-cols-2 gap-4">
            <Field label="Nome">
              <input type="text" name="name" required className="w-full rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none placeholder:text-white/30 focus:border-[#FF6900]" placeholder="Mario" />
            </Field>
            <Field label="Cognome">
              <input type="text" name="surname" required className="w-full rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none placeholder:text-white/30 focus:border-[#FF6900]" placeholder="Rossi" />
            </Field>
          </div>

          <Field label="Email">
            <input type="email" name="email" required className="w-full rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none placeholder:text-white/30 focus:border-[#FF6900]" placeholder="admin@example.com" />
          </Field>

          <Field label="Password (min 8 caratteri)">
            <input type="password" name="password" required minLength={8} className="w-full rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none placeholder:text-white/30 focus:border-[#FF6900]" placeholder="••••••••" />
          </Field>

          <button type="submit" disabled={loading} className="relative w-full py-3 mt-2 px-6 bg-linear-to-r from-[#FF6900] to-[#FF4422] text-white font-semibold rounded-xl shadow-lg shadow-[#FF6900]/25 hover:shadow-xl hover:shadow-[#FF6900]/40 focus:ring-2 focus:ring-[#FF6900] focus:ring-offset-2 focus:ring-offset-[#0A0A0A] disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200">
            {loading ? 'Creazione...' : 'Crea Amministratore'}
          </button>
        </form>
      </div>
    </div>
  );
}

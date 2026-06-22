import { useState } from 'react'
import { Activity, AlertTriangle, CheckCircle2, RefreshCw, Stethoscope } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useDoctor } from '@/queries'
import type { DoctorEmbeddings, DoctorReport } from '@/api/types'

function pct(n: number) {
  return `${Math.round(n * 100)}%`
}

function Stat({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="flex flex-col rounded-lg border border-white/5 bg-white/[0.02] px-2.5 py-1.5">
      <span className="text-[10px] font-semibold uppercase tracking-wider text-slate-500">{label}</span>
      <span className="text-sm font-bold text-white tabular-nums">{value}</span>
    </div>
  )
}

function IssueRow({ icon, label, count }: { icon: string; label: string; count: number }) {
  return (
    <div className="flex items-center justify-between rounded-md bg-rose-500/5 px-2 py-1 text-[11px] text-rose-200/90">
      <span className="flex items-center gap-1.5">
        <span aria-hidden>{icon}</span>
        {label}
      </span>
      <span className="font-bold tabular-nums">{count}</span>
    </div>
  )
}

function StatusPill({ ok, label, detail }: { ok: boolean; label: string; detail?: string }) {
  return (
    <div
      className={cn(
        'flex items-center justify-between rounded-md px-2 py-1 text-[11px]',
        ok
          ? 'bg-emerald-500/5 text-emerald-200/90'
          : 'bg-rose-500/10 text-rose-200/90',
      )}
      title={detail}
    >
      <span className="flex items-center gap-1.5">
        <span aria-hidden>{ok ? '✓' : '✗'}</span>
        {label}
      </span>
      {detail && <span className="truncate pl-2 text-[10px] opacity-70">{detail}</span>}
    </div>
  )
}

function EmbeddingsBlock({ emb }: { emb: DoctorEmbeddings }) {
  // Backend healthy = reachable AND model available. Coverage gaps are a
  // separate, non-fatal signal shown via the existing bar.
  const backendOk = emb.reachable && emb.model_available
  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center justify-between text-[11px] text-slate-400">
        <span className="inline-flex items-center gap-1">
          <Activity className="h-3 w-3" /> Embeddings
        </span>
        <span className="truncate pl-2 text-[10px] text-slate-500" title={`${emb.provider}/${emb.model}`}>
          {emb.provider}/{emb.model}
        </span>
      </div>

      <StatusPill
        ok={emb.reachable}
        label="Ollama server"
        detail={emb.server_url ?? (emb.reachable ? 'reachable' : 'unreachable')}
      />
      <StatusPill
        ok={emb.model_available}
        label="Model pulled"
        detail={
          emb.model_available
            ? emb.model
            : emb.server_error || `run \`ollama pull ${emb.model}\``
        }
      />

      {/* Coverage bar — only meaningful when backend is healthy */}
      {backendOk && (
        <>
          <div className="mt-0.5 flex items-center justify-between text-[11px] text-slate-400">
            <span>Coverage</span>
            <span className="tabular-nums">
              {emb.embedded}/{emb.embedded + emb.missing}
            </span>
          </div>
          <div className="h-1.5 overflow-hidden rounded-full bg-white/5">
            <div
              className={cn(
                'h-full rounded-full transition-all',
                emb.coverage >= 0.99
                  ? 'bg-emerald-400'
                  : emb.coverage >= 0.5
                    ? 'bg-amber-400'
                    : 'bg-rose-400',
              )}
              style={{ width: pct(emb.coverage) }}
            />
          </div>
        </>
      )}

      {/* List available models when server is up but configured model missing */}
      {!emb.model_available &&
        emb.reachable &&
        emb.available_models &&
        emb.available_models.length > 0 && (
          <div className="rounded-md bg-white/[0.02] px-2 py-1 text-[10px] text-slate-500">
            pulled: {emb.available_models.join(', ')}
          </div>
        )}
    </div>
  )
}

function Body({ report, loading, deep, onToggleDeep, onRefresh }: {
  report: DoctorReport | undefined
  loading: boolean
  deep: boolean
  onToggleDeep: () => void
  onRefresh: () => void
}) {
  if (loading && !report) {
    return (
      <div className="flex items-center gap-2 py-3 text-xs text-slate-500">
        <RefreshCw className="h-3 w-3 animate-spin" /> Running doctor…
      </div>
    )
  }
  if (!report) {
    return <div className="py-2 text-xs text-slate-500">No report yet.</div>
  }

  const issueCount =
    (report.invalid_files?.length ?? 0) +
    (report.stale_index_ids?.length ?? 0) +
    (report.missing_index_paths?.length ?? 0)

  const healthy = report.ok

  return (
    <div className="flex flex-col gap-2.5">
      <div className="flex items-center justify-between">
        <span
          className={cn(
            'inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-bold uppercase tracking-wide',
            healthy
              ? 'bg-emerald-400/10 text-emerald-300'
              : 'bg-rose-400/10 text-rose-300',
          )}
        >
          {healthy ? <CheckCircle2 className="h-3 w-3" /> : <AlertTriangle className="h-3 w-3" />}
          {healthy ? 'Healthy' : 'Issues found'}
        </span>
        <button
          onClick={onRefresh}
          disabled={loading}
          title="Re-run doctor"
          className="rounded-md p-1 text-slate-500 transition-colors hover:bg-white/5 hover:text-white disabled:opacity-40"
        >
          <RefreshCw className={cn('h-3.5 w-3.5', loading && 'animate-spin')} />
        </button>
      </div>

      <div className="grid grid-cols-2 gap-1.5">
        <Stat label="Domains" value={report.domains} />
        <Stat label="Memories" value={report.memories} />
      </div>

      {deep && ((report.vault_memories ?? 0) > 0 || (report.index_memories ?? 0) > 0) && (
        <div className="grid grid-cols-2 gap-1.5">
          <Stat label="Vault" value={report.vault_memories ?? 0} />
          <Stat label="Index" value={report.index_memories ?? 0} />
        </div>
      )}

      {deep && issueCount > 0 && (
        <div className="flex flex-col gap-1">
          {report.invalid_files && report.invalid_files.length > 0 && (
            <IssueRow icon="⚠" label="Invalid files" count={report.invalid_files.length} />
          )}
          {report.stale_index_ids && report.stale_index_ids.length > 0 && (
            <IssueRow icon="↺" label="Stale index rows" count={report.stale_index_ids.length} />
          )}
          {report.missing_index_paths && report.missing_index_paths.length > 0 && (
            <IssueRow icon="∅" label="Missing vault files" count={report.missing_index_paths.length} />
          )}
        </div>
      )}

      {report.embeddings && <EmbeddingsBlock emb={report.embeddings} />}

      {report.errors && report.errors.length > 0 && (
        <ul className="flex flex-col gap-0.5">
          {report.errors.slice(0, 3).map((e, i) => (
            <li key={i} className="truncate text-[11px] text-rose-300/80" title={e}>
              • {e}
            </li>
          ))}
          {report.errors.length > 3 && (
            <li className="text-[11px] text-slate-500">+{report.errors.length - 3} more</li>
          )}
        </ul>
      )}

      <button
        onClick={onToggleDeep}
        className={cn(
          'mt-0.5 rounded-md border px-2 py-1 text-[11px] font-semibold transition-colors',
          deep
            ? 'border-sky-400/30 bg-sky-400/10 text-sky-200'
            : 'border-white/10 bg-white/[0.02] text-slate-400 hover:border-white/20 hover:text-white',
        )}
      >
        {deep ? 'Deep audit on' : 'Run deep audit'}
      </button>
    </div>
  )
}

export function DoctorPanel() {
  const [deep, setDeep] = useState(false)
  const query = useDoctor({ deep })

  return (
    <section className="rounded-2xl border border-white/10 bg-black/20 p-3">
      <header className="mb-2.5 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
        <Stethoscope className="h-3.5 w-3.5" /> Doctor
      </header>
      <Body
        report={query.data}
        loading={query.isFetching}
        deep={deep}
        onToggleDeep={() => setDeep(d => !d)}
        onRefresh={() => query.refetch()}
      />
    </section>
  )
}

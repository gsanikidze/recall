import { Search } from 'lucide-react'

interface Props {
  value: string
  onChange: (v: string) => void
  placeholder?: string
}

export function SearchBar({ value, onChange, placeholder = 'Search memories…' }: Props) {
  return (
    <div className="relative flex items-center">
      <Search className="absolute left-3.5 h-4 w-4 text-sky-200/45" />
      <input
        type="text"
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full rounded-2xl border border-white/10 bg-black/25 py-2.5 pl-10 pr-3 text-sm text-white/85 shadow-inner shadow-black/20 placeholder:text-slate-500 focus:border-sky-400/50 focus:outline-none focus:ring-2 focus:ring-sky-400/15"
      />
    </div>
  )
}

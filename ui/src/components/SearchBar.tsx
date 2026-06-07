import { Search } from 'lucide-react'

interface Props {
  value: string
  onChange: (v: string) => void
  placeholder?: string
}

export function SearchBar({ value, onChange, placeholder = 'Search memories…' }: Props) {
  return (
    <div className="relative flex items-center">
      <Search className="absolute left-3 w-4 h-4 text-white/30" />
      <input
        type="text"
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full bg-white/5 border border-white/10 rounded-lg pl-9 pr-3 py-2 text-sm text-white/80 placeholder:text-white/30 focus:outline-none focus:border-violet-500/50"
      />
    </div>
  )
}

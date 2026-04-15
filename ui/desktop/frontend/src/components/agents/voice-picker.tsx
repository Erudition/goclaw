import { useTranslation } from 'react-i18next'
import { useVoices, useRefreshVoices } from '../../hooks/use-voices'
import { VoicePreviewButton } from './voice-preview-button'
import { Combobox } from '../common/Combobox'
import type { Voice } from '../../services/voices'

interface Props {
  value: string | null
  onChange: (voiceId: string) => void
  disabled?: boolean
}

function VoiceOption({ voice, selected }: { voice: Voice; selected: boolean }) {
  const labelEntries = ['gender', 'accent', 'age', 'use_case']
    .filter((k) => voice.labels?.[k])
    .map((k) => voice.labels![k])

  return (
    <span className={['flex items-center gap-1 w-full', selected ? 'text-accent' : ''].join(' ')}>
      <span className="flex-1 truncate" title={voice.name}>{voice.name}</span>
      {labelEntries.slice(0, 1).map((label) => (
        <span key={label} className="text-[10px] px-1 py-0.5 rounded bg-surface-tertiary text-text-muted shrink-0">
          {label}
        </span>
      ))}
      <VoicePreviewButton previewUrl={voice.preview_url} voiceName={voice.name} />
    </span>
  )
}

export function VoicePicker({ value, onChange, disabled }: Props) {
  const { t } = useTranslation('tts')
  const { data: voices, isLoading } = useVoices()
  const { mutate: refresh, isPending: refreshing } = useRefreshVoices()

  const options = voices.map((v) => ({ value: v.voice_id, label: v.name }))

  return (
    <div className="space-y-1">
      <div className="flex items-center gap-2">
        <div className="flex-1 min-w-0">
          <Combobox
            value={value ?? ''}
            onChange={onChange}
            options={options}
            placeholder={isLoading ? t('voice_loading') : t('voice_placeholder')}
            loading={isLoading}
            allowCustom={false}
            disabled={disabled}
          />
        </div>
        <button
          type="button"
          title={t('voice_refresh')}
          disabled={refreshing || isLoading}
          onClick={() => refresh()}
          className="shrink-0 p-1.5 rounded hover:bg-surface-tertiary transition-colors text-text-muted hover:text-text-primary disabled:opacity-50"
          aria-label={t('voice_refresh')}
        >
          <svg
            className={['w-3.5 h-3.5', refreshing ? 'animate-spin' : ''].join(' ')}
            viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}
            strokeLinecap="round" strokeLinejoin="round"
          >
            <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" />
            <path d="M21 3v5h-5" />
            <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" />
            <path d="M3 21v-5h5" />
          </svg>
        </button>
      </div>

      {/* Preview for selected voice */}
      {value && voices.find((v) => v.voice_id === value) && (
        <div className="flex items-center gap-1 text-[11px] text-text-muted">
          <VoicePreviewButton
            previewUrl={voices.find((v) => v.voice_id === value)?.preview_url}
            voiceName={voices.find((v) => v.voice_id === value)?.name ?? ''}
          />
          <span className="truncate">{voices.find((v) => v.voice_id === value)?.name}</span>
        </div>
      )}

      {!isLoading && voices.length === 0 && (
        <p className="text-[11px] text-text-muted">{t('voice_no_voices')}</p>
      )}
    </div>
  )
}

// Export VoiceOption for potential reuse
export { VoiceOption }

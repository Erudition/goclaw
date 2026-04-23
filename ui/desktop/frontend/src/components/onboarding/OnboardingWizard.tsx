import { useState, useEffect } from 'react'
import { useBootstrapStatus, type SetupStep } from '../../hooks/use-bootstrap-status'
import { SetupStepper } from './SetupStepper'
import { ProviderStep } from './ProviderStep'
import { ModelVerifyStep } from './ModelVerifyStep'
import { AgentStep } from './AgentStep'
import type { ProviderData } from '../../types/provider'

interface OnboardingWizardProps {
  onComplete: () => void
}

export function OnboardingWizard({ onComplete }: OnboardingWizardProps) {
  const { currentStep, loading, providers } = useBootstrapStatus()
  const [step, setStep] = useState<1 | 2 | 3>(1)
  const [createdProvider, setCreatedProvider] = useState<ProviderData | null>(null)
  const [selectedModel, setSelectedModel] = useState<string | null>(null)
  const [initialized, setInitialized] = useState(false)

  // Initialize step from server state (only on first load)
  useEffect(() => {
    if (loading || initialized) return
    if (currentStep === ('complete' as SetupStep)) {
      onComplete()
      return
    }
    setStep(currentStep as 1 | 2 | 3)
    setInitialized(true)
  }, [currentStep, loading, initialized, onComplete])

  if (loading || !initialized) {
    return (
      <div className="h-dvh flex items-center justify-center bg-surface-primary">
        <div className="w-6 h-6 border-2 border-accent border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  const completedSteps: number[] = []
  if (step > 1) completedSteps.push(1)
  if (step > 2) completedSteps.push(2)

  // Resume: find existing provider from server data (any enabled provider)
  const activeProvider = createdProvider ?? providers.find((p) => p.enabled) ?? providers[0] ?? null

  return (
    <div className="h-dvh flex items-center justify-center bg-surface-primary px-4 py-8">
      <div className="w-full max-w-2xl space-y-6">
        {/* Header */}
        <div className="text-center">
          <img src="/goclaw-icon.svg" alt="GoClaw" className="mx-auto mb-4 h-16 w-16" />
          <h1 className="text-4xl font-bold tracking-tight text-text-primary">GoClaw Setup</h1>
          <p className="mt-2 text-sm text-text-muted">
            Let's get your gateway up and running
          </p>
        </div>

        <SetupStepper currentStep={step} completedSteps={completedSteps} />

        {step === 1 && (
          <ProviderStep
            existingProvider={activeProvider}
            onComplete={(provider) => {
              setCreatedProvider(provider)
              setStep(2)
            }}
          />
        )}

        {step === 2 && activeProvider && (
          <ModelVerifyStep
            provider={activeProvider}
            initialModel={selectedModel}
            onBack={() => setStep(1)}
            onComplete={(model) => {
              setSelectedModel(model)
              setStep(3)
            }}
          />
        )}

        {step === 3 && activeProvider && (
          <AgentStep
            provider={activeProvider}
            model={selectedModel}
            onBack={() => setStep(2)}
            onComplete={onComplete}
          />
        )}

      </div>
    </div>
  )
}

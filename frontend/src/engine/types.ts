export interface RuleEngineContext {
  getValue: (ctrlId: string) => any
  setValue: (ctrlId: string, value: any) => void
  getControl: (ctrlId: string) => { id: string; type: string } | undefined
  hiddenFields: Set<string>
  disabledFields: Set<string>
}
